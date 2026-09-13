//go:build darwin

package credential

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxDarwinCredentialBytes = 4096

func validateDarwinCredential(account, value string, setting bool) error {
	if account == "" || len(account) > 256 || strings.ContainsRune(account, 0) || !utf8.ValidString(account) {
		return errors.New("credential account is invalid")
	}
	if setting && (value == "" || len(value) > maxDarwinCredentialBytes) {
		return errors.New("credential must contain 1 to 4096 bytes")
	}
	return nil
}

// Never include keychain diagnostic strings, account names or credential data.
func darwinStatus(status int32) error {
	switch status {
	case 0:
		return nil
	case -25300:
		return ErrNotFound
	default:
		return fmt.Errorf("keychain operation failed (status %d)", status)
	}
}
