package insertion

import (
	"errors"
	"strings"
)

// Stage identifies the operation that rejected automatic insertion, not an app.
type Stage string

const (
	Capture  Stage = "capture"
	Validate Stage = "validate"
	Send     Stage = "send"
)

// rejection retains only allowlisted constants, never a wrapped platform error,
// AX metadata, target identity or transcript. Its sentinel remains compatible
// with existing copy-required callers on every platform.
type rejection struct {
	stage  Stage
	reason string
}

func (e *rejection) Error() string {
	return "automatic insertion unavailable (" + string(e.stage) + ": " + e.reason + ")"
}
func (e *rejection) Unwrap() error { return ErrCopyRequired }

// NewRejection fails closed to generic copy guidance for unknown categories.
func NewRejection(stage Stage, reason string) error {
	switch stage {
	case Capture, Validate, Send:
	default:
		return ErrCopyRequired
	}
	if attributeRejection(reason) {
		return &rejection{stage: stage, reason: reason}
	}
	switch reason {
	case "owner_invalid", "target_invalid", "input_invalid", "ax_not_trusted", "secure_input",
		"pid_invalid", "self_target", "process_start_unavailable", "process_start_zero",
		"attribute_read_failed", "element_type_invalid", "element_timeout_failed",
		"role_not_allowed", "subrole_unavailable_or_secure", "not_enabled",
		"protected_content_unknown_or_true", "value_settable_query_failed", "value_not_settable",
		"system_element_unavailable", "system_timeout_failed", "focused_app_unavailable",
		"app_pid_failed", "focused_element_unavailable", "focused_window_unavailable",
		"element_pid_failed", "window_pid_failed", "element_pid_mismatch", "window_pid_mismatch",
		"element_window_unavailable", "element_window_mismatch", "process_start_changed",
		"secure_input_late", "ax_not_trusted_late", "process_changed", "process_reused",
		"focused_element_changed", "focused_window_changed", "modifiers_held",
		"event_source_unavailable", "keyboard_event_unavailable":
		return &rejection{stage: stage, reason: reason}
	default:
		return ErrCopyRequired
	}
}

// Attribute diagnostics contain only a known query and a bounded AX error kind.
func attributeRejection(reason string) bool {
	field, kind, ok := strings.Cut(reason, "__")
	if !ok {
		return false
	}
	switch field {
	case "focused_app", "focused_element", "focused_window", "element_window", "role", "enabled":
	default:
		return false
	}
	switch kind {
	case "cannot_complete", "unsupported", "no_value", "invalid_element", "api_disabled", "not_implemented", "failure", "other":
		return true
	}
	return false
}

// CopyRequired preserves a structured rejection while dropping raw wrappers.
// Callers may store this result for a run without retaining sensitive errors.
func CopyRequired(err error) error {
	var r *rejection
	if errors.As(err, &r) && r != nil {
		return NewRejection(r.stage, r.reason)
	}
	return ErrCopyRequired
}

// CopyRequiredMessage does not claim focus changed or that no text was typed:
// native event posting has no delivery acknowledgement and may be partial.
func CopyRequiredMessage(err error) string {
	const guidance = ". Copy the transcript and paste it where you want it."
	if r, ok := CopyRequired(err).(*rejection); ok {
		message := "Automatic insertion unavailable (" + string(r.stage) + ": " + r.reason + ")" + guidance
		if r.stage != Capture {
			message += " Check the target for any text already inserted before pasting."
		}
		return message
	}
	return "Automatic insertion unavailable" + guidance + " Check the target for any text already inserted before pasting."
}
