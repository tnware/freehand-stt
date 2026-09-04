package windowing

import "testing"

func TestOpenSettingsValidatesSection(t *testing.T) {
	var opened string
	service := NewService(func(section string) { opened = section }, nil, nil, nil, nil, nil)
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
	service := NewService(func(section string) { opened = section }, nil, nil, nil, nil, nil)
	if err := service.OpenSettings("  "); err != nil {
		t.Fatal(err)
	}
	if opened != "general" {
		t.Fatalf("opened section = %q, want general", opened)
	}
}

func TestHideSettingsIsOptional(t *testing.T) {
	NewService(nil, nil, nil, nil, nil, nil).HideSettings()
	hidden := false
	NewService(nil, func() { hidden = true }, nil, nil, nil, nil).HideSettings()
	if !hidden {
		t.Fatal("hide callback was not invoked")
	}
}

func TestSettingsVisibleUsesNativeState(t *testing.T) {
	service := NewService(nil, nil, func() bool { return true }, nil, nil, nil)
	if !service.SettingsVisible() {
		t.Fatal("visible native settings window was reported hidden")
	}
	if NewService(nil, nil, nil, nil, nil, nil).SettingsVisible() {
		t.Fatal("missing settings window was reported visible")
	}
}

func TestAboutWindowActions(t *testing.T) {
	opened := false
	hidden := false
	service := NewService(
		nil, nil, nil,
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
	if err := NewService(nil, nil, nil, nil, nil, nil).OpenAbout(); err == nil {
		t.Fatal("missing About window was treated as available")
	}
}
