package windowing

import "testing"

func TestOpenSettingsValidatesSection(t *testing.T) {
	var opened string
	service := NewService(func(section string) { opened = section }, nil, nil, nil, nil)
	if err := service.OpenSettings("processing"); err != nil {
		t.Fatal(err)
	}
	if opened != "processing" {
		t.Fatalf("opened section = %q", opened)
	}
	if err := service.OpenSettings("invented"); err == nil {
		t.Fatal("unknown settings section was accepted")
	}
	if opened != "processing" {
		t.Fatal("invalid section reached the window callback")
	}
}

func TestOpenSettingsDefaultsToGeneral(t *testing.T) {
	var opened string
	service := NewService(func(section string) { opened = section }, nil, nil, nil, nil)
	if err := service.OpenSettings("  "); err != nil {
		t.Fatal(err)
	}
	if opened != "general" {
		t.Fatalf("opened section = %q, want general", opened)
	}
}

func TestShellReadyIsExplicit(t *testing.T) {
	NewService(nil, nil, nil, nil, nil).ShellReady()
	ready := false
	NewService(nil, func() { ready = true }, nil, nil, nil).ShellReady()
	if !ready {
		t.Fatal("readiness callback was not invoked")
	}
}

func TestAboutWindowActions(t *testing.T) {
	opened := false
	hidden := false
	service := NewService(
		nil, nil,
		func() { opened = true },
		func() { hidden = true },
		func() bool { return opened && !hidden },
	)
	if err := service.OpenAbout(); err != nil {
		t.Fatal(err)
	}
	if !service.AboutVisible() {
		t.Fatal("opened About window was reported hidden")
	}
	service.HideAbout()
	if !hidden || service.AboutVisible() {
		t.Fatal("About hide did not reach its native callbacks")
	}
	if err := NewService(nil, nil, nil, nil, nil).OpenAbout(); err == nil {
		t.Fatal("missing About window was treated as available")
	}
}

func TestOpenSettingsAcceptsEveryTask(t *testing.T) {
	for _, section := range []string{"server", "processing", "speech"} {
		t.Run(section, func(t *testing.T) {
			opened := ""
			service := NewService(func(s string) { opened = s }, nil, nil, nil, nil)
			if err := service.OpenSettings(section); err != nil || opened != section {
				t.Fatalf("task setup inaccessible: section=%s err=%v", opened, err)
			}
		})
	}
}
