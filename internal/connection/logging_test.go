package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

// All metadata traffic stays in this in-memory transport; no endpoint is probed.
type metadataLogTransport func(*http.Request) (*http.Response, error)

func (f metadataLogTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type metadataLogCase struct {
	name, resultKind, logKind, outcome, level string
}

var metadataLogCases = []metadataLogCase{
	{"success", "", "", "completed", "INFO"},
	{"invalid", "invalid_settings", "invalid_settings", "failed", "WARN"},
	{"credential_missing", "credential_missing", "credential_missing", "failed", "WARN"},
	{"credential_unavailable", "credential_unavailable", "credential_unavailable", "failed", "ERROR"},
	{"http", "http", "http", "failed", "ERROR"},
	{"response", "response", "response", "failed", "ERROR"},
	{"network", "network", "network", "failed", "ERROR"},
	{"timeout", "timeout", "timeout", "failed", "ERROR"},
	// The existing bridge result classifies cancelled HTTP calls as network.
	{"cancelled", "network", "cancelled", "cancelled", "INFO"},
}

func metadataLogFixture(t *testing.T, scenario string) (*Service, *bytes.Buffer, *int) {
	t.Helper()
	var output bytes.Buffer
	calls := new(int)
	root, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	source := &savedSourceFake{connection: savedconnection.Connection{
		ID: "private-connection-id", Uses: []savedconnection.Purpose{savedconnection.Speech},
		Details: savedconnection.Details{BaseURL: "https://example.test/private-path", CompatibilityProfile: compatibility.KokoroFastAPI, AuthenticationMode: config.AuthenticationModeAPIKey},
	}, key: "private-credential"}
	switch scenario {
	case "credential_missing":
		source.err = credential.ErrNotFound
	case "credential_unavailable":
		source.err = errors.New("private-credential private-provider-error C:/private-file")
	}
	transport := metadataLogTransport(func(r *http.Request) (*http.Response, error) {
		*calls++
		if r.Method != http.MethodGet || (!strings.HasSuffix(r.URL.Path, "/audio/voices") && !strings.HasSuffix(r.URL.Path, "/models")) {
			t.Fatalf("unexpected non-metadata request: %s %s", r.Method, r.URL.Path)
		}
		switch scenario {
		case "invalid", "credential_missing", "credential_unavailable":
			t.Fatal("rejected request reached transport")
		case "network":
			return nil, errors.New("private-provider-error private-credential C:/private-file")
		case "timeout":
			return nil, fmt.Errorf("private-provider-error: %w", context.DeadlineExceeded)
		case "cancelled":
			cancel()
			return nil, r.Context().Err()
		}
		status := http.StatusOK
		body := `{"voices":["private-voice-id"],"data":[{"id":"private-model-id"}]}`
		if scenario == "http" {
			status, body = http.StatusUnauthorized, "private-provider-error private-credential"
		} else if scenario == "response" {
			body = "private-provider-error private-credential"
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	service := NewService(&keyFake{}, &keyFake{}, &keyFake{}, &inference.Client{HTTP: &http.Client{Transport: transport}}, slog.New(slog.NewJSONHandler(&output, nil)), source)
	service.rootContext = root
	return service, &output, calls
}

func assertMetadataLogPair(t *testing.T, output *bytes.Buffer, operation string, tc metadataLogCase) {
	t.Helper()
	for _, secret := range []string{"private-credential", "private-provider-error", "private-path", "private-model-id", "private-voice-id", "private-connection-id", "C:/private-file", "private-header", "private-query"} {
		if strings.Contains(output.String(), secret) {
			t.Errorf("log leaked synthetic content %q", secret)
		}
	}
	var records []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	for decoder.More() {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	if len(records) != 2 {
		t.Fatalf("want exactly one start/terminal pair, got %d: %s", len(records), output)
	}
	if records[0]["msg"] != operation+" started" || records[0]["level"] != "INFO" {
		t.Errorf("unexpected start: %v", records[0])
	}
	terminal := records[1]
	for key, want := range map[string]any{"msg": operation + " " + tc.outcome, "level": tc.level, "outcome": tc.outcome, "error_kind": tc.logKind, "component": "connection"} {
		if terminal[key] != want {
			t.Errorf("terminal %s = %v, want %v", key, terminal[key], want)
		}
	}
	if duration, ok := terminal["duration_ms"].(float64); !ok || duration < 0 {
		t.Errorf("missing/non-numeric/negative duration_ms: %v", terminal)
	}
}

func TestVoiceDiscoveryLogOutcomes(t *testing.T) {
	for _, tc := range metadataLogCases {
		t.Run(tc.name, func(t *testing.T) {
			service, output, calls := metadataLogFixture(t, tc.name)
			request := VoiceListRequest{ConnectionID: "private-connection-id", Model: "private-model-id"}
			if tc.name == "invalid" {
				request.Model += "\n"
			}
			result := service.ListSpeechVoices(request)
			if result.ErrorKind != tc.resultKind {
				t.Fatalf("result error_kind = %q, want %q", result.ErrorKind, tc.resultKind)
			}
			if tc.name == "success" && (len(result.Voices) != 1 || result.Voices[0].ID != "private-voice-id" || *calls != 1) {
				t.Fatalf("successful discovery changed: %+v, calls=%d", result, *calls)
			}
			assertMetadataLogPair(t, output, "voice discovery", tc)
		})
	}
}

type metadataLogKeys struct {
	keyFake
	err error
}

func (f *metadataLogKeys) Get() (string, error) { return "", f.err }

func TestConnectionMetadataLogOutcomes(t *testing.T) {
	for _, operation := range []string{"connection test", "post-processing connection test", "speech playback connection test", "saved connection test"} {
		t.Run(operation, func(t *testing.T) {
			for _, tc := range metadataLogCases {
				t.Run(tc.name, func(t *testing.T) {
					if operation == "post-processing connection test" && tc.name == "credential_missing" {
						// Cleanup credentials are optional, unlike API-key STT/TTS.
						t.Skip("missing optional credential is not a rejection")
					}
					service, output, calls := metadataLogFixture(t, tc.name)
					base := "https://example.test/private-path"
					draft := "private-credential"
					if tc.name == "invalid" {
						base += "?private-query=private-credential"
						service.savedConnections = nil
					}
					if strings.HasPrefix(tc.name, "credential_") {
						draft = ""
						keys := &metadataLogKeys{err: service.savedConnections.(*savedSourceFake).err}
						service.keys, service.processKeys, service.ttsKeys = keys, keys, keys
					}
					var result ConnectionResult
					switch operation {
					case "connection test":
						result = service.TestConnection(ConnectionTestRequest{BaseURL: base, AuthenticationMode: config.AuthenticationModeAPIKey, Model: "private-model-id", CredentialDraft: draft, Headers: map[string]string{"X-Fixture": "private-header"}})
					case "post-processing connection test":
						result = service.TestPostProcessingConnection(PostProcessingConnectionTestRequest{BaseURL: base, Model: "private-model-id", CredentialDraft: draft})
					case "speech playback connection test":
						result = service.TestTextToSpeechConnection(TextToSpeechConnectionTestRequest{BaseURL: base, AuthenticationMode: config.AuthenticationModeAPIKey, Model: "private-model-id", CredentialDraft: draft})
					case "saved connection test":
						result = service.TestSavedConnection("private-connection-id")
					}
					if string(result.ErrorKind) != tc.resultKind {
						t.Fatalf("result error_kind = %q, want %q", result.ErrorKind, tc.resultKind)
					}
					if tc.name == "success" && (!result.Reachable || len(result.ModelIDs) != 1 || result.ModelIDs[0] != "private-model-id" || *calls != 1) {
						t.Fatalf("successful metadata result changed: %+v, calls=%d", result, *calls)
					}
					assertMetadataLogPair(t, output, operation, tc)
				})
			}
		})
	}
}
