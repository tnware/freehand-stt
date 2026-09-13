package windowing

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/savedconnection"
)

// ConnectionManagerRequest carries navigation only. No endpoint or credential
// draft is retained natively; the renderer exclusively owns its editor state.
type ConnectionManagerRequest struct {
	ID      string                  `json:"id"`
	Purpose savedconnection.Purpose `json:"purpose"`
	Create  bool                    `json:"create"`
}

type ConnectionNavigation struct {
	Exists func(string) bool
}

func ConfigureConnections(s *Service, navigation ConnectionNavigation) { s.connections = navigation }

func (s *Service) OpenConnectionManager(request ConnectionManagerRequest) error {
	return s.OpenTaskConnection(request, "")
}

func (s *Service) OpenTaskConnection(request ConnectionManagerRequest, origin string) error {
	if !validOrigin(origin) {
		return errors.New("unknown settings origin")
	}
	if request.Purpose != "" && !savedconnection.ValidPurpose(request.Purpose) {
		return errors.New("unknown connection purpose")
	}
	if len(request.ID) > 64 || !utf8.ValidString(request.ID) || strings.IndexFunc(request.ID, unicode.IsControl) >= 0 || (request.Create && request.ID != "") {
		return errors.New("invalid connection selection")
	}
	if request.ID != "" && (s.connections.Exists == nil || !s.connections.Exists(request.ID)) {
		return errors.New("connection no longer exists")
	}
	if s.openSettings == nil {
		return errors.New("connection manager is unavailable")
	}
	s.navigationMu.Lock()
	defer s.navigationMu.Unlock()
	s.setSettingsRequest(SettingsRequest{Section: "connections", Origin: origin, Connection: &request})
	s.openSettings("connections")
	return nil
}

// SettingsRequest carries only navigation. Draft ownership stays in the renderer.
type SettingsRequest struct {
	Section    string                    `json:"section"`
	Origin     string                    `json:"origin"`
	Connection *ConnectionManagerRequest `json:"connection,omitempty"`
}
type SettingsRequestState struct {
	Pending bool            `json:"pending"`
	Request SettingsRequest `json:"request"`
}
type SettingsNavigation struct {
	Ready   func()
	Visible func() bool
	Finish  func(string)
}

func ConfigureSettings(s *Service, navigation SettingsNavigation) { s.settingsNavigation = navigation }
func validOrigin(origin string) bool {
	switch origin {
	case "", "voice", "file", "tts":
		return true
	}
	return false
}
func (s *Service) setSettingsRequest(request SettingsRequest) {
	s.requestMu.Lock()
	defer s.requestMu.Unlock()
	s.settingsRequest, s.settingsPending = request, true
}
func (s *Service) TakeSettingsRequest() SettingsRequestState {
	s.requestMu.Lock()
	defer s.requestMu.Unlock()
	state := SettingsRequestState{Request: s.settingsRequest, Pending: s.settingsPending}
	s.settingsRequest, s.settingsPending = SettingsRequest{}, false
	return state
}
func (s *Service) SettingsReady() {
	if s.settingsNavigation.Ready != nil {
		s.settingsNavigation.Ready()
	}
}
func (s *Service) SettingsVisible() bool {
	return s.settingsNavigation.Visible != nil && s.settingsNavigation.Visible()
}
func (s *Service) FinishSettings(origin string) error {
	if !validOrigin(origin) {
		return errors.New("unknown settings origin")
	}
	if s.settingsNavigation.Finish == nil {
		return errors.New("settings window is unavailable")
	}
	s.navigationMu.Lock()
	defer s.navigationMu.Unlock()
	s.settingsNavigation.Finish(origin)
	return nil
}
