//go:build windows

package overlay

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/tnware/freehand-stt/internal/audio"
	"golang.org/x/sys/windows"
)

type overlayAnimation struct {
	shown            bool
	hiding           bool
	showAt           time.Time
	hideAt           time.Time
	morphAt          time.Time
	target           overlayView
	from             overlayView
	lastView         overlayView
	workArea         nativeRect
	anchorValid      bool
	anchorGeneration uint64
	dpi              uint32
}

// StatusOverlay owns one native Win32 tool window on a dedicated message-loop
// thread. The window is presentation-only and never owns focus or insertion.
type StatusOverlay struct {
	mu        sync.RWMutex
	view      overlayView
	options   OverlayOptions
	hwnd      atomic.Uintptr
	threadID  atomic.Uint32
	closed    atomic.Bool
	closeOnce sync.Once
	done      chan struct{}

	surface           overlaySurface
	anim              overlayAnimation
	animationsEnabled bool
	highContrast      bool
	fontFamily        uintptr
	fonts             map[overlayFontKey]uintptr
	captionCache      overlayCaptionCache

	// Live capture amplitude. The source is set once during composition; the
	// ring, envelope and scratch buffer belong to the message-loop thread.
	source   atomic.Value
	levels   *audio.LevelRing
	envelope audio.Envelope
	scratch  []float64
	sampled  time.Time
}

// SetLevelSource attaches capture amplitude to the recording meter. Without
// one the meter animates from the clock, so the overlay degrades to its
// previous behaviour rather than going flat.
func (o *StatusOverlay) SetLevelSource(source LevelSource) {
	if o == nil || source == nil {
		return
	}
	o.source.Store(source)
}

func (o *StatusOverlay) levelSource() LevelSource {
	if o == nil {
		return nil
	}
	source, _ := o.source.Load().(LevelSource)
	return source
}

// sampleLevels advances the meter. Readings are taken on their own cadence
// rather than once per frame, so the history scrolls at a fixed rate whatever
// the paint rate is.
func (o *StatusOverlay) sampleLevels(now time.Time, recording bool) {
	source := o.levelSource()
	if source == nil {
		return
	}
	if o.levels == nil {
		o.levels = audio.NewLevelRing(overlayLevelBars)
		o.envelope = audio.NewEnvelope()
	}
	if !recording {
		// Draining keeps a peak captured mid-transition from appearing at the
		// head of the next recording.
		source.TakeLevel()
		o.levels.Reset()
		o.envelope.Reset()
		o.sampled = time.Time{}
		return
	}
	if !o.sampled.IsZero() && now.Sub(o.sampled) < overlayLevelMS*time.Millisecond {
		return
	}
	o.sampled = now
	// Decibels first, so the envelope smooths what the eye will see rather
	// than raw amplitude.
	o.levels.Push(o.envelope.Push(audio.NormalizeLevel(source.TakeLevel())))
}

// liveLevels returns the meter history, or nil when the meter is not driving
// this frame.
func (o *StatusOverlay) liveLevels(recording bool) []float64 {
	if !recording || o.levels == nil || o.levelSource() == nil {
		return nil
	}
	o.scratch = o.levels.Snapshot(o.scratch)
	return o.scratch
}

func NewStatusOverlay() (*StatusOverlay, error) {
	o := &StatusOverlay{
		done:              make(chan struct{}),
		options:           DefaultOverlayOptions(),
		animationsEnabled: clientAreaAnimationsEnabled(),
		highContrast:      highContrastEnabled(),
		fonts:             make(map[overlayFontKey]uintptr),
	}
	ready := make(chan error, 1)
	go o.loop(ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return o, nil
}

// clientAreaAnimationsEnabled follows the Windows Animation Effects setting.
// If the preference cannot be read, retain the existing animated behaviour.
func clientAreaAnimationsEnabled() bool {
	enabled := int32(1)
	result, _, _ := systemParametersInfo.Call(
		spiGetClientAreaAnimation,
		0,
		uintptr(unsafe.Pointer(&enabled)),
		0,
	)
	return result == 0 || enabled != 0
}

func highContrastEnabled() bool {
	info := highContrastInfo{Size: uint32(unsafe.Sizeof(highContrastInfo{}))}
	result, _, _ := systemParametersInfo.Call(
		spiGetHighContrast,
		uintptr(info.Size),
		uintptr(unsafe.Pointer(&info)),
		0,
	)
	return result != 0 && info.Flags&hcfHighContrastOn != 0
}

// Configure updates independent presentation options on the native message
// loop. It never recreates or directly manipulates the HWND from the caller's
// thread.
func (o *StatusOverlay) Configure(options OverlayOptions) error {
	if o == nil || o.closed.Load() {
		return errors.New("native status overlay is closed")
	}
	o.mu.Lock()
	o.options = normalizeOverlayOptions(options)
	o.mu.Unlock()
	hwnd := o.hwnd.Load()
	if hwnd == 0 {
		return errors.New("native status overlay window is unavailable")
	}
	result, _, callErr := postMessage.Call(hwnd, wmOverlayOptions, 0, 0)
	if result == 0 {
		return callFailure("native status overlay options could not be queued", callErr)
	}
	return nil
}

func (o *StatusOverlay) Update(status OverlayStatus) error {
	if o == nil || o.closed.Load() {
		return errors.New("native status overlay is closed")
	}
	o.mu.Lock()
	o.view = resolveOverlayView(status)
	o.mu.Unlock()
	hwnd := o.hwnd.Load()
	if hwnd == 0 {
		return errors.New("native status overlay window is unavailable")
	}
	result, _, callErr := postMessage.Call(hwnd, wmOverlayApply, 0, 0)
	if result == 0 {
		return callFailure("native status overlay update could not be queued", callErr)
	}
	return nil
}

func (o *StatusOverlay) Close() error {
	if o == nil {
		return nil
	}
	var closeErr error
	o.closeOnce.Do(func() {
		o.closed.Store(true)
		hwnd := o.hwnd.Load()
		if hwnd != 0 {
			if result, _, callErr := postMessage.Call(hwnd, wmClose, 0, 0); result == 0 {
				closeErr = callFailure("native status overlay close could not be queued", callErr)
				if threadID := o.threadID.Load(); threadID != 0 {
					postThreadMessage.Call(uintptr(threadID), 0x0012, 0, 0)
				}
			}
		}
		<-o.done
	})
	return closeErr
}

func (o *StatusOverlay) snapshot() overlayView {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.view
}

func (o *StatusOverlay) optionSnapshot() OverlayOptions {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.options.Scale == 0 {
		return DefaultOverlayOptions()
	}
	return o.options
}

func (o *StatusOverlay) loop(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := getCurrentThreadID.Call()
	o.threadID.Store(uint32(threadID))
	if err := gdiplusInit(); err != nil {
		ready <- err
		close(o.done)
		return
	}
	instance, _, instanceErr := getModuleHandle.Call(0)
	if instance == 0 {
		ready <- callFailure("native status overlay module handle is unavailable", instanceErr)
		close(o.done)
		return
	}
	class := overlayWindowClass{
		Size:       uint32(unsafe.Sizeof(overlayWindowClass{})),
		WindowProc: overlayWindowProc,
		Instance:   instance,
		ClassName:  overlayClassName,
	}
	if result, _, classErr := registerClassEx.Call(uintptr(unsafe.Pointer(&class))); result == 0 {
		ready <- callFailure("native status overlay window class could not be registered", classErr)
		close(o.done)
		return
	}

	title := windows.StringToUTF16Ptr("Freehand status")
	hwnd, _, createErr := createWindowEx.Call(
		nativeOverlayExtendedStyle(),
		uintptr(unsafe.Pointer(overlayClassName)),
		uintptr(unsafe.Pointer(title)),
		wsPopup,
		0, 0, 1, 1,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		unregisterClass.Call(uintptr(unsafe.Pointer(overlayClassName)), instance)
		ready <- callFailure("native status overlay window could not be created", createErr)
		close(o.done)
		return
	}
	o.hwnd.Store(hwnd)
	overlayWindows.Store(hwnd, o)
	ready <- nil

	var message nativeMessage
	for {
		result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
	overlayWindows.Delete(hwnd)
	o.hwnd.Store(0)
	unregisterClass.Call(uintptr(unsafe.Pointer(overlayClassName)), instance)
	close(o.done)
}

func statusOverlayWindowProc(hwnd uintptr, message uint32, wparam, lparam uintptr) uintptr {
	switch message {
	case wmOverlayApply:
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).apply(hwnd)
		}
		return 0
	case wmOverlayOptions:
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).reconfigure(hwnd)
		}
		return 0
	case wmTimer:
		if wparam == overlayTimerID {
			if value, ok := overlayWindows.Load(hwnd); ok {
				value.(*StatusOverlay).tick(hwnd)
			}
		}
		return 0
	case wmPaint:
		// A layered window presented through UpdateLayeredWindow is not painted
		// from the update region, but the region still has to be validated or
		// Windows keeps resending WM_PAINT.
		var paint paintStruct
		beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
		endPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).render(hwnd)
		}
		return 0
	case wmDPIChanged:
		if value, ok := overlayWindows.Load(hwnd); ok {
			overlay := value.(*StatusOverlay)
			overlay.resolveAnchor(hwnd)
			overlay.render(hwnd)
		}
		return 0
	case wmSettingChange:
		if value, ok := overlayWindows.Load(hwnd); ok {
			value.(*StatusOverlay).refreshSystemPreferences(hwnd)
		}
		return 0
	case wmEraseBkgnd:
		return 1
	case wmNCHitTest:
		return ^uintptr(0) // HTTRANSPARENT
	case wmMouseActivate:
		return maNoActivate
	case wmClose:
		destroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		killTimer.Call(hwnd, overlayTimerID)
		if value, ok := overlayWindows.Load(hwnd); ok {
			overlay := value.(*StatusOverlay)
			overlay.surface.release()
			overlay.releaseFonts()
		}
		gdiplusRelease()
		postQuitMessage.Call(0)
		return 0
	}
	result, _, _ := defWindowProc.Call(hwnd, uintptr(message), wparam, lparam)
	return result
}

func callFailure(prefix string, err error) error {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return errors.New(prefix)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
