package managedruntime

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

type ProviderID string

const NeMoSpeechCPP ProviderID = "nemo-speech-cpp"
const MaxInstances = 8
const LegacyInstanceID = "nemo-default"

type Instance struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Provider  ProviderID `json:"provider"`
	Model     string     `json:"model"`
	AutoStart bool       `json:"autoStart"`
}

type Contract struct {
	Role                 compatibility.Role   `json:"role"`
	CompatibilityProfile compatibility.ID     `json:"compatibilityProfile"`
	ModelProfile         modelprofile.ID      `json:"modelProfile"`
	Behavior             modelprofile.Profile `json:"behavior"`
}

type ProviderDescriptor struct {
	ID        ProviderID `json:"id"`
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Supported bool       `json:"supported"`
	Models    []Model    `json:"models"`
}

// Providers own their metadata, qualification and adapter construction. This
// registry contains implemented providers only, not future placeholders.
type provider interface {
	descriptor() ProviderDescriptor
	qualify(string, compatibility.Role) (Contract, error)
	newAdapter(string) runtimeAdapter
}

var providers = map[ProviderID]provider{NeMoSpeechCPP: nemoProvider{}}

func Qualify(id ProviderID, model string, role compatibility.Role) (Contract, error) {
	p, ok := providers[id]
	if !ok {
		return Contract{}, errors.New("Choose a supported managed runtime provider.")
	}
	return p.qualify(model, role)
}

var instanceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func ValidateInstance(i Instance) error {
	if !instanceIDPattern.MatchString(i.ID) || windowsReservedID(i.ID) {
		return errors.New("Choose a safe runtime instance ID of at most 64 characters.")
	}
	if !utf8.ValidString(i.Name) || len(i.Name) > 80 || strings.TrimSpace(i.Name) == "" || strings.ContainsAny(i.Name, "\x00\r\n") {
		return errors.New("Choose a runtime name of at most 80 UTF-8 bytes.")
	}
	p, ok := providers[i.Provider]
	if !ok {
		return errors.New("Choose a supported managed runtime provider.")
	}
	for _, model := range p.descriptor().Models {
		if model.ID == i.Model {
			return nil
		}
	}
	return errors.New("Choose a qualified model for this runtime provider.")
}
func windowsReservedID(id string) bool {
	switch id {
	case "con", "prn", "aux", "nul":
		return true
	}
	return len(id) == 4 && (strings.HasPrefix(id, "com") || strings.HasPrefix(id, "lpt")) && id[3] >= '1' && id[3] <= '9'
}

// ValidateInstances validates durable shape, not new-provider admission. Older
// inventories may contain copies; rejecting them here would prevent loading
// otherwise repairable settings. Manager reservations prohibit adding copies.
func ValidateInstances(instances []Instance) error {
	if len(instances) > MaxInstances {
		return errors.New("Too many managed runtime instances.")
	}
	seen := make(map[string]bool, len(instances))
	for _, i := range instances {
		if err := ValidateInstance(i); err != nil {
			return err
		}
		if seen[i.ID] {
			return errors.New("Runtime instance IDs must be unique.")
		}
		seen[i.ID] = true
	}
	return nil
}
func instanceDirectory(root, id string) string {
	if id == LegacyInstanceID {
		return filepath.Join(root, "managed-runtime")
	}
	return filepath.Join(root, "managed-runtimes", id)
}
