package config

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

const (
	MaxBaseURLBytes     = 2048
	MaxHealthPathBytes  = 1024
	MaxHeaderCount      = 32
	MaxHeaderNameBytes  = 256
	MaxHeaderValueBytes = 4096
	MaxHeaderBytes      = 16 * 1024
)

type AuthenticationMode string

const (
	AuthenticationModeAPIKey AuthenticationMode = "api-key"
	AuthenticationModeNone   AuthenticationMode = "none"
)

var headerNameRE = regexp.MustCompile(`^[!#$%&'*+\-.^_` + "`" + `|~0-9A-Za-z]+$`)

// validatePersistedSTTSettings permits an unconfigured audio-file task independently
// of Voice setup. Model choice can follow metadata discovery; recording/file
// request admission enforces the selected backend's model requirement.
func validatePersistedSTTSettings(s Settings) error {
	if s.ManagedInstanceID != "" {
		return validateManagedTransport(s.BaseURL, s.AllowInsecureHTTP, s.AuthenticationMode, s.HealthPath, s.Headers)
	}
	if s.BaseURL == "" && s.Model == "" {
		switch s.AuthenticationMode {
		case AuthenticationModeAPIKey, AuthenticationModeNone:
		default:
			return errors.New("authentication mode is invalid")
		}
		if len(s.HealthPath) > MaxHealthPathBytes {
			return fmt.Errorf("health path must be at most %d bytes", MaxHealthPathBytes)
		}
		if s.HealthPath != "" && (!strings.HasPrefix(s.HealthPath, "/") || strings.ContainsAny(s.HealthPath, "?#\r\n")) {
			return errors.New("health path must start with / and contain no query or fragment; it is appended to the base URL path")
		}
		return validateHeaders(s.Headers)
	}
	_, err := compatibility.Resolve(s.CompatibilityProfile, compatibility.Transcription)
	if err != nil {
		return err
	}
	return validateSTTConnection(s.BaseURL, s.AllowInsecureHTTP, s.AuthenticationMode, s.Model, s.HealthPath, s.Headers, false)
}

// ValidateSTTConnection validates only renderer-controlled values needed for
// an STT metadata probe. A model is optional so discovery can run before the
// user has selected one.
func ValidateSTTConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model, healthPath string, headers map[string]string) error {
	return validateSTTConnection(baseURL, allowInsecureHTTP, authenticationMode, model, healthPath, headers, false)
}

func validateSTTConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model, healthPath string, headers map[string]string, requireModel bool) error {
	if err := validateBaseURL(baseURL, allowInsecureHTTP, ""); err != nil {
		return err
	}
	switch authenticationMode {
	case AuthenticationModeAPIKey, AuthenticationModeNone:
	default:
		return errors.New("authentication mode is invalid")
	}
	if (requireModel && strings.TrimSpace(model) == "") || len(model) > 200 {
		return errors.New("model is required and must be at most 200 characters")
	}
	if len(healthPath) > MaxHealthPathBytes {
		return fmt.Errorf("health path must be at most %d bytes", MaxHealthPathBytes)
	}
	if healthPath != "" && (!strings.HasPrefix(healthPath, "/") || strings.ContainsAny(healthPath, "?#\r\n")) {
		return errors.New("health path must start with / and contain no query or fragment; it is appended to the base URL path")
	}
	return validateHeaders(headers)
}

func validateHeaders(headers map[string]string) error {
	blocked := map[string]bool{"authorization": true, "host": true, "content-length": true, "cookie": true, "connection": true, "keep-alive": true, "proxy-authenticate": true, "proxy-authorization": true, "te": true, "trailer": true, "transfer-encoding": true, "upgrade": true}
	if len(headers) > MaxHeaderCount {
		return fmt.Errorf("at most %d custom headers are allowed", MaxHeaderCount)
	}
	headerBytes := 0
	for k, v := range headers {
		if len(k) > MaxHeaderNameBytes {
			return fmt.Errorf("header name must be at most %d bytes", MaxHeaderNameBytes)
		}
		if len(v) > MaxHeaderValueBytes {
			return fmt.Errorf("header %q value must be at most %d bytes", k, MaxHeaderValueBytes)
		}
		headerBytes += len(k) + len(v)
		if headerBytes > MaxHeaderBytes {
			return fmt.Errorf("custom headers must total at most %d bytes", MaxHeaderBytes)
		}
		trimmedName := strings.TrimSpace(k)
		ck := http.CanonicalHeaderKey(trimmedName)
		lower := strings.ToLower(ck)
		secretLooking := strings.Contains(lower, "api-key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret")
		if k != trimmedName || ck == "" || !headerNameRE.MatchString(trimmedName) || blocked[lower] || secretLooking || !validHeaderValue(v) {
			return fmt.Errorf("header %q is not allowed", k)
		}
	}
	return nil
}

// ValidateTextToSpeechConnection validates only values needed for standard
// OpenAI-compatible model discovery. A model is optional until one is chosen.
func ValidateTextToSpeechConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model string) error {
	return validateTextToSpeechConnection(baseURL, allowInsecureHTTP, authenticationMode, model, false)
}

func validateTextToSpeechConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model string, requireModel bool) error {
	if err := validateBaseURL(baseURL, allowInsecureHTTP, "speech playback "); err != nil {
		return err
	}
	switch authenticationMode {
	case AuthenticationModeAPIKey, AuthenticationModeNone:
	default:
		return errors.New("speech playback authentication mode is invalid")
	}
	if (requireModel && strings.TrimSpace(model) == "") || len(model) > 200 {
		return errors.New("speech playback model is required and must be at most 200 characters")
	}
	return nil
}

// ValidatePostProcessingConnection validates only endpoint and optional model
// values needed for a metadata probe.
func ValidatePostProcessingConnection(baseURL string, allowInsecureHTTP bool, model string) error {
	return validatePostProcessingConnection(baseURL, allowInsecureHTTP, model, false)
}

func validatePostProcessingConnection(baseURL string, allowInsecureHTTP bool, model string, requireModel bool) error {
	if err := validateBaseURL(baseURL, allowInsecureHTTP, "post-processing "); err != nil {
		return fieldError("postProcessing.baseURL", err.Error(), err)
	}
	if (requireModel && strings.TrimSpace(model) == "") || len(model) > 200 {
		return fieldError("postProcessing.model", "Choose a cleanup model of at most 200 characters.", errors.New("post-processing model is required and must be at most 200 characters"))
	}
	return nil
}

func validateBaseURL(baseURL string, allowInsecureHTTP bool, prefix string) error {
	if len(baseURL) > MaxBaseURLBytes {
		return fmt.Errorf("%sbase URL must be at most %d bytes", prefix, MaxBaseURLBytes)
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%sbase URL must be an HTTP or HTTPS URL without credentials, query, or fragment", prefix)
	}
	if u.Scheme == "http" && !allowInsecureHTTP {
		return fmt.Errorf("%sbase URL uses insecure HTTP; enable Allow insecure HTTP to continue", prefix)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("%sbase URL must use HTTPS unless insecure HTTP is explicitly enabled", prefix)
	}
	return nil
}

func validHeaderValue(value string) bool {
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b == '\t' || (b >= 0x20 && b != 0x7f) {
			continue
		}
		return false
	}
	return true
}
