// Package windowing owns the renderer boundary for native application windows.
// The application composition root retains the actual Wails window handles;
// this service exposes only the narrow actions the renderer is allowed to ask
// for.
package windowing

import (
	"errors"
	"strings"
	"sync"
)

var settingsSections = map[string]struct{}{
	"vocabulary":          {},
	"general":             {},
	"shortcuts":           {},
	"audio":               {},
	"overlay":             {},
	"server":              {},
	"voice-transcription": {},
	"connections":         {},
	"processing":          {},
	"speech":              {},
	"history":             {},
}

type Service struct {
	navigationMu       sync.Mutex
	requestMu          sync.Mutex
	connections        ConnectionNavigation
	settingsRequest    SettingsRequest
	settingsPending    bool
	settingsNavigation SettingsNavigation
	openSettings       func(string)
	shellReady         func()
	openAbout          func()
	hideAbout          func()
	aboutVisible       func() bool
}

func NewService(
	openSettings func(string),
	shellReady func(),
	openAbout func(),
	hideAbout func(),
	aboutVisible func() bool,
) *Service {
	return &Service{
		openSettings: openSettings,
		shellReady:   shellReady,
		openAbout:    openAbout,
		hideAbout:    hideAbout,
		aboutVisible: aboutVisible,
	}
}

// ShellReady acknowledges that the main renderer has installed its navigation
// listeners. Pending native menu requests may now be delivered exactly once.
func (s *Service) ShellReady() {
	if s.shellReady != nil {
		s.shellReady()
	}
}

// OpenSettings reveals the dedicated singleton Settings window at a known
// section. Renderer-controlled values are validated before they reach the
// application window manager.
func (s *Service) OpenSettings(section string) error { return s.OpenTaskSettings(section, "") }

func (s *Service) OpenTaskSettings(section, origin string) error {
	if !validOrigin(origin) {
		return errors.New("unknown settings origin")
	}
	section = strings.TrimSpace(section)
	if section == "" {
		section = "general"
	}
	if _, ok := settingsSections[section]; !ok {
		return errors.New("unknown settings section")
	}
	if s.openSettings == nil {
		return errors.New("settings navigation is unavailable")
	}
	s.navigationMu.Lock()
	defer s.navigationMu.Unlock()
	s.setSettingsRequest(SettingsRequest{Section: section, Origin: origin})
	s.openSettings(section)
	return nil
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
