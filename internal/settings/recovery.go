package settings

import (
	"errors"
	"fmt"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/diagnostics"
)

// RetryConfiguration reloads the authoritative saved configuration. An expected
// validation failure is returned as renderer-safe status rather than a rejected
// promise so the recovery dialog can update without losing its actions.
func (s *Service) RetryConfiguration() (result SettingsDTO, err error) {
	s.publicationMu.Lock()
	defer s.publicationMu.Unlock()
	started := time.Now()
	s.log().Info("settings recovery retry started")
	s.saveMu.Lock()
	if s.closed.Load() {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("application is shutting down")
	}
	if s.loader == nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("settings recovery is unavailable")
	}
	next, loadErr := s.loader.Load()
	if loadErr != nil {
		failure := config.LoadFailureFor(loadErr)
		s.configuration = ConfigurationStatus{
			RecoveryRequired: true,
			ErrorKind:        failure.Kind,
			Message:          failure.Message,
		}
		result = s.settingsSnapshotLocked()
		s.saveMu.Unlock()
		s.log().Warn("settings recovery retry failed",
			"duration_ms", time.Since(started).Milliseconds(),
			"error_kind", failure.Kind,
		)
		return result, nil
	}
	reservation, err := s.reserveManaged(next.ManagedRuntimes)
	if err != nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, err
	}
	defer reservation.Finish(false)
	result, old, err := s.applyRecoveredSettingsLocked(next, false)
	s.saveMu.Unlock()
	if err != nil {
		s.log().Warn("settings recovery apply failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return SettingsDTO{}, err
	}
	reservation.Finish(true)
	s.publishSettingsChange(old, next, result)
	s.log().Info("settings recovery completed",
		"duration_ms", time.Since(started).Milliseconds(),
		"outcome", "reloaded",
	)
	return result, nil
}

// ResetConfiguration is the only recovery operation that replaces the saved
// document. Credentials remain in the native credential store and are never
// copied into the replacement configuration.
func (s *Service) ResetConfiguration() (result SettingsDTO, err error) {
	s.publicationMu.Lock()
	defer s.publicationMu.Unlock()
	started := time.Now()
	s.log().Info("settings recovery reset started")
	s.saveMu.Lock()
	if s.closed.Load() {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("application is shutting down")
	}
	if !s.configuration.RecoveryRequired {
		result = s.settingsSnapshotLocked()
		s.saveMu.Unlock()
		return result, nil
	}
	next := config.Default()
	reservation, err := s.reserveManaged(next.ManagedRuntimes)
	if err != nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, err
	}
	defer reservation.Finish(false)
	result, old, err := s.applyRecoveredSettingsLocked(next, true)
	s.saveMu.Unlock()
	if err != nil {
		s.log().Warn("settings recovery reset failed", "duration_ms", time.Since(started).Milliseconds(), "error_kind", diagnostics.ErrorKind(err))
		return SettingsDTO{}, err
	}
	reservation.Finish(true)
	s.publishSettingsChange(old, next, result)
	s.log().Info("settings recovery completed", "duration_ms", time.Since(started).Milliseconds(), "outcome", "reset")
	return result, nil
}

func (s *Service) applyRecoveredSettingsLocked(next config.Settings, persist bool) (SettingsDTO, config.Settings, error) {
	validate := config.ValidateStored
	if persist {
		validate = config.Validate
	}
	if err := validate(next); err != nil {
		return SettingsDTO{}, config.Settings{}, err
	}
	old := s.current()
	// Reconcile native state even when a failed rollback left the same runtime snapshot.
	rollbackShortcuts := rollbackStep{name: "shortcuts"}
	if s.shortcutChanged != nil {
		var err error
		rollbackShortcuts.run, err = s.shortcutChanged(next)
		if err != nil {
			return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("shortcuts were not changed: %w", err), rollbackShortcuts)
		}
	}
	rollbackStartup := rollbackStep{name: "startup", run: func() error {
		return s.startup.Set(old.StartWithWindows)
	}}
	if err := s.startup.Set(next.StartWithWindows); err != nil {
		return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("startup setting was not changed: %w", err), rollbackStartup, rollbackShortcuts)
	}
	if persist {
		save := s.store.Save
		if resetter, ok := s.store.(interface{ Reset(config.Settings) error }); ok {
			save = resetter.Reset
		}
		if err := save(next); err != nil {
			return SettingsDTO{}, config.Settings{}, rollback(fmt.Errorf("settings were not reset: %w", err), rollbackStartup, rollbackShortcuts)
		}
	}
	s.mu.Lock()
	s.cfg = next
	s.mu.Unlock()
	s.configuration = ConfigurationStatus{}
	return s.settingsSnapshotLocked(), old, nil
}
