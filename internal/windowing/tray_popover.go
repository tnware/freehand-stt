package windowing

import "errors"

// TrayPopoverNavigation is configured once, before services start. It carries
// window actions only; recording and insertion remain owned by dictation.
type TrayPopoverNavigation struct {
	OpenMain func()
	Hide     func()
	Visible  func() bool
}

func ConfigureTrayPopover(s *Service, navigation TrayPopoverNavigation) {
	s.trayPopover = navigation
}

func (s *Service) OpenMain() error {
	if s.trayPopover.OpenMain == nil {
		return errors.New("main window is unavailable")
	}
	s.HideTrayPopover()
	s.trayPopover.OpenMain()
	return nil
}

// HideTrayPopover retains the renderer and never changes active jobs.
func (s *Service) HideTrayPopover() {
	if s.trayPopover.Hide != nil {
		s.trayPopover.Hide()
	}
}

func (s *Service) TrayPopoverVisible() bool {
	return s.trayPopover.Visible != nil && s.trayPopover.Visible()
}
