package speechlanguage

import "testing"

func TestPickerCatalogIsIndependentAndStable(t *testing.T) {
	options := Options()
	if len(options) < 180 {
		t.Fatal("missing ISO language choices")
	}
	seen := map[string]bool{}
	for _, option := range options {
		if seen[option.Code] || option.Label == "" {
			t.Fatal("invalid language catalog")
		}
		seen[option.Code] = true
	}
	for _, code := range []string{"en", "es", "fr", "ja", "zh", "haw", "yue"} {
		if !seen[code] {
			t.Errorf("missing %s", code)
		}
	}
	options[0].Label = "changed"
	if Options()[0].Label == "changed" {
		t.Fatal("catalog mutated across snapshots")
	}
}
