package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"net"
	"net/url"
)

// WithManagedRuntime injects endpoint resolution and a non-persisting apply hook.
// Resolve receives the committed preferences so a transitioning runtime cannot
// supply a previous model. Apply runs after commit, outside the settings locks.
func WithManagedRuntime(resolve func(managedruntime.Preferences) (managedruntime.Endpoint, error), apply func(managedruntime.Preferences)) Option {
	return func(s *Service) { s.managedResolve = resolve; s.managedChanged = apply }
}

// SaveManagedPreferences is an ordinary Go persistence callback, not a binding.
// The runtime save handshake holds no runtime lock. Publish the committed
// preferences outside saveMu but inside publicationMu, so an earlier general
// settings publication cannot override this commit's staged runtime snapshot.
func SaveManagedPreferences(s *Service, p managedruntime.Preferences) error {
	if err := managedruntime.Validate(p); err != nil {
		return err
	}
	if !s.publicationMu.TryLock() {
		return errors.New("settings are busy; try again")
	}
	defer s.publicationMu.Unlock()
	if !s.saveMu.TryLock() {
		return errors.New("settings are busy; try again")
	}
	if s.closed.Load() || s.configuration.RecoveryRequired {
		s.saveMu.Unlock()
		return errors.New("settings are unavailable; recover settings before changing managed speech")
	}
	next := s.current()
	next.ManagedRuntime = p
	err := s.store.Save(next)
	if err != nil {
		if failure := config.LoadFailureFor(err); failure.Kind == "commit_uncertain" {
			s.configuration = ConfigurationStatus{RecoveryRequired: true, ErrorKind: failure.Kind, Message: failure.Message}
		}
	} else {
		s.mu.Lock()
		s.cfg = next
		s.mu.Unlock()
	}
	result := s.settingsSnapshotLocked()
	s.saveMu.Unlock()
	if err == nil && s.managedChanged != nil {
		s.managedChanged(p)
	}
	if s.settingsChanged != nil && (err == nil || result.Configuration.RecoveryRequired) {
		s.settingsChanged(result)
	}
	return err
}

// effectiveCurrent is an admission projection, never an editable or persisted
// snapshot. Failure deliberately clears transcription peers; Capture reports
// the actionable error and never retries the saved remote endpoint.
func (s *Service) effectiveCurrent() config.Settings {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	v := s.current()
	if !v.ManagedRuntime.Enabled {
		return v
	}
	if !s.configuration.RecoveryRequired && !s.closed.Load() {
		if next, err := s.managedSettings(v, true); err == nil {
			return next
		}
	}
	v.BaseURL, v.Model, v.HealthPath = "", "", ""
	v.AuthenticationMode = config.AuthenticationModeNone
	v.Headers = map[string]string{}
	v.VoiceTranscription = config.DefaultVoiceTranscription()
	return v
}

func (s *Service) managedSettings(v config.Settings, voice bool) (config.Settings, error) {
	if !v.ManagedRuntime.Enabled {
		return v, nil
	}
	if s.managedResolve == nil {
		return config.Settings{}, errors.New("local speech runtime is unavailable")
	}
	endpoint, err := s.managedResolve(v.ManagedRuntime)
	if err != nil {
		return config.Settings{}, errors.New("local speech runtime is not ready")
	}
	if !endpoint.Enabled || endpoint.BaseURL == "" || endpoint.Model == "" {
		return config.Settings{}, errors.New("local speech runtime is not ready")
	}
	u, parseErr := url.Parse(endpoint.BaseURL)
	if parseErr != nil || u.Scheme != "http" || !net.ParseIP(u.Hostname()).IsLoopback() {
		return config.Settings{}, errors.New("local speech runtime endpoint is invalid")
	}
	if err := config.ValidateSTTConnection(endpoint.BaseURL, true, config.AuthenticationModeNone, endpoint.Model, "", nil); err != nil {
		return config.Settings{}, errors.New("local speech runtime endpoint is invalid")
	}
	v.BaseURL, v.Model = endpoint.BaseURL, endpoint.Model
	v.ModelProfile = modelprofile.ID(endpoint.Profile)
	v.CompatibilityProfile = compatibility.NeMoSpeechV1
	v.AuthenticationMode = config.AuthenticationModeNone
	v.AllowInsecureHTTP = true // Only the owned loopback peer validated above.
	v.Headers = map[string]string{}
	v.HealthPath = ""
	v.TranscriptionOptions = config.TranscriptionOptions{}
	if v.Language == "" {
		v.Language = "auto"
	}
	r := &v.VoiceTranscription
	r.BaseURL, r.Model = endpoint.BaseURL, endpoint.Model
	r.ModelProfile, r.CompatibilityProfile = v.ModelProfile, v.CompatibilityProfile
	r.AuthenticationMode = config.AuthenticationModeNone
	r.AllowInsecureHTTP = true
	r.Headers = map[string]string{}
	r.HealthPath = ""
	r.TranscriptionOptions = config.TranscriptionOptions{}
	r.Realtime = voice && endpoint.Realtime
	if r.Language == "" {
		r.Language = "auto"
	}
	return v, nil
}
