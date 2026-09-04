package app

import (
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/platform"
)

// The main-window meter is fed far more slowly than the overlay's. Every
// reading crosses the Wails bridge as a message into WebView2, so this is a
// rate chosen for the cost of the transport rather than for the display. One
// scalar thirty times a second is nothing for the bridge, while a per-frame
// history would be dozens of numbers sixty times a second.
const (
	levelEvent = "dictation:level"
	levelHz    = 30
)

// levelPump carries capture amplitude to the settings window.
//
// It sends one scalar per tick rather than a whole history, and the renderer
// keeps its own ring. That is the cheaper half of the bargain: a history per
// message would be dozens of numbers per tick to say what one number already
// says.
type levelPump struct {
	tap      platform.LevelSource
	emit     func(level float64)
	wanted   func() bool
	envelope audio.Envelope
	quiet    bool
	done     chan struct{}
}

func newLevelPump(tap platform.LevelSource, emit func(float64), wanted func() bool) *levelPump {
	return &levelPump{
		tap:      tap,
		emit:     emit,
		wanted:   wanted,
		envelope: audio.NewEnvelopeAt(levelHz),
		quiet:    true,
		done:     make(chan struct{}),
	}
}

func (p *levelPump) run() {
	ticker := time.NewTicker(time.Second / levelHz)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.tick()
		}
	}
}

func (p *levelPump) tick() {
	if !p.wanted() {
		if p.quiet {
			return
		}
		// Settle the meter once on the way out rather than leaving the last
		// loud reading frozen on screen, then go silent.
		p.quiet = true
		p.envelope.Reset()
		p.tap.TakeLevel()
		p.emit(0)
		return
	}
	p.quiet = false
	p.emit(p.envelope.Push(audio.NormalizeLevel(p.tap.TakeLevel())))
}

func (p *levelPump) stop() {
	select {
	case <-p.done:
	default:
		close(p.done)
	}
}

// levelsWanted reports whether anyone can see the main window meter. There
// is no point paying for the bridge while the window is hidden, which during
// dictation is most of the time.
func (a *App) levelsWanted() bool {
	if a.dictation == nil || dictation.Snapshot(a.dictation).State != dictation.Recording {
		return false
	}
	return a.mainWindow.visible()
}

func (a *App) emitLevel(level float64) {
	if a.wails != nil {
		a.wails.Event.Emit(levelEvent, level)
	}
}
