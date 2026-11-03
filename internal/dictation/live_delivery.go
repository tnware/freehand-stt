package dictation

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/realtime"
)

// liveDelivery has one worker and one coalescing mailbox, not a goroutine or
// queue per caption. Its context belongs exclusively to one recording run.
type liveDelivery struct {
	mu                  sync.Mutex
	reportMu            sync.Mutex
	ctx                 context.Context
	cancel              context.CancelFunc
	policy              insertion.Policy
	target              insertion.Target
	wake, finish, done  chan struct{}
	once                sync.Once
	latest              realtime.Update
	final, posted       string
	attempted, disabled bool
	err                 error
	notify              func(string)
}
type liveDeliveryResult struct {
	final, posted string
	attempted     bool
	err           error
}

func liveInsertEnabled(cfg config.Settings) bool {
	return cfg.LiveInsert && cfg.AutoInsert && !cfg.PostProcessing.Enabled && cfg.VoiceTranscription.Realtime && cfg.VoiceTranscription.CompatibilityProfile == compatibility.NeMoSpeechV1 && cfg.VoiceTranscription.ModelProfile == modelprofile.Nemotron35
}
func newLiveDelivery(ctx context.Context, policy insertion.Policy, target insertion.Target, notify func(string)) *liveDelivery {
	ctx, cancel := context.WithCancel(ctx)
	d := &liveDelivery{ctx: ctx, cancel: cancel, policy: policy, target: target, notify: notify, wake: make(chan struct{}, 1), finish: make(chan struct{}), done: make(chan struct{})}
	go d.run()
	return d
}
func (d *liveDelivery) offer(update realtime.Update) {
	if d == nil || !update.Finalized {
		return
	}
	d.mu.Lock()
	previous := d.final
	if strings.HasPrefix(update.Final, previous) {
		d.final = update.Final
	}
	if update.EventID == "" || !strings.HasPrefix(update.Final, previous) {
		d.disabled = true
		if d.attempted {
			d.err = insertion.ErrCopyRequired
			d.cancel()
		}
	}
	d.latest = update
	d.mu.Unlock()
	d.report()
	select {
	case d.wake <- struct{}{}:
	default:
	}
}
func (d *liveDelivery) abort() {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.disabled = true
	d.err = insertion.ErrCopyRequired
	d.cancel()
	d.mu.Unlock()
	d.report()
}
func (d *liveDelivery) report() {
	d.reportMu.Lock()
	defer d.reportMu.Unlock()
	if d.notify == nil {
		return
	}
	d.mu.Lock()
	message := "Recording; live typing finalized phrases"
	if d.disabled {
		message = "Recording; text will be typed after stopping"
		if d.attempted {
			message = "Live typing stopped; recording continues. Check the target before copying the final transcript."
		}
	}
	d.mu.Unlock()
	d.notify(message)
}
func (d *liveDelivery) wait() liveDeliveryResult {
	if d == nil {
		return liveDeliveryResult{}
	}
	d.once.Do(func() { close(d.finish) })
	<-d.done
	d.mu.Lock()
	defer d.mu.Unlock()
	return liveDeliveryResult{final: d.final, posted: d.posted, attempted: d.attempted, err: d.err}
}
func (d *liveDelivery) run() {
	defer close(d.done)
	defer d.cancel()
	var session *insertion.Session
	defer func() {
		if session != nil {
			session.Close()
		}
	}()
	deliver := func() {
		defer d.report()
		d.mu.Lock()
		update := d.latest
		if d.disabled || d.err != nil || update.Final == d.posted || d.ctx.Err() != nil {
			d.mu.Unlock()
			return
		}
		if !strings.HasPrefix(update.Final, d.posted) {
			d.err = insertion.ErrCopyRequired
			d.disabled = true
			d.mu.Unlock()
			return
		}
		suffix := update.Final[len(d.posted):]
		// Adopt and reserve the attempt atomically with mailbox eligibility.
		// BeginSession performs no native validation for a valid target.
		if session == nil {
			var err error
			session, err = d.policy.BeginSession(d.ctx, d.target, insertion.DirectInput)
			if err != nil {
				d.disabled = true
				d.mu.Unlock()
				return
			}
		}
		d.attempted = true // Consume before native work, including an ambiguous failure.
		d.mu.Unlock()
		sendCtx, cancel := context.WithTimeout(d.ctx, 3*time.Second)
		err := session.Deliver(sendCtx, update.Turn, suffix)
		cancel()
		d.mu.Lock()
		if err != nil {
			d.err = insertion.CopyRequired(err)
			d.disabled = true
		} else {
			d.posted = update.Final
		}
		d.mu.Unlock()
	}
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.finish:
			deliver()
			return
		case <-d.wake:
			deliver()
		}
	}
}

func liveRecoveryMessage(message string) string {
	return message + " Text already typed has not been removed. Copy the transcript if needed, and check the target for text already inserted before pasting."
}
