package connection

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ConnectionTestRequest struct {
	CompatibilityProfile compatibility.ID          `json:"compatibilityProfile"`
	BaseURL              string                    `json:"baseURL"`
	AllowInsecureHTTP    bool                      `json:"allowInsecureHTTP"`
	AuthenticationMode   config.AuthenticationMode `json:"authenticationMode"`
	Model                string                    `json:"model"`
	HealthPath           string                    `json:"healthPath,omitempty"`
	Headers              map[string]string         `json:"headers,omitempty"`
	CredentialDraft      string                    `json:"credentialDraft,omitempty"`
}

type PostProcessingConnectionTestRequest struct {
	CompatibilityProfile compatibility.ID `json:"compatibilityProfile"`
	BaseURL              string           `json:"baseURL"`
	AllowInsecureHTTP    bool             `json:"allowInsecureHTTP"`
	Model                string           `json:"model"`
	CredentialDraft      string           `json:"credentialDraft,omitempty"`
}

type TextToSpeechConnectionTestRequest struct {
	CompatibilityProfile compatibility.ID          `json:"compatibilityProfile"`
	BaseURL              string                    `json:"baseURL"`
	AllowInsecureHTTP    bool                      `json:"allowInsecureHTTP"`
	AuthenticationMode   config.AuthenticationMode `json:"authenticationMode"`
	Model                string                    `json:"model"`
	CredentialDraft      string                    `json:"credentialDraft,omitempty"`
}

type ConnectionProbe string

const (
	ConnectionProbeHealth ConnectionProbe = "health"
	ConnectionProbeModels ConnectionProbe = "models"
)

type ModelPresence string

const (
	ModelPresenceUnavailable ModelPresence = "unavailable"
	ModelPresenceListed      ModelPresence = "listed"
	ModelPresenceNotListed   ModelPresence = "not-listed"
)

type ConnectionErrorKind string

const (
	ConnectionErrorCredentialMissing     ConnectionErrorKind = "credential_missing"
	ConnectionErrorCredentialUnavailable ConnectionErrorKind = "credential_unavailable"
	ConnectionErrorDNS                   ConnectionErrorKind = "dns"
	ConnectionErrorTLS                   ConnectionErrorKind = "tls"
	ConnectionErrorHTTP                  ConnectionErrorKind = "http"
	ConnectionErrorResponseTooLarge      ConnectionErrorKind = "response_too_large"
	ConnectionErrorResponse              ConnectionErrorKind = "response"
	ConnectionErrorInvalidURL            ConnectionErrorKind = "invalid_url"
	ConnectionErrorInvalidSettings       ConnectionErrorKind = "invalid_settings"
	ConnectionErrorTimeout               ConnectionErrorKind = "timeout"
	ConnectionErrorNetwork               ConnectionErrorKind = "network"
)

type ConnectionResult struct {
	Reachable           bool                `json:"reachable"`
	Probe               ConnectionProbe     `json:"probe"`
	RequestedURL        string              `json:"requestedURL"`
	HTTPStatus          int                 `json:"httpStatus"`
	LatencyMilliseconds int64               `json:"latencyMilliseconds"`
	ErrorKind           ConnectionErrorKind `json:"errorKind"`
	CheckedAt           time.Time           `json:"checkedAt"`
	ModelPresence       ModelPresence       `json:"modelPresence"`
	ModelIDs            []string            `json:"modelIDs"`
}

type Service struct {
	keys        credential.Store
	processKeys credential.Store
	ttsKeys     credential.Store
	client      *inference.Client
	logger      *slog.Logger
	lifecycleMu sync.RWMutex
	rootContext context.Context
	rootCancel  context.CancelFunc
	closed      atomic.Bool
}

func NewService(keys, processKeys, ttsKeys credential.Store, client *inference.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	return &Service{keys: keys, processKeys: processKeys, ttsKeys: ttsKeys, client: client, logger: logger.With("component", "connection")}
}

func (s *Service) operationContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	s.lifecycleMu.RLock()
	root := s.rootContext
	s.lifecycleMu.RUnlock()
	if root == nil {
		root = context.Background()
	}
	ctx, cancel := context.WithTimeout(root, timeout)
	if s.closed.Load() {
		cancel()
	}
	return ctx, cancel
}

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return diagnostics.DiscardLogger()
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.closed.Store(false)
	s.lifecycleMu.Lock()
	s.rootContext, s.rootCancel = context.WithCancel(ctx)
	s.lifecycleMu.Unlock()
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.closed.Store(true)
	s.lifecycleMu.Lock()
	cancel := s.rootCancel
	s.rootCancel = nil
	s.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// Endpoint metadata bindings. Each request includes only values used by its
// metadata probe; unrelated settings cannot invalidate the operation.

// TestConnection probes STT health or model discovery without invoking a model.
func (s *Service) TestConnection(request ConnectionTestRequest) (result ConnectionResult) {
	checkedAt := time.Now().UTC()
	probe, requestedURL, targetErr := inference.MetadataTarget(request.BaseURL, request.HealthPath)
	validationError := ""
	if len(request.CredentialDraft) > settings.MaxAPIKeyBytes {
		validationError = fmt.Sprintf("API key must be at most %d bytes", settings.MaxAPIKeyBytes)
	} else if _, err := compatibility.Resolve(request.CompatibilityProfile, compatibility.Transcription); err != nil {
		validationError = "compatibility_profile"
	} else if err := config.ValidateSTTConnection(request.BaseURL, request.AllowInsecureHTTP, request.AuthenticationMode, request.Model, request.HealthPath, request.Headers); err != nil {
		validationError = safeConnectionValidationError(err)
	}
	result = ConnectionResult{
		Probe:         ConnectionProbe(probe),
		RequestedURL:  requestedURL,
		CheckedAt:     checkedAt,
		ModelPresence: ModelPresenceUnavailable,
	}
	server := connectionServer(requestedURL)
	credentialSource := "none"
	if request.AuthenticationMode == config.AuthenticationModeAPIKey {
		credentialSource = "stored"
	}
	if request.AuthenticationMode == config.AuthenticationModeAPIKey && strings.TrimSpace(request.CredentialDraft) != "" {
		credentialSource = "draft"
	}
	s.log().Info("connection test started",
		"server", server,
		"probe", result.Probe,
		"credential_source", credentialSource,
	)
	defer func() {
		s.log().Info("connection test completed",
			"server", server,
			"probe", result.Probe,
			"credential_source", credentialSource,
			"reachable", result.Reachable,
			"http_status", result.HTTPStatus,
			"latency_ms", result.LatencyMilliseconds,
			"error_kind", result.ErrorKind,
			"validation_error", validationError,
			"model_presence", result.ModelPresence,
			"model_count", len(result.ModelIDs),
		)
	}()
	if validationError != "" {
		result.ErrorKind = ConnectionErrorInvalidSettings
		return result
	}
	if targetErr != nil {
		result.ErrorKind = ConnectionErrorInvalidURL
		return result
	}
	key := ""
	if request.AuthenticationMode == config.AuthenticationModeAPIKey {
		key = request.CredentialDraft
	}
	if request.AuthenticationMode == config.AuthenticationModeAPIKey && strings.TrimSpace(key) == "" {
		var e error
		key, e = s.keys.Get()
		if e != nil {
			if errors.Is(e, credential.ErrNotFound) {
				result.ErrorKind = ConnectionErrorCredentialMissing
			} else {
				result.ErrorKind = ConnectionErrorCredentialUnavailable
			}
			return result
		}
	}
	defer func() { key = "" }()
	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	metadata := s.client.TestMetadata(ctx, request.BaseURL, request.HealthPath, key, request.Model, request.Headers)
	result.Reachable = metadata.Reachable
	result.Probe = ConnectionProbe(metadata.Probe)
	result.RequestedURL = metadata.RequestedURL
	result.HTTPStatus = metadata.HTTPStatus
	result.LatencyMilliseconds = metadata.LatencyMilliseconds
	result.ErrorKind = ConnectionErrorKind(metadata.ErrorKind)
	result.ModelPresence = ModelPresence(metadata.ModelPresence)
	result.ModelIDs = metadata.ModelIDs
	return result
}

// TestPostProcessingConnection probes post-processing model discovery without
// sending transcript content or invoking a model.
func (s *Service) TestPostProcessingConnection(request PostProcessingConnectionTestRequest) (result ConnectionResult) {
	checkedAt := time.Now().UTC()
	probe, requestedURL, targetErr := inference.MetadataTarget(request.BaseURL, "")
	validationError := ""
	if len(request.CredentialDraft) > settings.MaxAPIKeyBytes {
		validationError = fmt.Sprintf("post-processing API key must be at most %d bytes", settings.MaxAPIKeyBytes)
	} else if _, err := compatibility.Resolve(request.CompatibilityProfile, compatibility.PostProcessing); err != nil {
		validationError = "compatibility_profile"
	} else if err := config.ValidatePostProcessingConnection(request.BaseURL, request.AllowInsecureHTTP, request.Model); err != nil {
		validationError = safeConnectionValidationError(err)
	}
	result = ConnectionResult{
		Probe: ConnectionProbe(probe), RequestedURL: requestedURL,
		CheckedAt: checkedAt, ModelPresence: ModelPresenceUnavailable,
	}
	server := connectionServer(requestedURL)
	credentialSource := "none"
	if strings.TrimSpace(request.CredentialDraft) != "" {
		credentialSource = "draft"
	} else if s.processKeys != nil && s.processKeys.Configured() {
		credentialSource = "stored"
	}
	s.log().Info("post-processing connection test started", "server", server, "probe", result.Probe, "credential_source", credentialSource)
	defer func() {
		s.log().Info("post-processing connection test completed",
			"server", server, "probe", result.Probe, "credential_source", credentialSource,
			"reachable", result.Reachable, "http_status", result.HTTPStatus,
			"latency_ms", result.LatencyMilliseconds, "error_kind", result.ErrorKind,
			"validation_error", validationError, "model_presence", result.ModelPresence,
			"model_count", len(result.ModelIDs),
		)
	}()
	if validationError != "" {
		result.ErrorKind = ConnectionErrorInvalidSettings
		return result
	}
	if targetErr != nil {
		result.ErrorKind = ConnectionErrorInvalidURL
		return result
	}
	key := request.CredentialDraft
	if strings.TrimSpace(key) == "" && s.processKeys != nil {
		stored, err := s.processKeys.Get()
		if err == nil {
			key = stored
		} else if !errors.Is(err, credential.ErrNotFound) {
			result.ErrorKind = ConnectionErrorCredentialUnavailable
			return result
		}
	}
	defer func() { key = "" }()
	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	metadata := s.client.TestMetadata(ctx, request.BaseURL, "", key, request.Model, nil)
	result.Reachable = metadata.Reachable
	result.Probe = ConnectionProbe(metadata.Probe)
	result.RequestedURL = metadata.RequestedURL
	result.HTTPStatus = metadata.HTTPStatus
	result.LatencyMilliseconds = metadata.LatencyMilliseconds
	result.ErrorKind = ConnectionErrorKind(metadata.ErrorKind)
	result.ModelPresence = ModelPresence(metadata.ModelPresence)
	result.ModelIDs = metadata.ModelIDs
	return result
}

// TestTextToSpeechConnection discovers models without synthesizing speech.
// Voice discovery is intentionally absent because the compatible API does not
// define a portable endpoint for it.
func (s *Service) TestTextToSpeechConnection(request TextToSpeechConnectionTestRequest) (result ConnectionResult) {
	checkedAt := time.Now().UTC()
	probe, requestedURL, targetErr := inference.MetadataTarget(request.BaseURL, "")
	validationError := ""
	if len(request.CredentialDraft) > settings.MaxAPIKeyBytes {
		validationError = fmt.Sprintf("speech playback API key must be at most %d bytes", settings.MaxAPIKeyBytes)
	} else if _, err := compatibility.Resolve(request.CompatibilityProfile, compatibility.Speech); err != nil {
		validationError = "compatibility_profile"
	} else if err := config.ValidateTextToSpeechConnection(request.BaseURL, request.AllowInsecureHTTP, request.AuthenticationMode, request.Model); err != nil {
		validationError = safeConnectionValidationError(err)
	}
	result = ConnectionResult{Probe: ConnectionProbe(probe), RequestedURL: requestedURL, CheckedAt: checkedAt, ModelPresence: ModelPresenceUnavailable}
	server := connectionServer(requestedURL)
	credentialSource := "none"
	if request.AuthenticationMode == config.AuthenticationModeAPIKey {
		credentialSource = "stored"
		if strings.TrimSpace(request.CredentialDraft) != "" {
			credentialSource = "draft"
		}
	}
	s.log().Info("speech playback connection test started", "server", server, "probe", result.Probe, "credential_source", credentialSource)
	defer func() {
		s.log().Info("speech playback connection test completed",
			"server", server, "probe", result.Probe, "credential_source", credentialSource,
			"reachable", result.Reachable, "http_status", result.HTTPStatus,
			"latency_ms", result.LatencyMilliseconds, "error_kind", result.ErrorKind,
			"validation_error", validationError, "model_presence", result.ModelPresence,
			"model_count", len(result.ModelIDs),
		)
	}()
	if validationError != "" {
		result.ErrorKind = ConnectionErrorInvalidSettings
		return result
	}
	if targetErr != nil {
		result.ErrorKind = ConnectionErrorInvalidURL
		return result
	}
	key := ""
	if request.AuthenticationMode == config.AuthenticationModeAPIKey {
		key = request.CredentialDraft
		if strings.TrimSpace(key) == "" {
			if s.ttsKeys == nil {
				result.ErrorKind = ConnectionErrorCredentialMissing
				return result
			}
			stored, err := s.ttsKeys.Get()
			if err != nil {
				if errors.Is(err, credential.ErrNotFound) {
					result.ErrorKind = ConnectionErrorCredentialMissing
				} else {
					result.ErrorKind = ConnectionErrorCredentialUnavailable
				}
				return result
			}
			key = stored
		}
	}
	defer func() { key = "" }()
	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	metadata := s.client.TestMetadata(ctx, request.BaseURL, "", key, request.Model, nil)
	result.Reachable = metadata.Reachable
	result.Probe = ConnectionProbe(metadata.Probe)
	result.RequestedURL = metadata.RequestedURL
	result.HTTPStatus = metadata.HTTPStatus
	result.LatencyMilliseconds = metadata.LatencyMilliseconds
	result.ErrorKind = ConnectionErrorKind(metadata.ErrorKind)
	result.ModelPresence = ModelPresence(metadata.ModelPresence)
	result.ModelIDs = metadata.ModelIDs
	return result
}

func safeConnectionValidationError(err error) string {
	message := err.Error()
	switch {
	case strings.HasPrefix(message, "base URL"),
		strings.HasPrefix(message, "post-processing base URL"),
		strings.HasPrefix(message, "speech playback base URL"),
		strings.HasPrefix(message, "model "),
		strings.HasPrefix(message, "post-processing model "),
		strings.HasPrefix(message, "speech playback model "),
		strings.HasPrefix(message, "language "),
		strings.HasPrefix(message, "health path"),
		strings.HasPrefix(message, "maximum duration"),
		strings.HasPrefix(message, "voice activity detection"),
		strings.HasPrefix(message, "silence splitting"),
		strings.HasPrefix(message, "segment target"),
		strings.HasPrefix(message, "segment silence"),
		strings.HasPrefix(message, "microphone identifier"):
		return message
	case strings.Contains(message, "shortcut"):
		return "one or more shortcuts are invalid"
	case strings.Contains(message, "header"):
		return "one or more custom headers are invalid"
	default:
		return "settings validation failed"
	}
}

func connectionServer(requestedURL string) string {
	parsed, err := url.Parse(requestedURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
