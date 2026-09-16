package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/settings"
)

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
	metadata := s.client.TestProfileMetadata(ctx, request.CompatibilityProfile, compatibility.Transcription, request.BaseURL, healthPath, key, request.Model, request.Headers)
	operationErr = ctx.Err()
	result.Reachable = metadata.Reachable
	result.Probe = ConnectionProbe(metadata.Probe)
	result.RequestedURL = metadata.RequestedURL
	result.HTTPStatus = metadata.HTTPStatus
	result.LatencyMilliseconds = metadata.LatencyMilliseconds
	result.ErrorKind = ConnectionErrorKind(metadata.ErrorKind)
	result.ModelPresence = ModelPresence(metadata.ModelPresence)
	result.ModelIDs = metadata.ModelIDs
	result.Models = metadata.Models
	result.ServerVersion = metadata.ServerVersion
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
	metadata := s.client.TestProfileMetadata(ctx, request.CompatibilityProfile, compatibility.Speech, request.BaseURL, "", key, request.Model, nil)
	result.Reachable = metadata.Reachable
	result.Probe = ConnectionProbe(metadata.Probe)
	result.RequestedURL = metadata.RequestedURL
	result.HTTPStatus = metadata.HTTPStatus
	result.LatencyMilliseconds = metadata.LatencyMilliseconds
	result.ErrorKind = ConnectionErrorKind(metadata.ErrorKind)
	result.ModelPresence = ModelPresence(metadata.ModelPresence)
	result.ModelIDs = metadata.ModelIDs
	result.Models = metadata.Models
	result.ServerVersion = metadata.ServerVersion
	operationErr = ctx.Err()
	return result
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
		if errors.Is(err, settings.ErrManagedUnavailable) {
			result.ErrorKind = ConnectionErrorRuntimeUnavailable
		} else if errors.Is(err, credential.ErrNotFound) {
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
	metadata := s.client.TestProfileMetadata(ctx, c.Details.CompatibilityProfile, "", c.Details.BaseURL, health, key, "", c.Details.Headers)
	operationErr = ctx.Err()
	result.Reachable = metadata.Reachable
	result.Probe = ConnectionProbe(metadata.Probe)
	result.RequestedURL = metadata.RequestedURL
	result.HTTPStatus = metadata.HTTPStatus
	result.LatencyMilliseconds = metadata.LatencyMilliseconds
	result.ErrorKind = ConnectionErrorKind(metadata.ErrorKind)
	result.ModelPresence = ModelPresence(metadata.ModelPresence)
	result.ModelIDs = metadata.ModelIDs
	result.Models = metadata.Models
	result.ServerVersion = metadata.ServerVersion
	return
}
