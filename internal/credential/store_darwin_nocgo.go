//go:build darwin && !cgo

package credential

import "errors"

var errKeychainUnavailable = errors.New("macOS Keychain requires cgo")

func backendGet(string) (string, error) { return "", errKeychainUnavailable }
func backendSet(string, string) error   { return errKeychainUnavailable }
func backendDelete(string) error        { return errKeychainUnavailable }
