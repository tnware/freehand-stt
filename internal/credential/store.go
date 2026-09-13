package credential

import (
	"errors"
)

const service = "Freehand"

const (
	STTAccount            = "stt-api-key"
	PostProcessingAccount = "post-processing-api-key"
	TextToSpeechAccount   = "text-to-speech-api-key"
)

var ErrNotFound = errors.New("credential not found")

type Store interface {
	Get() (string, error)
	Set(string) error
	Delete() error
	Configured() bool
}
type Keyring struct{ Account string }

func (k Keyring) account() string {
	if k.Account == "" {
		return STTAccount
	}
	return k.Account
}

func (k Keyring) Get() (string, error) {
	v, e := backendGet(k.account())
	if errors.Is(e, ErrNotFound) {
		return "", ErrNotFound
	}
	return v, e
}
func (k Keyring) Set(v string) error {
	if v == "" {
		return errors.New("credential cannot be empty")
	}
	return backendSet(k.account(), v)
}
func (k Keyring) Delete() error {
	e := backendDelete(k.account())
	if errors.Is(e, ErrNotFound) {
		return nil
	}
	return e
}
func (k Keyring) Configured() bool { _, e := k.Get(); return e == nil }
