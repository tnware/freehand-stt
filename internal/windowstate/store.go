package windowstate

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const stateFileName = "window-state.json"

// Store persists disposable UI placement independently from product settings.
type Store struct {
	Path string
	mu   sync.Mutex
}

// NewStore locates the placement file beside Freehand's settings file.
func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return &Store{Path: filepath.Join(dir, "Freehand", stateFileName)}, nil
}

// Load returns found=false when no placement has been stored yet.
func (s *Store) Load() (placement Placement, found bool, err error) {
	if s == nil {
		return Placement{}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Placement{}, false, nil
	}
	if err != nil {
		return Placement{}, false, err
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, 16*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&placement); err != nil {
		return Placement{}, false, err
	}
	if placement.Width <= 0 || placement.Height <= 0 {
		return Placement{}, false, errors.New("window placement has invalid dimensions")
	}
	return placement, true, nil
}

// Save atomically replaces the current placement.
func (s *Store) Save(placement Placement) error {
	if s == nil {
		return nil
	}
	if placement.Width <= 0 || placement.Height <= 0 {
		return errors.New("window placement has invalid dimensions")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(placement, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.Path), "window-state-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err = temporary.Chmod(0o600); err == nil {
		_, err = temporary.Write(data)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporaryPath, s.Path)
}
