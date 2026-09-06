package connection

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVoiceDiscoveryCapturesSavedConnectionAndCredential(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/v1/audio/voices" || r.Header.Get("Authorization") != "Bearer saved-key" {
			t.Error("incorrect captured request")
		}
		w.Write([]byte(`{"voices":["af_heart"]}`))
	}))
	defer server.Close()
	source := &savedSourceFake{connection: savedconnection.Connection{ID: "saved", Uses: []savedconnection.Purpose{savedconnection.Speech}, Details: savedconnection.Details{BaseURL: server.URL + "/v1", CompatibilityProfile: compatibility.KokoroFastAPI, AuthenticationMode: config.AuthenticationModeAPIKey}}, key: "saved-key"}
	active := &keyFake{value: "unrelated-key"}
	service := NewService(active, active, active, &inference.Client{HTTP: server.Client()}, nil, source)
	request := VoiceListRequest{ConnectionID: "saved", Model: "kokoro"}
	result := service.ListSpeechVoices(request)
	if result.ErrorKind != "" || len(result.Voices) != 1 || calls != 1 || active.reads != 0 {
		t.Fatal(result)
	}
	source.err = credential.ErrNotFound
	if result = service.ListSpeechVoices(request); result.ErrorKind != "credential_missing" || calls != 1 {
		t.Fatal(result)
	}
	source.err = nil
	service.ServiceShutdown()
	if result = service.ListSpeechVoices(request); result.ErrorKind == "" || calls != 1 {
		t.Fatal("shutdown permitted discovery", result)
	}
	source.connection.Uses = []savedconnection.Purpose{savedconnection.Transcription}
	if result = service.ListSpeechVoices(request); result.ErrorKind != "invalid_settings" || calls != 1 {
		t.Fatal("wrong role accepted")
	}
}
