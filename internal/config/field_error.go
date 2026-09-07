package config

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
