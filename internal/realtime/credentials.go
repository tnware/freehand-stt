package realtime

import "strings"

// credentialGuard is scoped to one immutable request. Check both wire text and
// presentation text: parsing model markers or joining deltas can create a match
// that was absent from an individual event.
type credentialGuard string

func (key credentialGuard) check(values ...string) error {
	if key != "" {
		for _, value := range values {
			if strings.Contains(value, string(key)) {
				return credentialReflectionError{}
			}
		}
	}
	return nil
}

type credentialReflectionError struct{}

func (credentialReflectionError) Error() string {
	return "realtime transcription response rejected"
}

func (credentialReflectionError) DiagnosticKind() string { return "credential_reflection" }
