package credential

import (
	"errors"
)

const service = "Freehand"

var ErrNotFound = errors.New("credential not found")

type Store interface {
	Get() (string, error)
	Set(string) error
	Delete() error
	Configured() bool
}
type Keyring struct{ Account string }

func (k Keyring) Get() (string, error) {
	if k.Account == "" {
		return "", errors.New("credential account is required")
	}
	v, e := backendGet(k.Account)
	if errors.Is(e, ErrNotFound) {
		return "", ErrNotFound
	}
	return v, e
}
func (k Keyring) Set(v string) error {
	if k.Account == "" {
		return errors.New("credential account is required")
	}
	if v == "" {
		return errors.New("credential cannot be empty")
	}
	return backendSet(k.Account, v)
}
func (k Keyring) Delete() error {
	if k.Account == "" {
		return errors.New("credential account is required")
	}
	e := backendDelete(k.Account)
	if errors.Is(e, ErrNotFound) {
		return nil
	}
	return e
}
func (k Keyring) Configured() bool { _, e := k.Get(); return e == nil }
