package windowing

import (
	"reflect"
	"testing"
)

func TestSettingsFinishAndReadinessAreIndependent(t *testing.T) {
	var got []string
	visible := true
	s := NewService(func(string) {}, func() { got = append(got, "main-ready") }, nil, nil, nil)
	ConfigureSettings(s, SettingsNavigation{
		Ready:   func() { got = append(got, "settings-ready") },
		Visible: func() bool { return visible },
		Finish:  func(origin string) { visible = false; got = append(got, "finish:"+origin) },
	})
	if !s.SettingsVisible() {
		t.Fatal("visibility not observed")
	}
	_ = s.OpenTaskSettings("speech", "tts")
	for _, origin := range []string{"invented", " TTS ", "VOICE", "voice\n"} {
		if s.FinishSettings(origin) == nil || s.OpenTaskConnection(ConnectionManagerRequest{}, origin) == nil {
			t.Fatal("invalid origin accepted")
		}
	}
	if len(got) != 0 || !s.SettingsVisible() || !s.TakeSettingsRequest().Pending {
		t.Fatal("invalid action changed native state")
	}
	s.ShellReady()
	s.SettingsReady()
	for _, origin := range []string{"", "voice", "file", "tts"} {
		if err := s.FinishSettings(origin); err != nil {
			t.Fatal(err)
		}
	}
	if s.SettingsVisible() || !reflect.DeepEqual(got, []string{"main-ready", "settings-ready", "finish:", "finish:voice", "finish:file", "finish:tts"}) {
		t.Fatalf("callbacks=%v", got)
	}
}

func TestDedicatedSettingsLatestRequestAndOriginValidation(t *testing.T) {
	opens := 0
	s := NewService(func(string) { opens++ }, nil, nil, nil, nil)
	ConfigureConnections(s, ConnectionNavigation{Exists: func(string) bool { return true }})
	if err := s.OpenTaskConnection(ConnectionManagerRequest{ID: "one"}, "voice"); err != nil {
		t.Fatal(err)
	}
	if err := s.OpenTaskSettings("speech", "invalid"); err == nil {
		t.Fatal("invalid origin accepted")
	}
	state := s.TakeSettingsRequest()
	if opens != 1 || !state.Pending || state.Request.Origin != "voice" || state.Request.Connection == nil || state.Request.Connection.ID != "one" {
		t.Fatalf("invalid origin changed request: %+v", state)
	}
	if s.TakeSettingsRequest().Pending {
		t.Fatal("request was not consumed")
	}
	_ = s.OpenTaskConnection(ConnectionManagerRequest{}, "file")
	_ = s.OpenSettings("speech")
	state = s.TakeSettingsRequest()
	if !state.Pending || state.Request.Section != "speech" || state.Request.Origin != "" || state.Request.Connection != nil || opens != 3 {
		t.Fatalf("latest settings request lost: %+v", state)
	}
}
