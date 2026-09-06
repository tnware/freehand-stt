package connection

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func checkFor(t *testing.T, checks []Check, kind CheckKind) Check {
	t.Helper()
	for _, c := range checks {
		if c.Kind == kind {
			return c
		}
	}
	t.Fatal("missing check", kind)
	return Check{}
}
func TestAssessmentSeparatesMetadataModelAndConfiguration(t *testing.T) {
	opts := modelsettings.Defaults()[savedconnection.Transcription]
	result := ConnectionResult{Reachable: true, HTTPStatus: 200, Probe: ConnectionProbeModels, ModelPresence: ModelPresenceNotListed}
	checks := assess(result, savedconnection.Transcription, compatibility.Generic, "alias", &opts, config.AuthenticationModeAPIKey)
	if checkFor(t, checks, CheckConnection).Status != CheckPassed || checkFor(t, checks, CheckModel).Status != CheckAttention || checkFor(t, checks, CheckConfiguration).Status != CheckPassed {
		t.Fatal("checks were conflated")
	}
	if !strings.Contains(checkFor(t, checks, CheckAuthentication).Detail, "does not prove") {
		t.Fatal("metadata claimed inference authorization")
	}
	result.Probe = ConnectionProbeHealth
	result.ModelPresence = ModelPresenceUnavailable
	checks = assess(result, savedconnection.Transcription, compatibility.WhisperCPP, "", &opts, config.AuthenticationModeNone)
	if c := checkFor(t, checks, CheckModel); c.Status != CheckUnverified || c.Summary != "Server-loaded model" {
		t.Fatal("health claimed model verification")
	}
}
func TestAssessmentRequiresVoiceAndServerSideReasoningConfirmation(t *testing.T) {
	result := ConnectionResult{Reachable: true, HTTPStatus: 200, Probe: ConnectionProbeModels, ModelPresence: ModelPresenceListed}
	speech := modelsettings.Defaults()[savedconnection.Speech]
	checks := assess(result, savedconnection.Speech, compatibility.Speaches, "voice-model", &speech, config.AuthenticationModeNone)
	if c := checkFor(t, checks, CheckConfiguration); c.Status != CheckAttention || c.Summary != "Choose a voice" {
		t.Fatal("missing voice accepted")
	}
	cleanup := modelsettings.Defaults()[savedconnection.Cleanup]
	cleanup.Profile = modelprofile.S1Mini
	checks = assess(result, savedconnection.Cleanup, compatibility.Generic, "s1", &cleanup, config.AuthenticationModeNone)
	if checkFor(t, checks, CheckConfiguration).Status != CheckUnverified {
		t.Fatal("generic reasoning assumed disabled")
	}
	checks = assess(result, savedconnection.Cleanup, compatibility.LlamaCPP, "s1", &cleanup, config.AuthenticationModeNone)
	if checkFor(t, checks, CheckConfiguration).Status != CheckPassed {
		t.Fatal("qualified reasoning override not recognized")
	}
}
func TestDiagnosticsUseOnlyMetadataEvenWhenModelOptionsAreInvalid(t *testing.T) {
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/v1/models" {
			t.Error("unexpected inference request")
		}
		w.Write([]byte(`{"data":[{"id":"asr"}]}`))
	}))
	defer server.Close()
	opts := modelsettings.Defaults()[savedconnection.Transcription]
	opts.Transcription.Hotwords = "unsupported-canary"
	s := NewService(&keyFake{}, &keyFake{}, &keyFake{}, &inference.Client{HTTP: server.Client()}, nil)
	result := s.TestConnection(ConnectionTestRequest{BaseURL: server.URL + "/v1", CompatibilityProfile: compatibility.Generic, AuthenticationMode: config.AuthenticationModeNone, Model: "asr", Options: &opts})
	if calls != 1 || result.ErrorKind != "" || checkFor(t, result.Checks, CheckConfiguration).Status != CheckAttention {
		t.Fatal("model validation blocked metadata or failed to report invalid options")
	}
	body, _ := json.Marshal(result.Checks)
	if strings.Contains(string(body), "unsupported-canary") {
		t.Fatal("diagnostics reflected option contents")
	}
}
func TestAuthenticationFailuresAndMetadataOnlySavedScope(t *testing.T) {
	for _, status := range []int{401, 403} {
		r := ConnectionResult{Reachable: true, HTTPStatus: status, ErrorKind: ConnectionErrorHTTP}
		checks := assess(r, "", compatibility.Generic, "", nil, config.AuthenticationModeAPIKey)
		if len(checks) != 2 || checkFor(t, checks, CheckAuthentication).Status != CheckAttention {
			t.Fatal("authentication failure or saved scope incorrect")
		}
	}
}
