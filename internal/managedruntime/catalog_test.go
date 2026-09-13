package managedruntime

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestCatalogBehaviorMatchesProductionContracts(t *testing.T) {
	b, err := os.ReadFile("testdata/catalog-v0.1.0.json")
	if err != nil {
		t.Fatal(err)
	}
	models, err := parseCatalog(b)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the renderer boundary without depending on a manual backend's
	// settings model-profile catalog.
	encoded, err := json.Marshal(models)
	if err != nil {
		t.Fatal(err)
	}
	var rendered []struct {
		ID       string                `json:"id"`
		Profile  modelprofile.ID       `json:"profile"`
		Realtime bool                  `json:"realtime"`
		Behavior *modelprofile.Profile `json:"behavior"`
	}
	if err := json.Unmarshal(encoded, &rendered); err != nil {
		t.Fatal(err)
	}
	wantProfiles := map[string]modelprofile.ID{
		"nemotron-3.5": modelprofile.Nemotron35,
		"parakeet-tdt": modelprofile.ParakeetTDT,
	}
	if len(rendered) != len(wantProfiles) {
		t.Fatalf("got %d models, want %d", len(rendered), len(wantProfiles))
	}
	for _, model := range rendered {
		t.Run(model.ID, func(t *testing.T) {
			id, ok := wantProfiles[model.ID]
			if !ok || model.Profile != id {
				t.Fatalf("unexpected model/profile: %s/%s", model.ID, model.Profile)
			}
			contract, err := modelprofile.Resolve(id, compatibility.NeMoSpeechV1, compatibility.Transcription)
			if err != nil {
				t.Fatal(err)
			}
			if model.Behavior == nil {
				t.Fatal("renderer catalog is missing resolved behavior metadata")
			}
			if !reflect.DeepEqual(*model.Behavior, contract.Profile) {
				t.Fatalf("behavior = %+v, want production profile %+v", *model.Behavior, contract.Profile)
			}
			if model.Realtime != contract.Capabilities.Realtime {
				t.Fatal("catalog realtime disagrees with production contract")
			}
			if len(model.Behavior.Languages) == 0 {
				t.Fatal("missing authoritative language options")
			}
			for _, language := range model.Behavior.Languages {
				if err := modelprofile.ValidateTranscription(id, compatibility.NeMoSpeechV1, language.Code, compatibility.TranscriptionOptions{}); err != nil {
					t.Fatalf("advertised language %q rejected by request validation: %v", language.Code, err)
				}
			}
			if id == modelprofile.Nemotron35 {
				if !model.Behavior.Capabilities.LanguageHint || !model.Behavior.RealtimeLanguageHint || !model.Realtime {
					t.Fatal("Nemotron must advertise completed/realtime language hints and realtime")
				}
				realtime, err := modelprofile.Resolve(id, compatibility.NeMoSpeechV1, compatibility.Realtime)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(model.Behavior.Languages, realtime.Languages) {
					t.Fatal("Nemotron language controls disagree with realtime request contract")
				}
			} else {
				if model.Realtime || model.Behavior.Capabilities.LanguageHint || model.Behavior.RealtimeLanguageHint {
					t.Fatal("Parakeet must not advertise realtime or language hints")
				}
				if len(model.Behavior.Languages) != 1 || model.Behavior.Languages[0].Code != "auto" {
					t.Fatal("Parakeet controls must offer automatic detection only")
				}
			}
			if model.Behavior.Capabilities.FileStreaming || model.Behavior.Capabilities.TranscriptionPrompt || model.Behavior.Capabilities.TranscriptionHotwords || model.Behavior.Capabilities.TranscriptionTemperature {
				t.Fatal("catalog advertises unsupported transcription options")
			}
		})
	}
}

func TestOfficialMetadataOnlyCatalog(t *testing.T) {
	b, e := os.ReadFile("testdata/catalog-v0.1.0.json")
	if e != nil {
		t.Fatal(e)
	}
	ms, e := parseCatalog(b)
	if e != nil {
		t.Fatal(e)
	}
	if len(ms) != 2 || ms[0].ID != "nemotron-3.5" || ms[1].ID != "parakeet-tdt" || ms[0].Installed || ms[0].SizeBytes != 741548352 {
		t.Fatalf("catalog %+v", ms)
	}
	if _, e = parseCatalog([]byte(`{"models":[{"repo":"nvidia/nemotron-3.5-asr-streaming-0.6b","aliases":["nemotron-3.5"],"revision":"bad","roles":["asr"],"commands":["serve"]}]}`)); e == nil {
		t.Fatal("accepted unqualified revision")
	}
}
