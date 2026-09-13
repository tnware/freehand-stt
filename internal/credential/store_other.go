//go:build !darwin

package credential

import (
	"errors"
	keyring "github.com/zalando/go-keyring"
)

func backendGet(account string) (string, error) {
	value, err := keyring.Get(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return value, err
}
func backendSet(account, value string) error { return keyring.Set(service, account, value) }
func backendDelete(account string) error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
