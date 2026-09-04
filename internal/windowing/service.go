// Package windowing owns the renderer boundary for native application windows.
// The application composition root retains the actual Wails window handles;
// this service exposes only the narrow actions the renderer is allowed to ask
// for.
package windowing

import (
	"errors"
	"strings"
)

var settingsSections = map[string]struct{}{
	"general":    {},
	"shortcuts":  {},
	"audio":      {},
	"overlay":    {},
	"server":     {},
	"processing": {},
	"history":    {},
}

type Service struct {
	openSettings    func(string)
	hideSettings    func()
	settingsVisible func() bool
	openAbout       func()
	hideAbout       func()
	aboutVisible    func() bool
}

func NewService(
	openSettings func(string),
	hideSettings func(),
	settingsVisible func() bool,
	openAbout func(),
	hideAbout func(),
	aboutVisible func() bool,
) *Service {
	return &Service{
		openSettings:    openSettings,
		hideSettings:    hideSettings,
		settingsVisible: settingsVisible,
		openAbout:       openAbout,
		hideAbout:       hideAbout,
		aboutVisible:    aboutVisible,
	}
}

// SettingsVisible reports whether the native Settings window is open, including
// while minimised, so a renderer reload can recover the cross-window mutation
// guard without waiting for another event.
func (s *Service) SettingsVisible() bool {
	return s.settingsVisible != nil && s.settingsVisible()
}

// OpenSettings reveals the singleton native settings window at a known
// section. Renderer-controlled values are validated before they reach the
// application window manager.
func (s *Service) OpenSettings(section string) error {
	section = strings.TrimSpace(section)
	if section == "" {
		section = "general"
	}
	if _, ok := settingsSections[section]; !ok {
		return errors.New("unknown settings section")
	}
	if s.openSettings == nil {
		return errors.New("settings window is unavailable")
	}
	s.openSettings(section)
	return nil
}

// HideSettings hides the singleton settings window after the renderer has
// resolved any unsaved draft.
func (s *Service) HideSettings() {
	if s.hideSettings != nil {
		s.hideSettings()
	}
}

// AboutVisible reports whether the reusable native About window is open,
// including while minimised.
func (s *Service) AboutVisible() bool {
	return s.aboutVisible != nil && s.aboutVisible()
}

// OpenAbout reveals the singleton native About window.
func (s *Service) OpenAbout() error {
	if s.openAbout == nil {
		return errors.New("about window is unavailable")
	}
	s.openAbout()
	return nil
}

// HideAbout hides the singleton About window without destroying its WebView.
func (s *Service) HideAbout() {
	if s.hideAbout != nil {
		s.hideAbout()
	}
}
