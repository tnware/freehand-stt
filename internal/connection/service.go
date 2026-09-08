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
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ConnectionTestRequest struct {
	Options              *modelsettings.Options    `json:"options,omitempty"`
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
	Options              *modelsettings.Options `json:"options,omitempty"`
	CompatibilityProfile compatibility.ID       `json:"compatibilityProfile"`
	BaseURL              string                 `json:"baseURL"`
	AllowInsecureHTTP    bool                   `json:"allowInsecureHTTP"`
	Model                string                 `json:"model"`
	CredentialDraft      string                 `json:"credentialDraft,omitempty"`
}

type TextToSpeechConnectionTestRequest struct {
	Options              *modelsettings.Options    `json:"options,omitempty"`
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
	Checks              []Check             `json:"checks,omitempty"`
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

type SavedConnectionSource interface {
	ResolveSavedConnection(string) (savedconnection.Connection, string, error)
}

type Service struct {
	savedConnections SavedConnectionSource
	keys             credential.Store
	processKeys      credential.Store
	ttsKeys          credential.Store
	client           *inference.Client
	logger           *slog.Logger
	lifecycleMu      sync.RWMutex
	rootContext      context.Context
	rootCancel       context.CancelFunc
	closed           atomic.Bool
}

func NewService(keys, processKeys, ttsKeys credential.Store, client *inference.Client, logger *slog.Logger, sources ...SavedConnectionSource) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	s := &Service{keys: keys, processKeys: processKeys, ttsKeys: ttsKeys, client: client, logger: logger.With("component", "connection")}
	if len(sources) > 0 {
		s.savedConnections = sources[0]
	}
	return s
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

// metadataLogOutcome uses the probe's bounded result taxonomy. Capture ctxErr
// before the operation's deferred cancel: cleanup must not turn a failure into
// cancellation, and cancellation after a successful result must not hide it.
// This affects diagnostics only, not the renderer's existing result contract.
func metadataLogOutcome(errorKind string, ctxErr error) (slog.Level, string, string) {
	if errorKind == "" {
		return slog.LevelInfo, "completed", ""
	}
	if errors.Is(ctxErr, context.Canceled) {
		return slog.LevelInfo, "cancelled", diagnostics.ErrorKind(ctxErr)
	}
	switch errorKind {
	case "invalid_settings", "invalid_url", "credential_missing", "unsupported":
		return slog.LevelWarn, "failed", errorKind
	default:
		return slog.LevelError, "failed", errorKind
	}
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
	started := time.Now()
	var operationErr error
	defer func() {
		result.Checks = assess(result, savedconnection.Transcription, request.CompatibilityProfile, request.Model, request.Options, request.AuthenticationMode)
	}()
	checkedAt := time.Now().UTC()
	healthPath := compatibility.TranscriptionHealthPath(request.CompatibilityProfile, request.HealthPath)
	probe, requestedURL, targetErr := inference.MetadataTarget(request.BaseURL, healthPath)
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
		level, outcome, errorKind := metadataLogOutcome(string(result.ErrorKind), operationErr)
		s.log().Log(context.Background(), level, "connection test "+outcome,
			"outcome", outcome, "duration_ms", time.Since(started).Milliseconds(),
			"server", server,
			"probe", result.Probe,
			"credential_source", credentialSource,
			"reachable", result.Reachable,
			"http_status", result.HTTPStatus,
			"latency_ms", result.LatencyMilliseconds,
			"error_kind", errorKind,
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
	metadata := s.client.TestMetadata(ctx, request.BaseURL, healthPath, key, request.Model, request.Headers)
	operationErr = ctx.Err()
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
	started := time.Now()
	var operationErr error
	defer func() {
		result.Checks = assess(result, savedconnection.Cleanup, request.CompatibilityProfile, request.Model, request.Options, config.AuthenticationModeNone)
	}()
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
		level, outcome, errorKind := metadataLogOutcome(string(result.ErrorKind), operationErr)
		s.log().Log(context.Background(), level, "post-processing connection test "+outcome,
			"outcome", outcome, "duration_ms", time.Since(started).Milliseconds(),
			"server", server, "probe", result.Probe, "credential_source", credentialSource,
			"reachable", result.Reachable, "http_status", result.HTTPStatus,
			"latency_ms", result.LatencyMilliseconds, "error_kind", errorKind,
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
	operationErr = ctx.Err()
	return result
}

// TestTextToSpeechConnection discovers models without synthesizing speech.
// Voice discovery is intentionally absent because the compatible API does not
// define a portable endpoint for it.
func (s *Service) TestTextToSpeechConnection(request TextToSpeechConnectionTestRequest) (result ConnectionResult) {
	started := time.Now()
	var operationErr error
	defer func() {
		result.Checks = assess(result, savedconnection.Speech, request.CompatibilityProfile, request.Model, request.Options, request.AuthenticationMode)
	}()
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
		level, outcome, errorKind := metadataLogOutcome(string(result.ErrorKind), operationErr)
		s.log().Log(context.Background(), level, "speech playback connection test "+outcome,
			"outcome", outcome, "duration_ms", time.Since(started).Milliseconds(),
			"server", server, "probe", result.Probe, "credential_source", credentialSource,
			"reachable", result.Reachable, "http_status", result.HTTPStatus,
			"latency_ms", result.LatencyMilliseconds, "error_kind", errorKind,
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
	operationErr = ctx.Err()
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

// TestSavedConnection checks an explicitly saved connection without selecting it or invoking a model.
func (s *Service) TestSavedConnection(id string) (result ConnectionResult) {
	started := time.Now()
	var operationErr error
	useCount := 0
	s.log().Info("saved connection test started")
	defer func() {
		level, outcome, errorKind := metadataLogOutcome(string(result.ErrorKind), operationErr)
		s.log().Log(context.Background(), level, "saved connection test "+outcome,
			"outcome", outcome, "duration_ms", time.Since(started).Milliseconds(),
			"use_count", useCount, "reachable", result.Reachable, "error_kind", errorKind, "latency_ms", result.LatencyMilliseconds)
	}()
	auth := config.AuthenticationModeNone
	defer func() { result.Checks = assess(result, "", compatibility.Generic, "", nil, auth) }()
	result.CheckedAt = time.Now().UTC()
	result.ModelPresence = ModelPresenceUnavailable
	if s.savedConnections == nil {
		result.ErrorKind = ConnectionErrorInvalidSettings
		return
	}
	c, key, err := s.savedConnections.ResolveSavedConnection(id)
	if err != nil {
		result.ErrorKind = ConnectionErrorCredentialUnavailable
		if errors.Is(err, credential.ErrNotFound) {
			result.ErrorKind = ConnectionErrorCredentialMissing
		}
		return
	}
	auth = c.Details.AuthenticationMode
	useCount = len(c.Uses)
	defer func() { key = "" }()
	if err = savedconnection.ValidateUses(c.Uses, c.Details); err != nil {
		result.ErrorKind = ConnectionErrorInvalidSettings
		return
	}
	if c.Details.AuthenticationMode == config.AuthenticationModeNone {
		key = ""
	}
	health := ""
	if c.Supports(savedconnection.Transcription) || c.Supports(savedconnection.Voice) {
		health = compatibility.TranscriptionHealthPath(c.Details.CompatibilityProfile, c.Details.HealthPath)
	}
	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	metadata := s.client.TestMetadata(ctx, c.Details.BaseURL, health, key, "", c.Details.Headers)
	operationErr = ctx.Err()
	result.Reachable = metadata.Reachable
	result.Probe = ConnectionProbe(metadata.Probe)
	result.RequestedURL = metadata.RequestedURL
	result.HTTPStatus = metadata.HTTPStatus
	result.LatencyMilliseconds = metadata.LatencyMilliseconds
	result.ErrorKind = ConnectionErrorKind(metadata.ErrorKind)
	result.ModelPresence = ModelPresence(metadata.ModelPresence)
	result.ModelIDs = metadata.ModelIDs
	return
}
