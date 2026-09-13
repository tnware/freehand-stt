package windowing

import (
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"strings"
	"testing"
)

func TestConnectionNavigationAlwaysKeepsLatestRequest(t *testing.T) {
	opens := 0
	s := NewService(func(string) { opens++ }, nil, nil, nil, nil)
	ConfigureConnections(s, ConnectionNavigation{Exists: func(string) bool { return true }})
	first := ConnectionManagerRequest{Create: true, Purpose: savedconnection.Voice}
	latest := ConnectionManagerRequest{ID: "existing"}
	for _, req := range []ConnectionManagerRequest{first, latest} {
		if err := s.OpenConnectionManager(req); err != nil {
			t.Fatal(err)
		}
	}
	state := s.TakeSettingsRequest()
	if opens != 2 || !state.Pending || (state.Request.Connection == nil || *state.Request.Connection != latest) {
		t.Fatalf("latest navigation = %+v", state)
	}
	for i := 0; i < 3; i++ {
		if state = s.TakeSettingsRequest(); state.Pending || state.Request != (SettingsRequest{}) {
			t.Fatalf("consume retained request: %+v", state)
		}
	}
	if err := s.OpenConnectionManager(first); err != nil {
		t.Fatal(err)
	}
	_ = s.TakeSettingsRequest()
	if state = s.TakeSettingsRequest(); state.Pending || state.Request != (SettingsRequest{}) {
		t.Fatal("clear retained request")
	}
}

func TestSettingsSupersedesConnectionNavigation(t *testing.T) {
	s := NewService(func(string) {}, nil, nil, nil, nil)
	ConfigureConnections(s, ConnectionNavigation{})
	if err := s.OpenConnectionManager(ConnectionManagerRequest{Create: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.OpenSettings("speech"); err != nil {
		t.Fatal(err)
	}
	if state := s.TakeSettingsRequest(); !state.Pending || state.Request.Connection != nil {
		t.Fatal("settings retained stale connection request")
	}
}

func TestConnectionNavigationRejectsInvalidRequestsWithoutReplacingPending(t *testing.T) {
	opens := 0
	s := NewService(func(string) { opens++ }, nil, nil, nil, nil)
	ConfigureConnections(s, ConnectionNavigation{Exists: func(id string) bool { return id == "existing" }})
	valid := ConnectionManagerRequest{ID: "existing"}
	for _, req := range []ConnectionManagerRequest{{Purpose: "invented"}, {ID: "deleted"}, {ID: "bad\n"}, {ID: strings.Repeat("a", 65)}, {ID: string([]byte{255})}, {Create: true, ID: "existing"}} {
		if err := s.OpenConnectionManager(valid); err != nil {
			t.Fatal(err)
		}
		before := opens
		if s.OpenConnectionManager(req) == nil {
			t.Fatal("invalid navigation accepted")
		}
		state := s.TakeSettingsRequest()
		if opens != before || !state.Pending || (state.Request.Connection == nil || *state.Request.Connection != valid) {
			t.Fatal("invalid request changed navigation")
		}
	}
	if NewService(nil, nil, nil, nil, nil).OpenConnectionManager(ConnectionManagerRequest{}) == nil {
		t.Fatal("missing navigation accepted")
	}
}
