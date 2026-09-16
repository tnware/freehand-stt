package config

import (
	"errors"
)

// FieldError is safe, field-addressable guidance returned by settings validation.
// Field uses a Settings JSON property path; composed validators may identify the
// owning property instead of an individual option. Rejected values and underlying
// errors must not be serialized across the renderer boundary.
type FieldError struct {
	Kind    string `json:"kind"`
	Field   string `json:"field"`
	Message string `json:"message"`
	cause   error
}

func (e *FieldError) Error() string { return e.Message }
func (e *FieldError) Unwrap() error { return e.cause }

func fieldError(field, message string, cause error) error {
	return &FieldError{Kind: "settings_validation", Field: field, Message: message, cause: cause}
}

type LoadFailure struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// LoadFailureFor keeps classified database errors renderer-safe and masks
// unclassified causes, which may contain paths or driver diagnostics.
func LoadFailureFor(err error) LoadFailure {
	var classified interface{ ConfigurationFailure() LoadFailure }
	if errors.As(err, &classified) {
		return classified.ConfigurationFailure()
	}
	return LoadFailure{Kind: "unavailable", Message: "The saved configuration could not be loaded."}
}
