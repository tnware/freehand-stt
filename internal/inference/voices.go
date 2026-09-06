package inference

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

const MaxDiscoveredVoices = 500

type VoiceScope string

const (
	VoiceScopeModel  VoiceScope = "model"
	VoiceScopeServer VoiceScope = "server"
)

type Voice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
}

type VoicesResult struct {
	Voices              []Voice    `json:"voices"`
	Scope               VoiceScope `json:"scope"`
	ErrorKind           string     `json:"errorKind"`
	HTTPStatus          int        `json:"httpStatus"`
	LatencyMilliseconds int64      `json:"latencyMilliseconds"`
	Truncated           bool       `json:"truncated"`
}

// ListVoices reads bounded provider metadata, never loads or invokes a model.
// Speaches model metadata is preferred; older servers and aliases can use its
// server-wide voice list without claiming that it belongs to the selected model.
func (c *Client) ListVoices(ctx context.Context, backend compatibility.ID, base, key, model string) (result VoicesResult) {
	start := time.Now()
	defer func() { result.LatencyMilliseconds = time.Since(start).Milliseconds() }()
	contract, err := compatibility.Resolve(backend, compatibility.Speech)
	if err != nil || !contract.Capabilities.VoiceDiscovery {
		result.ErrorKind = "unsupported"
		return
	}
	get := func(route string) ([]byte, bool) {
		target, err := endpoint(base, route)
		if err != nil {
			result.ErrorKind = "invalid_url"
			return nil, false
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			result.ErrorKind = "invalid_url"
			return nil, false
		}
		req.Header.Set("Accept", "application/json")
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			result.ErrorKind = metadataNetworkErrorKind(err)
			return nil, false
		}
		defer resp.Body.Close()
		result.HTTPStatus = resp.StatusCode
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
		if err != nil {
			result.ErrorKind = "response"
			return nil, false
		}
		if len(body) > maxResponse {
			result.ErrorKind = "response_too_large"
			return nil, false
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			result.ErrorKind = "http"
			return nil, false
		}
		return body, true
	}
	var voices []json.RawMessage
	if backend == compatibility.Speaches && model != "" {
		body, ok := get("models")
		if !ok && result.HTTPStatus != 404 && result.HTTPStatus != 405 {
			return
		}
		if ok {
			var catalog struct {
				Data []struct {
					ID     string            `json:"id"`
					Voices []json.RawMessage `json:"voices"`
				} `json:"data"`
			}
			if json.Unmarshal(body, &catalog) != nil || catalog.Data == nil {
				result.ErrorKind = "response"
				return
			}
			for _, m := range catalog.Data {
				if m.ID == model && m.Voices != nil {
					voices = m.Voices
					result.Scope = VoiceScopeModel
					break
				}
			}
		}
	}
	if voices == nil {
		result.ErrorKind = ""
		body, ok := get("audio/voices")
		if !ok {
			return
		}
		var list struct {
			Voices []json.RawMessage `json:"voices"`
		}
		if json.Unmarshal(body, &list) != nil || list.Voices == nil {
			result.ErrorKind = "response"
			return
		}
		voices = list.Voices
		result.Scope = VoiceScopeServer
	}
	seen := map[string]bool{}
	for _, raw := range voices {
		var v Voice
		if len(raw) > 0 && raw[0] == '"' {
			if backend != compatibility.KokoroFastAPI || json.Unmarshal(raw, &v.ID) != nil {
				result.ErrorKind = "response"
				result.Voices = nil
				return
			}
		} else if json.Unmarshal(raw, &v) != nil {
			result.ErrorKind = "response"
			result.Voices = nil
			return
		}
		if !validVoiceID(v.ID, key) || seen[v.ID] {
			continue
		}
		if len(result.Voices) >= MaxDiscoveredVoices {
			result.Truncated = true
			continue
		}
		seen[v.ID] = true
		v.Name = safePeerString(v.Name, key)
		if v.Name == "" {
			v.Name = v.ID
		}
		v.Language = safePeerString(v.Language, key)
		result.Voices = append(result.Voices, v)
	}
	return
}
func validVoiceID(id, key string) bool {
	if id == "" || len(id) > 200 || !utf8.ValidString(id) || strings.TrimSpace(id) != id || (key != "" && strings.Contains(id, key)) {
		return false
	}
	for _, r := range id {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
