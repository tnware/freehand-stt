package shortcut

import (
	"errors"
	"fmt"
	"sync"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/hotkey"
)

type Global interface {
	Register(string, func()) error
	Unregister(string) error
}

type Hold interface{ Configure(string) error }

type Controller struct {
	mu        sync.Mutex
	global    Global
	hold      Hold
	active    config.Settings
	toggle    func()
	show      func()
	hasState  bool
	suspended bool
}

func New(global Global, hold Hold, toggle, show func()) *Controller {
	return &Controller{global: global, hold: hold, toggle: toggle, show: show}
}

func normalize(settings config.Settings) (config.Settings, error) {
	var err error
	settings.ToggleShortcut, err = hotkey.NormalizeFor(hotkey.ToggleRecording, settings.ToggleShortcut)
	if err != nil {
		return config.Settings{}, err
	}
	settings.ShowShortcut, err = hotkey.NormalizeFor(hotkey.ShowFreehand, settings.ShowShortcut)
	if err != nil {
		return config.Settings{}, err
	}
	settings.HoldShortcut, err = hotkey.NormalizeFor(hotkey.HoldToTalk, settings.HoldShortcut)
	if err != nil {
		return config.Settings{}, err
	}
	return settings, nil
}

func (c *Controller) Configure(next config.Settings) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	next, err = normalize(next)
	if err != nil {
		return fmt.Errorf("shortcut configuration is invalid: %w", err)
	}
	if c.suspended {
		return errors.New("shortcuts cannot be changed while shortcut capture is active")
	}
	old := c.active
	if c.hasState && ((next.ToggleShortcut == old.ShowShortcut && next.ToggleShortcut != old.ToggleShortcut) ||
		(next.ShowShortcut == old.ToggleShortcut && next.ShowShortcut != old.ShowShortcut)) {
		return errors.New("toggle/show shortcut swap cannot be applied atomically; save an unused intermediate shortcut first")
	}
	type binding struct {
		action hotkey.ShortcutAction
		value  string
		cb     func()
	}
	newBindings := []binding{}
	oldBindings := []binding{}
	if !c.hasState || next.ToggleShortcut != old.ToggleShortcut {
		newBindings = append(newBindings, binding{hotkey.ToggleRecording, next.ToggleShortcut, c.toggle})
		if c.hasState {
			oldBindings = append(oldBindings, binding{hotkey.ToggleRecording, old.ToggleShortcut, c.toggle})
		}
	}
	if !c.hasState || next.ShowShortcut != old.ShowShortcut {
		newBindings = append(newBindings, binding{hotkey.ShowFreehand, next.ShowShortcut, c.show})
		if c.hasState {
			oldBindings = append(oldBindings, binding{hotkey.ShowFreehand, old.ShowShortcut, c.show})
		}
	}
	registered := []binding{}
	for _, item := range newBindings {
		if err := c.global.Register(item.value, item.cb); err != nil {
			for i := len(registered) - 1; i >= 0; i-- {
				_ = c.global.Unregister(registered[i].value)
			}
			return fmt.Errorf("%s shortcut %q was rejected by Windows; it may be reserved or already used by another application: %w", hotkey.ActionLabel(item.action), item.value, err)
		}
		registered = append(registered, item)
	}
	if c.hold != nil && (!c.hasState || next.HoldShortcut != old.HoldShortcut) {
		if err := c.hold.Configure(next.HoldShortcut); err != nil {
			for i := len(registered) - 1; i >= 0; i-- {
				_ = c.global.Unregister(registered[i].value)
			}
			return fmt.Errorf("hold shortcut was not changed: %w", err)
		}
	}
	for _, item := range oldBindings {
		if err := c.global.Unregister(item.value); err != nil {
			for _, prior := range oldBindings {
				_ = c.global.Register(prior.value, prior.cb)
			}
			for i := len(registered) - 1; i >= 0; i-- {
				_ = c.global.Unregister(registered[i].value)
			}
			if c.hold != nil {
				_ = c.hold.Configure(old.HoldShortcut)
			}
			return fmt.Errorf("old shortcut could not be released; previous bindings were restored: %w", err)
		}
	}
	c.active = next
	c.hasState = true
	return nil
}

func (c *Controller) Suspend() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.suspended {
		return nil
	}
	if !c.hasState {
		return errors.New("shortcuts are not configured yet")
	}
	bindings := []struct {
		action hotkey.ShortcutAction
		value  string
		cb     func()
	}{{hotkey.ToggleRecording, c.active.ToggleShortcut, c.toggle}, {hotkey.ShowFreehand, c.active.ShowShortcut, c.show}}
	unregistered := 0
	for index, binding := range bindings {
		if err := c.global.Unregister(binding.value); err != nil {
			var rollback error
			for prior := index - 1; prior >= 0; prior-- {
				rollback = errors.Join(rollback, c.global.Register(bindings[prior].value, bindings[prior].cb))
			}
			return errors.Join(fmt.Errorf("shortcut capture could not suspend %q: %w", binding.value, err), rollback)
		}
		unregistered++
	}
	if c.hold != nil {
		if err := c.hold.Configure(""); err != nil {
			var rollback error
			for _, binding := range bindings {
				rollback = errors.Join(rollback, c.global.Register(binding.value, binding.cb))
			}
			return errors.Join(fmt.Errorf("hold-to-talk could not be suspended for shortcut capture: %w", err), rollback)
		}
	}
	if unregistered == len(bindings) {
		c.suspended = true
	}
	return nil
}

func (c *Controller) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.suspended {
		return nil
	}
	bindings := []struct {
		action hotkey.ShortcutAction
		value  string
		cb     func()
	}{{hotkey.ToggleRecording, c.active.ToggleShortcut, c.toggle}, {hotkey.ShowFreehand, c.active.ShowShortcut, c.show}}
	if c.hold != nil {
		if err := c.hold.Configure(c.active.HoldShortcut); err != nil {
			return fmt.Errorf("hold-to-talk could not be restored after shortcut capture: %w", err)
		}
	}
	registered := 0
	for index, binding := range bindings {
		if err := c.global.Register(binding.value, binding.cb); err != nil {
			var rollback error
			for prior := index - 1; prior >= 0; prior-- {
				rollback = errors.Join(rollback, c.global.Unregister(bindings[prior].value))
			}
			if c.hold != nil {
				rollback = errors.Join(rollback, c.hold.Configure(""))
			}
			return errors.Join(fmt.Errorf("shortcut capture ended but the %s shortcut %q could not be restored: %w", hotkey.ActionLabel(binding.action), binding.value, err), rollback)
		}
		registered++
	}
	if registered == len(bindings) {
		c.suspended = false
	}
	return nil
}
