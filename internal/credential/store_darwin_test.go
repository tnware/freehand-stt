//go:build darwin

package credential

import (
	"errors"
	"strings"
	"testing"
)

// These tests deliberately never call SecItem APIs or inspect the login keychain.
func TestDarwinCredentialValidation(t *testing.T) {
	for _, tc := range []struct {
		account, value string
		valid          bool
	}{
		{"connection:test-identity", "secret", true}, {"connection:123", "ユニコード", true},
		{"", "secret", false}, {"a\x00b", "secret", false},
		{strings.Repeat("a", 257), "secret", false},
		{"connection:test-identity", "", false}, {"connection:test-identity", strings.Repeat("s", 4097), false},
	} {
		if err := validateDarwinCredential(tc.account, tc.value, true); (err == nil) != tc.valid {
			t.Errorf("validation valid=%v: %v", tc.valid, err)
		}
	}
}
func TestDarwinStatusPreservesNotFoundAndBoundsErrors(t *testing.T) {
	if !errors.Is(darwinStatus(-25300), ErrNotFound) {
		t.Fatal("item not found must map to ErrNotFound")
	}
	if err := darwinStatus(0); err != nil {
		t.Fatal(err)
	}
	if err := darwinStatus(-25293); err == nil || len(err.Error()) > 100 {
		t.Fatal("expected bounded status error")
	}
	if service != "Freehand" {
		t.Fatal("service identity changed")
	}
}
