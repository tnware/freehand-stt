package windowing

import (
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"testing"
)

func TestConnectionWindowOwnsDraftThroughStartupAndReopen(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil, nil)
	opens := 0
	ConfigureConnections(s, ConnectionManagerWindow{Open: func() { opens++ }, Exists: func(id string) bool { return id == "existing" }})
	first := ConnectionManagerRequest{Create: true, Purpose: savedconnection.Voice}
	if err := s.OpenConnectionManager(first); err != nil {
		t.Fatal(err)
	}
	if err := s.OpenConnectionManager(ConnectionManagerRequest{ID: "existing"}); err != nil {
		t.Fatal(err)
	}
	state := s.CurrentConnectionManager()
	if opens != 2 || !state.Visible || state.Request != first {
		t.Fatal("reopen replaced the manager draft")
	}
	s.HideConnectionManager()
	if state = s.CurrentConnectionManager(); state.Visible || state.Request != (ConnectionManagerRequest{}) {
		t.Fatal("hide retained navigation state")
	}
	if err := s.OpenConnectionManager(ConnectionManagerRequest{ID: "existing"}); err != nil {
		t.Fatal(err)
	}
	if s.CurrentConnectionManager().Request.ID != "existing" {
		t.Fatal("new open did not select connection")
	}
}
func TestConnectionWindowRejectsInvalidNavigation(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil, nil)
	ConfigureConnections(s, ConnectionManagerWindow{Open: func() { t.Fatal("invalid navigation opened native window") }, Exists: func(string) bool { return false }})
	for _, r := range []ConnectionManagerRequest{{Purpose: "invented"}, {ID: "deleted"}, {ID: "bad\n"}, {Create: true, ID: "existing"}} {
		if s.OpenConnectionManager(r) == nil {
			t.Fatal("invalid navigation accepted")
		}
	}
}
