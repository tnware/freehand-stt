package compatibility

import "testing"

func TestCatalogAndResolverAgree(t *testing.T) {
	catalog := Profiles()
	for role, profiles := range map[Role][]Profile{Transcription: catalog.Transcription, PostProcessing: catalog.PostProcessing, Speech: catalog.Speech} {
		seen := map[ID]bool{}
		for _, profile := range profiles {
			if seen[profile.ID] {
				t.Fatalf("duplicate %s/%s", role, profile.ID)
			}
			seen[profile.ID] = true
			contract, err := Resolve(profile.ID, role)
			if profile.Available != (err == nil) {
				t.Fatalf("catalog/resolver disagree for %s/%s: %v", role, profile.ID, err)
			}
			if profile.Available && (contract.Path == "" || contract.Profile != profile) {
				t.Fatalf("incomplete contract: %#v", contract)
			}
			if !profile.Available && profile.Capabilities != (Capabilities{}) {
				t.Fatal("planned profile advertises implemented capabilities")
			}
		}
		if _, err := Resolve(ID("unknown"), role); err == nil {
			t.Fatal("unknown profile accepted")
		}
	}
	for _, pair := range []struct {
		id   ID
		role Role
	}{{LlamaCPP, Transcription}, {LlamaCPP, Speech}, {Speaches, PostProcessing}, {Generic, Role("unknown")}} {
		if _, err := Resolve(pair.id, pair.role); err == nil {
			t.Fatalf("wrong role accepted: %#v", pair)
		}
	}
	catalog.Transcription[0].Available = false
	if _, err := Resolve(Generic, Transcription); err != nil {
		t.Fatal("caller mutated authoritative registry")
	}
}
