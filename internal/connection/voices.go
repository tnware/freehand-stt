package connection

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

type VoiceListRequest struct {
	ConnectionID string `json:"connectionID"`
	Model        string `json:"model"`
}

// ListSpeechVoices captures a saved connection and its credential together.
// The renderer supplies only its selected connection/model, never a key or URL.
func (s *Service) ListSpeechVoices(request VoiceListRequest) (result inference.VoicesResult) {
	started := time.Now()
	s.log().Info("voice discovery started")
	defer func() {
		s.log().Info("voice discovery completed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", result.ErrorKind, "http_status", result.HTTPStatus, "voice_count", len(result.Voices))
	}()
	if s.savedConnections == nil || request.ConnectionID == "" || len(request.ConnectionID) > 200 || len(request.Model) > 200 || !utf8.ValidString(request.Model) || strings.TrimSpace(request.Model) != request.Model {
		result.ErrorKind = "invalid_settings"
		return
	}
	for _, r := range request.Model {
		if unicode.IsControl(r) {
			result.ErrorKind = "invalid_settings"
			return
		}
	}
	c, key, err := s.savedConnections.ResolveSavedConnection(request.ConnectionID)
	if err != nil {
		result.ErrorKind = "credential_unavailable"
		if errors.Is(err, credential.ErrNotFound) {
			result.ErrorKind = "credential_missing"
		}
		return
	}
	defer func() { key = "" }()
	if !c.Supports(savedconnection.Speech) || savedconnection.ValidateUses(c.Uses, c.Details) != nil {
		result.ErrorKind = "invalid_settings"
		return
	}
	if c.Details.AuthenticationMode == config.AuthenticationModeNone {
		key = ""
	}
	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	return s.client.ListVoices(ctx, c.Details.CompatibilityProfile, c.Details.BaseURL, key, request.Model)
}
