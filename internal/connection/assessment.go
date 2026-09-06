package connection

import (
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

type CheckStatus string

const (
	CheckPassed     CheckStatus = "passed"
	CheckAttention  CheckStatus = "attention"
	CheckUnverified CheckStatus = "unverified"
)

type CheckKind string

const (
	CheckConnection     CheckKind = "connection"
	CheckAuthentication CheckKind = "authentication"
	CheckModel          CheckKind = "model"
	CheckConfiguration  CheckKind = "configuration"
)

type Check struct {
	Kind    CheckKind   `json:"kind"`
	Status  CheckStatus `json:"status"`
	Summary string      `json:"summary"`
	Detail  string      `json:"detail"`
}

// assess projects bounded metadata and locally validated options. No check proves inference support.
func assess(r ConnectionResult, p savedconnection.Purpose, backend compatibility.ID, model string, options *modelsettings.Options, auth config.AuthenticationMode) []Check {
	connection := Check{Kind: CheckConnection, Status: CheckPassed, Summary: "Metadata received"}
	if r.Probe == ConnectionProbeHealth {
		connection.Summary = "Health endpoint responded"
	}
	if r.ErrorKind != "" {
		connection.Status = CheckAttention
		connection.Summary = "Connection needs attention"
		switch r.ErrorKind {
		case ConnectionErrorCredentialMissing:
			connection.Summary = "Check not sent"
			connection.Detail = "Add an API key in Connections, then check again."
		case ConnectionErrorCredentialUnavailable:
			connection.Summary = "Check not sent"
			connection.Detail = "Re-enter the key in Connections; the stored credential could not be read."
		case ConnectionErrorInvalidSettings, ConnectionErrorInvalidURL:
			connection.Summary = "Review connection settings"
			connection.Detail = "Check the backend profile, base URL, HTTP permission, and any custom health path or headers in Connections."
		case ConnectionErrorDNS:
			connection.Summary = "Server name not found"
			connection.Detail = "Check the hostname and your DNS or VPN connection."
		case ConnectionErrorTLS:
			connection.Summary = "TLS connection failed"
			connection.Detail = "Check the server certificate, hostname, and trusted certificate chain."
		case ConnectionErrorTimeout:
			connection.Summary = "Check timed out"
			connection.Detail = "The server did not complete the metadata request within 15 seconds. Check its address and availability."
		case ConnectionErrorHTTP:
			connection.Summary = "Server rejected the request"
			switch r.HTTPStatus {
			case 401:
				connection.Detail = "The server returned HTTP 401. Check the API key and authentication setting."
			case 403:
				connection.Detail = "The server returned HTTP 403. Check account permissions and server access rules."
			case 404, 405:
				connection.Detail = "The metadata route is unavailable. Check the base URL and backend profile; health-only servers cannot confirm model availability."
			case 429:
				connection.Detail = "The server is limiting requests. Wait and check its request or quota limits."
			default:
				connection.Detail = "Check server logs and the configured metadata route, then try again."
			}
		case ConnectionErrorResponse:
			connection.Summary = "Unexpected metadata response"
			connection.Detail = "The server answered, but did not return the expected metadata. Check the base URL and backend profile; a web page or login redirect is not a model list."
		case ConnectionErrorResponseTooLarge:
			connection.Summary = "Metadata response too large"
			connection.Detail = "The response exceeded the 1 MiB limit. Check that this URL serves metadata."
		default:
			connection.Summary = "Server could not be reached"
			connection.Detail = "Check that the server is running and its port is reachable through your network or firewall."
		}
	}
	authentication := Check{Kind: CheckAuthentication, Status: CheckUnverified, Summary: "Not verified", Detail: "No successful metadata response confirmed access."}
	if r.HTTPStatus == 401 || r.HTTPStatus == 403 || r.ErrorKind == ConnectionErrorCredentialMissing || r.ErrorKind == ConnectionErrorCredentialUnavailable {
		authentication.Status = CheckAttention
		authentication.Summary = "Review authentication"
		authentication.Detail = "Check the saved connection’s authentication method, key, and access permissions."
	} else if r.HTTPStatus >= 200 && r.HTTPStatus < 300 {
		authentication.Status = CheckPassed
		authentication.Summary = "Metadata request accepted"
		authentication.Detail = "This does not prove that inference routes accept the same credentials."
		if auth == config.AuthenticationModeNone && p != savedconnection.Cleanup {
			authentication.Summary = "Metadata accepted without a key"
		}
	}
	checks := []Check{connection, authentication}
	if options == nil || !savedconnection.ValidPurpose(p) {
		return checks
	}
	m := Check{Kind: CheckModel, Status: CheckUnverified, Summary: "Model could not be verified", Detail: "Run a successful model-list check to see whether this ID is advertised. Model execution is not tested."}
	if p == savedconnection.Transcription && backend == compatibility.WhisperCPP {
		m.Summary = "Server-loaded model"
		m.Detail = "whisper.cpp owns the loaded model. Its health endpoint does not identify or test that model."
	} else if strings.TrimSpace(model) == "" {
		m.Status = CheckAttention
		m.Summary = "Choose a model"
		m.Detail = "Select a saved or server-listed model, or enter the model ID supplied by your server."
	} else if r.ErrorKind == "" && r.Probe == ConnectionProbeModels {
		if r.ModelPresence == ModelPresenceListed {
			m.Status = CheckPassed
			m.Summary = "Selected model is advertised"
			m.Detail = "The model ID appears in metadata. This does not prove support for this feature."
		} else {
			m.Status = CheckAttention
			m.Summary = "Selected model is not advertised"
			m.Detail = "Refresh the list or check the ID with your server. An unlisted alias may still work; Freehand does not invoke it to find out."
		}
	} else if r.Probe == ConnectionProbeHealth {
		m.Detail = "A health check cannot confirm model IDs. Use the server’s model-list endpoint if it provides one."
	}
	checks = append(checks, m)
	c := Check{Kind: CheckConfiguration, Status: CheckPassed, Summary: "Options match the selected profiles", Detail: "These local checks do not prove that the deployed model implements every option."}
	// Endpoint and credentials are assessed separately. A placeholder permits option validation before model selection.
	id := strings.TrimSpace(model)
	if id == "" {
		id = "model-check"
	}
	d := savedconnection.Details{BaseURL: "https://metadata.invalid/v1", CompatibilityProfile: backend, AuthenticationMode: config.AuthenticationModeNone}
	e := modelsettings.Entry{Purpose: p, Model: id, Options: *options}
	if err := modelsettings.Validate(e, d); err != nil {
		c.Status = CheckAttention
		c.Summary = "Review model options"
		c.Detail = err.Error()
	} else if p == savedconnection.Speech && strings.TrimSpace(options.Voice) == "" {
		c.Status = CheckAttention
		c.Summary = "Choose a voice"
		c.Detail = "Enter a voice ID supported by the speech model. The standard model list cannot verify voices."
	} else if options.Profile == modelprofile.S1Mini {
		contract, err := modelprofile.Resolve(options.Profile, backend, compatibility.PostProcessing)
		if err == nil && !contract.Capabilities.CleanupDisableReasoning {
			c.Status = CheckUnverified
			c.Summary = "Server reasoning setting needs confirmation"
			c.Detail = "S1-mini requires reasoning off. This backend has no qualified request override; disable reasoning on the server. S1-mini supports English only."
		} else {
			c.Summary = "S1-mini requirements configured"
			c.Detail = "Requests disable reasoning. Cleanup is English-only; unknown language is assumed English."
		}
	}
	return append(checks, c)
}
