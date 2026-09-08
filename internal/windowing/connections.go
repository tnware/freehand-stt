package windowing

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/savedconnection"
)

// ConnectionManagerRequest carries navigation only. No endpoint or credential
// draft crosses windows; the manager fetches a fresh settings snapshot.
type ConnectionManagerRequest struct {
	ID      string                  `json:"id"`
	Purpose savedconnection.Purpose `json:"purpose"`
	Create  bool                    `json:"create"`
}

type ConnectionManagerState struct {
	Visible bool                     `json:"visible"`
	Request ConnectionManagerRequest `json:"request"`
}

type ConnectionManagerWindow struct {
	Open   func()
	Hide   func()
	Exists func(string) bool
}

func ConfigureConnections(s *Service, window ConnectionManagerWindow) { s.connections = window }

func (s *Service) OpenConnectionManager(request ConnectionManagerRequest) error {
	if request.Purpose != "" && !savedconnection.ValidPurpose(request.Purpose) {
		return errors.New("unknown connection purpose")
	}
	if len(request.ID) > 64 || !utf8.ValidString(request.ID) || strings.IndexFunc(request.ID, unicode.IsControl) >= 0 || (request.Create && request.ID != "") {
		return errors.New("invalid connection selection")
	}
	if request.ID != "" && (s.connections.Exists == nil || !s.connections.Exists(request.ID)) {
		return errors.New("connection no longer exists")
	}
	if s.connections.Open == nil {
		return errors.New("connection manager is unavailable")
	}
	s.connectionMu.Lock()
	// Reopening an already visible manager must not overwrite an unsaved draft.
	if !s.connectionOpen {
		s.connectionRequest = request
	}
	s.connectionOpen = true
	s.connectionMu.Unlock()
	s.connections.Open()
	return nil
}

func (s *Service) CurrentConnectionManager() ConnectionManagerState {
	s.connectionMu.Lock()
	defer s.connectionMu.Unlock()
	// Logical ownership includes startup and minimization. Do not synchronously
	// query a native window from a hidden renderer's initial binding call.
	return ConnectionManagerState{Request: s.connectionRequest, Visible: s.connectionOpen}
}

func (s *Service) HideConnectionManager() {
	if s.connections.Hide != nil {
		s.connections.Hide()
	}
	s.connectionMu.Lock()
	s.connectionRequest = ConnectionManagerRequest{}
	s.connectionOpen = false
	s.connectionMu.Unlock()
}
