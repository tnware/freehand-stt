package connection

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestProfileProbesRemainMetadataOnlyAndRejectPlannedSelections(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/v1/models" {
			t.Error("probe invoked non-metadata endpoint")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	keys := &keyFake{}
	service := NewService(keys, keys, keys, inference.New(), nil)
	for _, planned := range []bool{false, true} {
		sttID, chatID, ttsID := compatibility.Speaches, compatibility.LlamaCPP, compatibility.VLLMOmni
		if planned {
			sttID, chatID, ttsID = compatibility.LocalAI, compatibility.LocalAI, compatibility.LocalAI
		}
		results := []ConnectionResult{
			service.TestConnection(ConnectionTestRequest{CompatibilityProfile: sttID, BaseURL: server.URL + "/v1", AllowInsecureHTTP: true, AuthenticationMode: config.AuthenticationModeNone}),
			service.TestPostProcessingConnection(PostProcessingConnectionTestRequest{CompatibilityProfile: chatID, BaseURL: server.URL + "/v1", AllowInsecureHTTP: true}),
			service.TestTextToSpeechConnection(TextToSpeechConnectionTestRequest{CompatibilityProfile: ttsID, BaseURL: server.URL + "/v1", AllowInsecureHTTP: true, AuthenticationMode: config.AuthenticationModeNone}),
		}
		for _, result := range results {
			if planned && (result.ErrorKind != ConnectionErrorInvalidSettings || result.Reachable) {
				t.Fatalf("planned profile reached network: %#v", result)
			}
			if !planned && (result.ErrorKind != "" || !result.Reachable) {
				t.Fatalf("enabled profile failed metadata: %#v", result)
			}
		}
	}
	if calls != 3 {
		t.Fatalf("metadata request count=%d", calls)
	}
}

func TestWhisperCPPUsesOnlyHealthAndPreservesCustomPrefix(t *testing.T) {
	for _, custom := range []string{"", "/ready"} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			path := "/proxy/health"
			if custom != "" {
				path = "/proxy/ready"
			}
			if r.Method != "GET" || r.URL.Path != path {
				t.Errorf("unexpected probe %s %s", r.Method, r.URL.Path)
			}
			w.Write([]byte(`{"status":"ok"}`))
		}))
		keys := &keyFake{}
		service := NewService(keys, keys, keys, inference.New(), nil)
		result := service.TestConnection(ConnectionTestRequest{CompatibilityProfile: compatibility.WhisperCPP, BaseURL: server.URL + "/proxy", HealthPath: custom, AllowInsecureHTTP: true, AuthenticationMode: config.AuthenticationModeNone})
		server.Close()
		if calls != 1 || result.ErrorKind != "" || result.Probe != ConnectionProbeHealth || result.ModelPresence != ModelPresenceUnavailable || len(result.ModelIDs) != 0 {
			t.Fatalf("calls=%d result=%+v", calls, result)
		}
	}
}

func TestNeMoCombinedConnectionAndSpeechProbeKeepTheirScopes(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" {
			t.Error("metadata invoked inference")
		}
		switch r.URL.Path {
		case "/v1/models":
			w.Write([]byte(`{"data":[{"id":"asr","capability":"transcription"},{"id":"magpietts","capability":"speech"}]}`))
		case "/health":
			w.Write([]byte(`{"status":"ok","version":"0.1.0"}`))
		default:
			t.Errorf("unexpected metadata path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	source := &savedSourceFake{connection: savedconnection.Connection{ID: "combined", Uses: []savedconnection.Purpose{savedconnection.Voice, savedconnection.Transcription, savedconnection.Speech}, Details: savedconnection.Details{BaseURL: server.URL + "/v1", CompatibilityProfile: compatibility.NeMoSpeechV1, AllowInsecureHTTP: true, AuthenticationMode: config.AuthenticationModeNone}}}
	keys := &keyFake{}
	service := NewService(keys, keys, keys, inference.New(), nil, source)
	all := service.TestSavedConnection("combined")
	if all.ErrorKind != "" || !slices.Equal(all.ModelIDs, []string{"asr", "magpietts"}) || len(all.Models) != 2 {
		t.Fatalf("combined catalog=%+v", all)
	}
	speech := service.TestTextToSpeechConnection(TextToSpeechConnectionTestRequest{CompatibilityProfile: compatibility.NeMoSpeechV1, BaseURL: server.URL + "/v1", AllowInsecureHTTP: true, AuthenticationMode: config.AuthenticationModeNone, Model: "magpietts"})
	if speech.ErrorKind != "" || !slices.Equal(speech.ModelIDs, []string{"magpietts"}) || speech.ModelPresence != ModelPresenceListed || speech.ServerVersion != "0.1.0" || calls != 4 {
		t.Fatalf("speech probe=%+v calls=%d", speech, calls)
	}
}
