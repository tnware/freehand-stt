package savedconnection

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
)

// BuiltInID is the reserved, deterministic connection identity for one runtime.
// Keep it within SQLite's 64-character connection ID bound even for maximum
// length runtime IDs. Existing saved managed connection IDs are never replaced.
func BuiltInID(instanceID string) string {
	sum := sha256.Sum256([]byte(instanceID))
	return "builtin-" + hex.EncodeToString(sum[:28])
}

// BuiltIn projects only durable runtime identity and qualified uses. Readiness,
// live endpoint resolution and task selection remain with their existing owners.
func BuiltIn(instance managedruntime.Instance) Connection {
	name := strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, instance.Name))
	if name == "" {
		name = string(instance.Provider)
	}
	// Early GGML setup included the binary backend in the default name.
	// Keep identity/preferences intact, but never present that saved name as
	// execution status. Custom names and legacy aliases remain untouched.
	if instance.Provider == managedruntime.LlamaCPP && name == "llama.cpp (CPU)" {
		name = "llama.cpp"
	} else if instance.Provider == managedruntime.WhisperCPP && name == "whisper.cpp (CPU)" {
		name = "whisper.cpp"
	}
	c := Connection{
		ID: BuiltInID(instance.ID), Name: name, BuiltIn: true,
		Uses:    []Purpose{},
		Details: Details{ManagedInstanceID: instance.ID, AuthenticationMode: config.AuthenticationModeNone, Headers: map[string]string{}},
	}
	for _, p := range []Purpose{Voice, Transcription, Cleanup, Speech} {
		if _, err := managedruntime.Qualify(instance.Provider, instance.Model, Role(p)); err == nil {
			c.Uses = append(c.Uses, p)
		}
	}
	return c
}
