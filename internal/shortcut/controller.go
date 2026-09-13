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

type Hold interface {
	Start(string) error
	Configure(string) error
}

type Controller struct {
	mu     sync.Mutex
	global Global
	hold   Hold
	active config.Settings
	// Bound globals are separate from saved preferences: startup may be degraded.
	boundToggle string
	boundShow   string
	boundHold   string
	toggle      func()
	show        func()
	hasState    bool
	suspended   bool
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

// Start tolerates unavailable bindings without dropping independent shortcuts
// or forgetting saved preferences. Explicit changes stay atomic.
func (c *Controller) Start(next config.Settings) error     { return c.configure(next, true) }
func (c *Controller) Configure(next config.Settings) error { return c.configure(next, false) }
func (c *Controller) configure(next config.Settings, startup bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.configureLocked(next, startup)
}

type bindings struct {
	toggle, show, hold string
}

func (c *Controller) bound() bindings {
	return bindings{c.boundToggle, c.boundShow, c.boundHold}
}

// ConfigureWithRollback captures preferences and effective bindings atomically
// with applying a change. The settings transaction owns the returned rollback;
// restoring it never retries an unavailable saved startup chord.
func (c *Controller) ConfigureWithRollback(next config.Settings) (func() error, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hasState {
		return nil, errors.New("shortcuts are not configured yet")
	}
	previous, effective := c.active, c.bound()
	if err := c.configureLocked(next, false); err != nil {
		return nil, err
	}
	return func() error {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.applyLocked(previous, effective, false)
	}, nil
}

func (c *Controller) configureLocked(next config.Settings, startup bool) error {
	var err error
	next, err = normalize(next)
	if err != nil {
		return fmt.Errorf("shortcut configuration is invalid: %w", err)
	}
	// An unchanged preference must not retry an unavailable startup binding.
	target := c.bound()
	if !c.hasState || next.ToggleShortcut != c.active.ToggleShortcut {
		target.toggle = next.ToggleShortcut
	}
	if !c.hasState || next.ShowShortcut != c.active.ShowShortcut {
		target.show = next.ShowShortcut
	}
	if !c.hasState || next.HoldShortcut != c.active.HoldShortcut {
		target.hold = next.HoldShortcut
	}
	return c.applyLocked(next, target, startup)
}

func (c *Controller) applyLocked(next config.Settings, target bindings, startup bool) error {
	if c.suspended {
		return errors.New("shortcuts cannot be changed while shortcut capture is active")
	}
	if c.hasState && ((target.toggle != "" && target.toggle == c.boundShow && target.toggle != c.boundToggle) ||
		(target.show != "" && target.show == c.boundToggle && target.show != c.boundShow)) {
		return errors.New("toggle/show shortcut swap cannot be applied atomically; save an unused intermediate shortcut first")
	}
	type binding struct {
		action hotkey.ShortcutAction
		value  string
		cb     func()
	}
	newBindings := []binding{}
	oldBindings := []binding{}
	if target.toggle != c.boundToggle {
		if target.toggle != "" {
			newBindings = append(newBindings, binding{hotkey.ToggleRecording, target.toggle, c.toggle})
		}
		if c.boundToggle != "" {
			oldBindings = append(oldBindings, binding{hotkey.ToggleRecording, c.boundToggle, c.toggle})
		}
	}
	if target.show != c.boundShow {
		if target.show != "" {
			newBindings = append(newBindings, binding{hotkey.ShowFreehand, target.show, c.show})
		}
		if c.hasState {
			if c.boundShow != "" {
				oldBindings = append(oldBindings, binding{hotkey.ShowFreehand, c.boundShow, c.show})
			}
		}
	}
	registered := []binding{}
	var startupErr error
	for _, item := range newBindings {
		if err := c.global.Register(item.value, item.cb); err != nil {
			err = fmt.Errorf("%s shortcut %q was rejected by the operating system; it may be reserved or already used by another application. Record a different shortcut or clear it in Settings > Shortcuts: %w", hotkey.ActionLabel(item.action), item.value, err)
			if startup && !c.hasState {
				startupErr = errors.Join(startupErr, err)
				continue
			}
			for i := len(registered) - 1; i >= 0; i-- {
				_ = c.global.Unregister(registered[i].value)
			}
			return err
		}
		registered = append(registered, item)
	}
	var holdErr error
	boundHold := c.boundHold
	if c.hold != nil && (!c.hasState || target.hold != c.boundHold || next.HoldShortcut != c.active.HoldShortcut) {
		apply := c.hold.Configure
		if !c.hasState {
			apply = c.hold.Start
		}
		if err := apply(target.hold); err != nil {
			holdErr = fmt.Errorf("hold-to-talk is unavailable: %w", err)
			if !startup || c.hasState {
				for i := len(registered) - 1; i >= 0; i-- {
					_ = c.global.Unregister(registered[i].value)
				}
				return holdErr
			}
		} else {
			boundHold = target.hold
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
				_ = c.hold.Configure(c.boundHold)
			}
			return fmt.Errorf("old shortcut could not be released; previous bindings were restored: %w", err)
		}
	}
	for _, item := range oldBindings {
		if item.action == hotkey.ToggleRecording {
			c.boundToggle = ""
		} else {
			c.boundShow = ""
		}
	}
	for _, item := range registered {
		if item.action == hotkey.ToggleRecording {
			c.boundToggle = item.value
		} else {
			c.boundShow = item.value
		}
	}
	c.active = next
	c.boundHold = boundHold
	c.hasState = true
	return errors.Join(startupErr, holdErr)
}

// RetryHold is explicit, serialized with capture/settings, and never changes
// preferences or global registrations. Native Configure performs a bounded rearm.
func (c *Controller) RetryHold() error {
	if !c.mu.TryLock() {
		return errors.New("shortcuts are busy; try again")
	}
	defer c.mu.Unlock()
	if !c.hasState || c.suspended {
		return errors.New("hold-to-talk cannot be retried during startup or shortcut capture")
	}
	if c.hold == nil {
		return errors.New("hold-to-talk is unavailable")
	}
	if err := c.hold.Configure(c.active.HoldShortcut); err != nil {
		return err
	}
	c.boundHold = c.active.HoldShortcut
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
	}{{hotkey.ToggleRecording, c.boundToggle, c.toggle}, {hotkey.ShowFreehand, c.boundShow, c.show}}
	unregistered := 0
	for index, binding := range bindings {
		if binding.value == "" {
			unregistered++
			continue
		}
		if err := c.global.Unregister(binding.value); err != nil {
			var rollback error
			for prior := index - 1; prior >= 0; prior-- {
				if bindings[prior].value == "" {
					continue
				}
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
				if binding.value == "" {
					continue
				}
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
	}{{hotkey.ToggleRecording, c.boundToggle, c.toggle}, {hotkey.ShowFreehand, c.boundShow, c.show}}

	// Capture has ended even if held keys or lost permission prevent the optional
	// hook from rearming. Restore the independent globals before reporting that.
	var holdErr error
	if c.hold != nil {
		if err := c.hold.Configure(c.boundHold); err != nil {
			holdErr = fmt.Errorf("hold-to-talk could not be restored after shortcut capture: %w", err)
		}
	}
	registered := 0
	for index, binding := range bindings {
		if binding.value == "" {
			registered++
			continue
		}
		if err := c.global.Register(binding.value, binding.cb); err != nil {
			var rollback error
			for prior := index - 1; prior >= 0; prior-- {
				if bindings[prior].value == "" {
					continue
				}
				rollback = errors.Join(rollback, c.global.Unregister(bindings[prior].value))
			}
			if c.hold != nil {
				rollback = errors.Join(rollback, c.hold.Configure(""))
			}
			return errors.Join(fmt.Errorf("shortcut capture ended but the %s shortcut %q could not be restored: %w", hotkey.ActionLabel(binding.action), binding.value, err), holdErr, rollback)
		}
		registered++
	}
	if registered == len(bindings) {
		c.suspended = false
		if holdErr != nil {
			c.boundHold = ""
		}
	}
	return holdErr
}
