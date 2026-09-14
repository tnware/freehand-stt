// Package managedruntime owns the opt-in, isolated NeMo-Speech.cpp runtime.
// It never reads remote endpoint configuration or credentials.
package managedruntime

import (
	"errors"

	"github.com/tnware/freehand-stt/internal/modelprofile"
)

const Version = "0.1.0"

type Preferences struct {
	Enabled  bool   `json:"enabled"`
	Model    string `json:"model"`
	Realtime bool   `json:"realtime"`
}

func Defaults() Preferences           { return Preferences{Model: "nemotron-3.5", Realtime: true} }
func (p Preferences) Validate() error { return Validate(p) }
func Validate(p Preferences) error {
	q, ok := qualified[p.Model]
	if !ok {
		return errors.New("Choose a supported managed speech model.")
	}
	if p.Realtime && !q.Realtime {
		return errors.New("This model does not support realtime dictation.")
	}
	return nil
}

type Endpoint struct {
	Enabled  bool
	BaseURL  string
	Model    string
	Realtime bool
	Profile  string
}
type Model struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SizeBytes   int64  `json:"sizeBytes"`
	Installed   bool   `json:"installed"`
	Recommended bool   `json:"recommended"`
	Realtime    bool   `json:"realtime"`
	Profile     string `json:"profile"`
	// Contracts is the authoritative per-role capability catalog for this model.
	Contracts []Contract `json:"contracts"`
	// Behavior is the resolved model/backend contract, independent of manual settings.
	Behavior *modelprofile.Profile `json:"behavior,omitempty"`
}

// Operation is an admitted asynchronous operation, not a request receipt.
// Outcome is running, succeeded, cancelled, or failed. The latest result stays
// in status until another operation is admitted; ID is unique for this process.
type Operation struct {
	ID      uint64 `json:"id"`
	Kind    string `json:"kind"`
	Model   string `json:"model"`
	Outcome string `json:"outcome"`
	Error   string `json:"error"`
}

// AcquisitionProgress contains only bounded counters and an allowlisted phase.
// TotalBytes is zero when unknown. Bytes measure acquired bytes, not verified
// bytes; a full transfer is never evidence of successful installation.
type AcquisitionProgress struct {
	Phase      string `json:"phase"`
	Bytes      int64  `json:"bytes"`
	TotalBytes int64  `json:"totalBytes"`
}

type Status struct {
	Acquisition   AcquisitionProgress `json:"acquisition"`
	Operation     Operation           `json:"operation"`
	Supported     bool                `json:"supported"`
	State         string              `json:"state"`
	Enabled       bool                `json:"enabled"`
	SelectedModel string              `json:"selectedModel"`
	Realtime      bool                `json:"realtime"`
	Backend       string              `json:"backend"`
	Version       string              `json:"version"`
	Progress      float64             `json:"progress"`
	Phase         string              `json:"phase"`
	Error         string              `json:"error"`
	Models        []Model             `json:"models"`
}
