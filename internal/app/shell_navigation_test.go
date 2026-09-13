package app

import (
	"github.com/tnware/freehand-stt/internal/windowing"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestShellNavigationStartupCloseSurvivesNavigation(t *testing.T) {
	var navigation shellNavigation
	var got []string
	emit := func(event, value string) { got = append(got, event+":"+value) }
	navigation.request("shell:close-requested", "", emit)
	navigation.request("settings:open", "speech", emit)
	navigation.request("settings:open", "audio", emit)
	if len(got) != 0 {
		t.Fatal("delivered before listeners")
	}
	navigation.ready(emit)
	navigation.ready(emit)
	navigation.request("connections:open", "", emit)
	if !reflect.DeepEqual(got, []string{"settings:open:audio", "shell:close-requested:", "connections:open:"}) {
		t.Fatalf("delivery = %v", got)
	}
}

func TestShellNavigationConnectionSettingsReadyThenFreshConnection(t *testing.T) {
	var navigation shellNavigation
	var events []string
	emit := func(event, value string) { events = append(events, event+":"+value) }
	service := windowing.NewService(func(section string) { navigation.request("settings:open", section, emit) }, func() { navigation.ready(emit) }, nil, nil, nil)
	windowing.ConfigureConnections(service, windowing.ConnectionNavigation{Exists: func(string) bool { return true }})
	if err := service.OpenConnectionManager(windowing.ConnectionManagerRequest{ID: "A"}); err != nil {
		t.Fatal(err)
	}
	if err := service.OpenSettings("speech"); err != nil {
		t.Fatal(err)
	}
	service.ShellReady()
	if state := service.TakeSettingsRequest(); !state.Pending || state.Request.Connection != nil {
		t.Fatalf("settings retained A: %+v", state)
	}
	if err := service.OpenConnectionManager(windowing.ConnectionManagerRequest{ID: "B"}); err != nil {
		t.Fatal(err)
	}
	state := service.TakeSettingsRequest()
	if !state.Pending || (state.Request.Connection == nil || state.Request.Connection.ID != "B") {
		t.Fatalf("fresh B lost: %+v", state)
	}
	if service.TakeSettingsRequest().Pending {
		t.Fatal("consumed B replayed")
	}
	if !reflect.DeepEqual(events, []string{"settings:open:speech", "settings:open:connections"}) {
		t.Fatalf("events: %v", events)
	}
}

func TestShellNavigationFreshDispatchCannotOvertakeReplay(t *testing.T) {
	var navigation shellNavigation
	entered, release, freshDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	var got []string
	emit := func(event, value string) {
		if value == "pending" {
			close(entered)
			<-release
		}
		mu.Lock()
		got = append(got, value)
		mu.Unlock()
	}
	navigation.request("settings:open", "pending", emit)
	readyDone := make(chan struct{})
	go func() { navigation.ready(emit); close(readyDone) }()
	<-entered
	go func() { navigation.request("settings:open", "fresh", emit); close(freshDone) }()
	select {
	case <-freshDone:
		t.Error("fresh dispatch overtook blocked replay")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	<-readyDone
	<-freshDone
	if !reflect.DeepEqual(got, []string{"pending", "fresh"}) {
		t.Fatalf("dispatch order = %v", got)
	}
}

func TestStartupCloseDoesNotRevealMain(t *testing.T) {
	a := &App{shell: &shellNavigation{}, mainWindow: &windowController{}}
	a.requestShellClose()
	if a.mainWindow.pending {
		t.Fatal("close revealed main")
	}
	if !a.shell.pendingClose {
		t.Fatal("startup close dropped")
	}
}
