package connection

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelsettings"
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
