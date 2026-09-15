package inference

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

// ModelMetadata contains bounded advertised facts, never inferred model behavior.
type ModelMetadata struct {
	ID         string `json:"id"`
	Capability string `json:"capability"`
	Device     string `json:"device"`
}

// TestProfileMetadata narrows NeMo's mixed inventory to this task. An absent
// capability remains unknown; metadata never enables an unqualified profile.
func (c *Client) TestProfileMetadata(ctx context.Context, backend compatibility.ID, role compatibility.Role, base, health, key, model string, headers map[string]string) MetadataResult {
	started := time.Now()
	result := c.TestMetadata(ctx, base, health, key, model, headers)
	if backend != compatibility.NeMoSpeechV1 {
		result.Models = nil
		result.ServerVersion = ""
		return result
	}
	if result.ErrorKind != "" {
		return result
	}
	if health == "" {
		result.ModelIDs = nil
		result.ModelPresence = "not-listed"
		for _, item := range result.Models {
			if item.Capability != "" && item.Capability != string(role) {
				continue
			}
			result.ModelIDs = append(result.ModelIDs, item.ID)
			if item.ID == model {
				result.ModelPresence = "listed"
			}
		}
		// NeMo publishes its version at /health beside /v1. Preserve an
		// explicitly configured reverse-proxy prefix and never change origin.
		if u, err := url.Parse(base); err == nil && strings.HasSuffix(strings.TrimRight(u.Path, "/"), "/v1") {
			u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/v1")
			u.RawPath = ""
			probe, cancel := context.WithTimeout(ctx, 2*time.Second)
			version := c.TestMetadata(probe, u.String(), "/health", key, "", headers)
			cancel()
			if version.ErrorKind == "" {
				result.ServerVersion = version.ServerVersion
			}
		}
	}
	result.LatencyMilliseconds = time.Since(started).Milliseconds()
	return result
}

func advertisedCapability(value string) string {
	switch value {
	case "transcription", "speech", "diarization", "translation", "speech-translation", "speech-to-speech":
		return value
	case "":
		return ""
	default:
		return "unknown"
	}
}
