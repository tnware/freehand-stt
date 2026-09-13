package managedruntime

import "testing"

func TestPreferencesQualified(t *testing.T) {
	p := Defaults()
	if p.Enabled || p.Model != "nemotron-3.5" || !p.Realtime {
		t.Fatalf("defaults: %+v", p)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []Preferences{{Model: "../escape"}, {Model: "magpie"}, {Model: "parakeet-tdt", Realtime: true}, {Model: ""}} {
		if Validate(p) == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
	if err := Validate(Preferences{Model: "parakeet-tdt"}); err != nil {
		t.Fatal(err)
	}
}
