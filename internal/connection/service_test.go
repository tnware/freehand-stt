package connection

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/inference"
)

type keyFake struct {
	value string
	reads int
}

func (f *keyFake) Get() (string, error) {
	f.reads++
	if f.value == "" {
		return "", credential.ErrNotFound
	}
	return f.value, nil
}
func (f *keyFake) Set(value string) error { f.value = value; return nil }
func (f *keyFake) Delete() error          { f.value = ""; return nil }
func (f *keyFake) Configured() bool       { return f.value != "" }

func sttRequest(cfg config.Settings, draft string) ConnectionTestRequest {
	return ConnectionTestRequest{
		BaseURL: cfg.BaseURL, AllowInsecureHTTP: cfg.AllowInsecureHTTP,
		AuthenticationMode: cfg.AuthenticationMode, Model: cfg.Model,
		HealthPath: cfg.HealthPath, Headers: cfg.Headers, CredentialDraft: draft,
	}
}

func TestConnectionReturnsStructuredCredentialFailure(t *testing.T) {
	service := NewService(&keyFake{}, &keyFake{}, &keyFake{}, inference.New(), nil)
	cfg := config.Default()
	cfg.BaseURL = "https://example.test/v1"
	cfg.Model = "speech/stt"
	cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	result := service.TestConnection(sttRequest(cfg, ""))
	if result.ErrorKind != ConnectionErrorCredentialMissing || result.Reachable || result.Probe != ConnectionProbeModels || result.ModelPresence != ModelPresenceUnavailable {
		t.Fatalf("connection result = %#v", result)
	}
}

func TestConnectionUsesDraftWithoutPersistingOrReadingStoredCredential(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer draft-secret" {
			t.Errorf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	cfg := config.Default()
	cfg.BaseURL = server.URL
	cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	keys := &keyFake{}
	service := NewService(keys, &keyFake{}, &keyFake{}, &inference.Client{HTTP: server.Client()}, nil)
	result := service.TestConnection(sttRequest(cfg, "draft-secret"))
	if result.ErrorKind != "" || !result.Reachable || keys.reads != 0 || keys.value != "" {
		t.Fatalf("result=%#v reads=%d stored=%q", result, keys.reads, keys.value)
	}
}

func TestConnectionNoAuthOmitsAuthorizationAndCredentialLookup(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if authorization := request.Header.Get("Authorization"); authorization != "" {
			t.Errorf("unexpected authorization header %q", authorization)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"speech/stt"}]}`))
	}))
	defer server.Close()
	cfg := config.Default()
	cfg.BaseURL = server.URL
	cfg.AuthenticationMode = config.AuthenticationModeNone
	keys := &keyFake{value: "preserved"}
	service := NewService(keys, &keyFake{}, &keyFake{}, &inference.Client{HTTP: server.Client()}, nil)
	result := service.TestConnection(sttRequest(cfg, "ignored-draft"))
	if !result.Reachable || result.ErrorKind != "" || keys.reads != 0 {
		t.Fatalf("result=%#v reads=%d", result, keys.reads)
	}
}

func TestTextToSpeechConnectionUsesDedicatedCredentialAndDiscoversModels(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/models" || request.Header.Get("Authorization") != "Bearer tts-secret" {
			t.Errorf("request path=%q authorization=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"tts-1"},{"id":"local/kokoro"}]}`))
	}))
	defer server.Close()
	ttsKeys := &keyFake{value: "tts-secret"}
	service := NewService(&keyFake{value: "stt-secret"}, &keyFake{value: "processing-secret"}, ttsKeys, &inference.Client{HTTP: server.Client()}, nil)
	result := service.TestTextToSpeechConnection(TextToSpeechConnectionTestRequest{BaseURL: server.URL + "/v1", AuthenticationMode: config.AuthenticationModeAPIKey})
	if !result.Reachable || result.ErrorKind != "" || len(result.ModelIDs) != 2 || result.ModelIDs[1] != "local/kokoro" || ttsKeys.reads != 1 {
		t.Fatalf("result=%#v tts key reads=%d", result, ttsKeys.reads)
	}
}
