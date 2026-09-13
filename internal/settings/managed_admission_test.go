package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"testing"
)

func TestManagedEndpointRejectsForeignIdentityAndTransport(t *testing.T) {
	for _, mode := range []string{"id", "provider", "catalog", "contract", "remote", "ipv6", "userinfo", "query", "fragment", "port", "stopped"} {
		t.Run(mode, func(t *testing.T) {
			s, log, _, keys := transactionalService(false)
			i := testInstance()
			s.cfg.ManagedRuntimes = []managedruntime.Instance{i}
			s.cfg.ManagedInstanceID = i.ID
			s.cfg.AuthenticationMode = config.AuthenticationModeAPIKey
			keys.getErr = errors.New("locked")
			WithManagedRuntimes(func(i managedruntime.Instance, r compatibility.Role) (managedruntime.ResolvedEndpoint, error) {
				e, err := readyManagedEndpoint(i, r)
				switch mode {
				case "id":
					e.InstanceID = "other"
				case "provider":
					e.Provider = "other"
				case "catalog":
					e.CatalogModel = "other"
				case "contract":
					e.Contract.ModelProfile = "generic"
				case "remote":
					e.BaseURL = "http://example.com:80/v1"
				case "ipv6":
					e.BaseURL = "http://[::1]:8080/v1"
				case "userinfo":
					e.BaseURL = "http://user@127.0.0.1:8080/v1"
				case "query":
					e.BaseURL += "?secret=x"
				case "fragment":
					e.BaseURL += "#fragment"
				case "port":
					e.BaseURL = "http://127.0.0.1:0/v1"
				case "stopped":
					return e, errors.New("private error")
				}
				return e, err
			}, nil)(s)
			if _, err := RequestProfiles(s).Capture(); !errors.Is(err, ErrManagedUnavailable) {
				t.Fatalf("unsafe endpoint accepted: %v", err)
			}
			if len(*log) != 0 {
				t.Fatal("managed endpoint read manual credentials")
			}
		})
	}
}
func TestManagedCapturePreservesManualCleanupAndTaskOptions(t *testing.T) {
	s, _, _, keys := transactionalService(false)
	i := testInstance()
	s.cfg.ManagedRuntimes = []managedruntime.Instance{i}
	s.cfg.ManagedInstanceID = i.ID
	s.cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	s.cfg.Headers = map[string]string{"Private": "secret"}
	s.cfg.HealthPath = "/private"
	keys.getErr = errors.New("locked")
	s.cfg.Language = "en-US"
	s.cfg.PostProcessing.Enabled = true
	s.cfg.PostProcessing.BaseURL = "https://cleanup.example.test/v1"
	s.processKeys = &keyFake{log: &[]string{}, present: true, value: "cleanup-key"}
	WithManagedRuntimes(readyManagedEndpoint, nil)(s)
	p, err := RequestProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if p.Settings.Model != "served-model" || p.Settings.Language != "en-US" || len(p.Settings.Headers) != 0 || p.Settings.HealthPath != "" || p.STTCredential != "" || p.PostProcessingCredential != "cleanup-key" || p.Settings.PostProcessing.BaseURL != s.cfg.PostProcessing.BaseURL {
		t.Fatal("managed projection leaked transport or damaged task state")
	}
}
func TestUnavailableManagedCleanupDoesNotBlockRawTranscription(t *testing.T) {
	s, log, _, _ := transactionalService(false)
	s.cfg.PostProcessing.Enabled = true
	s.cfg.PostProcessing.ManagedInstanceID = "stopped-cleanup"
	s.processKeys = &keyFake{log: log, getErr: errors.New("locked")}
	p, err := RequestProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if !p.Settings.PostProcessing.Enabled || !errors.Is(p.PostProcessingUnavailable, ErrManagedUnavailable) || p.PostProcessingCredential != "" || len(*log) != 0 {
		t.Fatal("cleanup did not preserve raw fallback without keys")
	}
}
