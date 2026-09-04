//go:build windows

package platform

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/gen2brain/malgo"
	"github.com/tnware/freehand-stt/internal/audio"
)

type Capture struct {
	mu         sync.Mutex
	ctx        *malgo.AllocatedContext
	dev        captureDevice
	deviceID   string
	deviceName string
	pcm        []byte
	sink       audio.PCMStreamSink
	captured   int64
	maxBytes   int64
	limit      bool
	active     bool
	closed     atomic.Bool

	// Armed only while an unexpected native stop should become a coordinator
	// interruption. Intentional cleanup disarms before calling into miniaudio,
	// keeping the native callback lock-free and one-shot.
	interruptionArmed atomic.Bool
	deviceLost        atomic.Bool
	callbackFailure   atomic.Pointer[captureFailure]
	session           atomic.Pointer[captureSession]

	// Registered meters. The slice is replaced wholesale rather than mutated,
	// so the audio thread reads it with one atomic load and never locks.
	taps atomic.Pointer[[]*LevelTap]
}

type captureFailure struct{ err error }
type captureSession struct {
	interrupted chan error
}

type captureDevice interface {
	Start() error
	Stop() error
	Uninit()
}

// NewLevelTap registers an independent peak-hold. Taps are created during
// composition and live for the process; each consumer needs its own because
// reading a tap clears it.
func (c *Capture) NewLevelTap() *LevelTap {
	tap := &LevelTap{}
	for {
		current := c.taps.Load()
		next := []*LevelTap{tap}
		if current != nil {
			next = append(append([]*LevelTap{}, *current...), tap)
		}
		if c.taps.CompareAndSwap(current, &next) {
			return tap
		}
	}
}

// publishLevel measures the buffer once and fans the result out. With no taps
// registered it does no work at all, so the meter costs nothing when nothing
// is watching.
func (c *Capture) publishLevel(buffer []byte) {
	taps := c.taps.Load()
	if taps == nil || len(*taps) == 0 {
		return
	}
	level := audio.Peak(buffer)
	for _, tap := range *taps {
		tap.observe(level)
	}
}

func newContext() (*malgo.AllocatedContext, error) {
	return malgo.InitContext([]malgo.Backend{malgo.BackendWasapi}, malgo.ContextConfig{}, nil)
}
func (c *Capture) List(_ context.Context) ([]audio.Device, error) {
	ctx, e := newContext()
	if e != nil {
		return nil, e
	}
	defer func() { _ = ctx.Uninit(); ctx.Free() }()
	ds, e := ctx.Devices(malgo.Capture)
	if e != nil {
		return nil, e
	}
	out := []audio.Device{{ID: "", Name: "System default microphone", Default: true}}
	for i := range ds {
		out = append(out, audio.Device{ID: ds[i].ID.String(), Name: ds[i].Name(), Default: ds[i].IsDefault != 0})
	}
	return out, nil
}

func (c *Capture) DeviceName() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.deviceName
}

func (c *Capture) Prepare(ctx context.Context, id string) error {
	if c.closed.Load() {
		return errors.New("audio capture is closed")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed.Load() {
		return errors.New("audio capture is closed")
	}
	if c.active {
		return nil
	}
	return c.prepareDeviceLocked(id)
}

func (c *Capture) Start(ctx context.Context, id string, seconds int) (<-chan error, error) {
	return c.start(ctx, id, seconds, nil)
}

func (c *Capture) StartStream(ctx context.Context, id string, seconds int, sink audio.PCMStreamSink) (<-chan error, error) {
	if sink == nil {
		return nil, errors.New("PCM stream sink is required")
	}
	return c.start(ctx, id, seconds, sink)
}

func (c *Capture) start(_ context.Context, id string, seconds int, sink audio.PCMStreamSink) (<-chan error, error) {
	if c.closed.Load() {
		return nil, errors.New("audio capture is closed")
	}
	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return nil, errors.New("audio capture is closed")
	}
	if c.active {
		c.mu.Unlock()
		return nil, errors.New("capture already active")
	}
	// A new recording starts from silence rather than inheriting the last peak.
	if taps := c.taps.Load(); taps != nil {
		for _, tap := range *taps {
			tap.TakeLevel()
		}
	}
	var maxBytes int64
	if sink == nil {
		max, e := audio.MaxPCMBytes(seconds)
		if e != nil {
			c.mu.Unlock()
			return nil, e
		}
		maxBytes = int64(max)
	} else {
		if seconds < 1 || seconds > audio.MaxSegmentedRecordingDurationSeconds {
			c.mu.Unlock()
			return nil, errors.New("segmented recording duration is out of range")
		}
		maxBytes = int64(seconds) * audio.SampleRate * 2
	}
	if e := c.prepareDeviceLocked(id); e != nil {
		c.mu.Unlock()
		return nil, e
	}
	c.pcm = nil
	if sink == nil {
		c.pcm = make([]byte, 0, int(maxBytes))
	}
	c.sink = sink
	c.captured = 0
	c.maxBytes = maxBytes
	c.limit = false
	c.interruptionArmed.Store(false)
	c.deviceLost.Store(false)
	c.callbackFailure.Store(nil)
	session := &captureSession{interrupted: make(chan error, 1)}
	c.session.Store(session)
	c.interruptionArmed.Store(true)
	c.active = true
	dev := c.dev
	c.mu.Unlock()
	if e := dev.Start(); e != nil {
		c.interruptionArmed.Store(false)
		c.session.Store(nil)
		c.mu.Lock()
		var staleContext *malgo.AllocatedContext
		c.active = false
		c.sink = nil
		c.pcm = nil
		if c.dev == dev {
			c.dev = nil
			staleContext = c.ctx
			c.ctx = nil
			c.deviceID = ""
			c.deviceName = ""
		}
		c.mu.Unlock()
		dev.Uninit()
		if staleContext != nil {
			_ = staleContext.Uninit()
			staleContext.Free()
		}
		return nil, e
	}
	return session.interrupted, nil
}

// prepareDeviceLocked retains an initialized, stopped device between
// recordings. malgo explicitly supports Stop followed by Start; avoiding
// repeated WASAPI context/device creation keeps the hotkey path warm without
// capturing audio while the app is idle.
func (c *Capture) prepareDeviceLocked(id string) error {
	if c.dev != nil && c.deviceID == id {
		return nil
	}
	if c.dev != nil {
		c.dev.Uninit()
		c.dev = nil
		c.deviceID = ""
		c.deviceName = ""
	}
	if c.ctx == nil {
		ctx, err := newContext()
		if err != nil {
			return err
		}
		c.ctx = ctx
	}
	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.SampleRate = audio.SampleRate
	cfg.Capture.Format = malgo.FormatS16
	cfg.Capture.Channels = 1
	cfg.Capture.ShareMode = malgo.Shared
	name := "System default"
	if id != "" {
		var selected *malgo.DeviceID
		ds, err := c.ctx.Devices(malgo.Capture)
		if err == nil {
			for i := range ds {
				if id == ds[i].ID.String() {
					deviceID := ds[i].ID
					selected = &deviceID
					name = ds[i].Name()
					break
				}
			}
		}
		if selected == nil {
			return errors.New("selected microphone is unavailable")
		}
		cfg.Capture.DeviceID = selected.Pointer()
	}
	dev, err := malgo.InitDevice(c.ctx.Context, cfg, malgo.DeviceCallbacks{
		Data: c.captureData,
		Stop: c.onDeviceStopped,
	})
	if err != nil {
		return err
	}
	if c.closed.Load() {
		dev.Uninit()
		return errors.New("audio capture is closed")
	}
	c.dev = dev
	c.deviceID = id
	c.deviceName = name
	return nil
}

func (c *Capture) captureData(_, input []byte, _ uint32) {
	// Measured before the lock, and on the full buffer: the meter should show
	// what the microphone heard even on the frame the cap truncates.
	c.publishLevel(input)
	c.mu.Lock()
	if !c.active {
		c.mu.Unlock()
		return
	}
	remain := c.maxBytes - c.captured
	if remain > 0 {
		if int64(len(input)) > remain {
			input = input[:int(remain)]
			c.limit = true
		}
		c.captured += int64(len(input))
		if c.sink != nil {
			if !c.sink.WritePCM(input) {
				c.mu.Unlock()
				c.signalInterruption(audio.ErrStreamOverrun, false)
				return
			}
		} else {
			c.pcm = append(c.pcm, input...)
		}
	} else {
		c.limit = true
	}
	c.mu.Unlock()
}

// onDeviceStopped runs from miniaudio's native stop callback. It cannot block,
// tear down the device, or enter the coordinator directly. The per-recording
// buffered channel hands cleanup to ordinary Go control flow exactly once.
func (c *Capture) onDeviceStopped() {
	c.signalInterruption(audio.ErrDeviceInterrupted, true)
}

func (c *Capture) signalInterruption(cause error, deviceLost bool) {
	if !c.interruptionArmed.CompareAndSwap(true, false) {
		return
	}
	session := c.session.Load()
	if session == nil {
		return
	}
	if deviceLost {
		c.deviceLost.Store(true)
	}
	c.callbackFailure.Store(&captureFailure{err: cause})
	select {
	case session.interrupted <- cause:
	default:
	}
}
func (c *Capture) Stop(context.Context) (audio.Result, error) {
	c.mu.Lock()
	if !c.active {
		c.mu.Unlock()
		return audio.Result{}, nil
	}
	dev, sink := c.dev, c.sink
	c.interruptionArmed.Store(false)
	c.session.Store(nil)
	c.active = false
	c.sink = nil
	c.mu.Unlock()
	e := dev.Stop()
	if sink != nil {
		sink.Close()
	}
	c.mu.Lock()
	out := append([]byte(nil), c.pcm...)
	limit := c.limit
	lost := c.deviceLost.Swap(false)
	for i := range c.pcm {
		c.pcm[i] = 0
	}
	c.pcm = nil
	c.captured = 0
	c.maxBytes = 0
	c.mu.Unlock()
	failure := c.callbackFailure.Swap(nil)
	var cleanupErr error
	if lost || failure != nil || e != nil {
		c.mu.Lock()
		var staleContext *malgo.AllocatedContext
		if c.dev == dev {
			c.dev = nil
			staleContext = c.ctx
			c.ctx = nil
			c.deviceID = ""
			c.deviceName = ""
		}
		c.mu.Unlock()
		dev.Uninit()
		if staleContext != nil {
			cleanupErr = staleContext.Uninit()
			staleContext.Free()
		}
	}
	if lost || failure != nil {
		for i := range out {
			out[i] = 0
		}
		if failure != nil {
			return audio.Result{}, errors.Join(failure.err, cleanupErr)
		}
		return audio.Result{}, errors.Join(audio.ErrDeviceInterrupted, cleanupErr)
	}
	if e != nil {
		for i := range out {
			out[i] = 0
		}
		return audio.Result{}, errors.Join(e, cleanupErr)
	}
	return audio.Result{PCM: out, LimitReached: limit}, nil
}
func (c *Capture) Cancel(ctx context.Context) error {
	r, e := c.Stop(ctx)
	for i := range r.PCM {
		r.PCM[i] = 0
	}
	return e
}
func (c *Capture) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	stopErr := c.Cancel(context.Background())
	c.mu.Lock()
	dev, ctx := c.dev, c.ctx
	c.dev = nil
	c.ctx = nil
	c.deviceID = ""
	c.deviceName = ""
	c.mu.Unlock()
	if dev != nil {
		dev.Uninit()
	}
	var contextErr error
	if ctx != nil {
		contextErr = ctx.Uninit()
		ctx.Free()
	}
	return errors.Join(stopErr, contextErr)
}
