package windowing

import (
	"reflect"
	"testing"
)

func TestTrayPopoverNavigation(t *testing.T) {
	s := NewService(nil, nil, nil, nil, nil)
	if s.OpenMain() == nil {
		t.Fatal("unconfigured main must report unavailable")
	}
	s.HideTrayPopover()
	if s.TrayPopoverVisible() {
		t.Fatal("unconfigured popover visible")
	}
	calls := []string{}
	visible := true
	ConfigureTrayPopover(s, TrayPopoverNavigation{
		OpenMain: func() { calls = append(calls, "main") },
		Hide:     func() { calls = append(calls, "hide"); visible = false },
		Visible:  func() bool { return visible },
	})
	if !s.TrayPopoverVisible() {
		t.Fatal("visibility not delegated")
	}
	if err := s.OpenMain(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"hide", "main"}) {
		t.Fatalf("calls = %v", calls)
	}
	if s.TrayPopoverVisible() {
		t.Fatal("main left popover open")
	}
	s.HideTrayPopover()
}
