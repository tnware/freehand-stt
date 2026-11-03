//go:build linux && cgo

package keyboard

import (
	"errors"
	"sync"

	"github.com/tnware/freehand-stt/internal/hotkey"
)

// Global provides repeat-safe Toggle/Show on native X11 only. Wayland callers
// must use the compositor's portal instead. Each registration owns one XCB
// connection and exact passive grab, without Wails' process-global Xlib handler.
type Global struct {
	mu     sync.Mutex
	closed bool
	hooks  map[string]*HoldHook
}

func NewGlobal() *Global { return &Global{hooks: make(map[string]*HoldHook)} }
func (g *Global) Register(value string, callback func()) error {
	if callback == nil {
		return errors.New("global shortcut callback is required")
	}
	chord, err := hotkey.Parse(value)
	if err != nil {
		return err
	}
	if chord == (hotkey.Chord{}) {
		return errors.New("global shortcut is empty")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return errors.New("global shortcuts are closed")
	}
	name := chord.String()
	if _, exists := g.hooks[name]; exists {
		return errors.New("global shortcut is already registered")
	}
	hook := NewHoldHook(callback, nil, nil)
	hook.toggle = true
	if err = hook.Start(name); err != nil {
		_ = hook.Close()
		return err
	}
	g.hooks[name] = hook
	return nil
}
func (g *Global) Unregister(value string) error {
	chord, err := hotkey.Parse(value)
	if err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	hook, exists := g.hooks[chord.String()]
	if !exists {
		return nil
	}
	delete(g.hooks, chord.String())
	return hook.Close()
}
func (g *Global) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	var err error
	for name, hook := range g.hooks {
		err = errors.Join(err, hook.Close())
		delete(g.hooks, name)
	}
	return err
}
