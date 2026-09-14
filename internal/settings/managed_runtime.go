package settings

import (
	"errors"
	"net/url"
	"reflect"
	"slices"
	"strconv"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

// ErrManagedUnavailable distinguishes runtime admission from credential failure.
var ErrManagedUnavailable = errors.New("managed runtime is unavailable; start or repair the selected instance")

func WithManagedRuntimes(resolve func(managedruntime.Instance, compatibility.Role) (managedruntime.ResolvedEndpoint, error), apply func([]managedruntime.Instance)) Option {
	return func(s *Service) { s.managedResolve = resolve; s.managedChanged = apply }
}

// WithManagedInventory reserves runtime ownership before durable inventory writes.
// Publication belongs to the reservation, not the settings-change observer.
func WithManagedInventory(reserve func([]managedruntime.Instance) (*managedruntime.InventoryReservation, error)) Option {
	return func(s *Service) { s.managedReserve = reserve }
}

func (s *Service) reserveManaged(instances []managedruntime.Instance) (*managedruntime.InventoryReservation, error) {
	if s.managedReserve == nil {
		return nil, nil
	}
	return s.managedReserve(instances)
}

// SaveManagedInstances is the ordinary Go runtime persistence callback, not a binding.
// Publication is serialized with all other settings commits, outside saveMu.
func SaveManagedInstances(s *Service, instances []managedruntime.Instance) error {
	instances = slices.Clone(instances)
	if err := managedruntime.ValidateInstances(instances); err != nil {
		return err
	}
	reservation, err := s.reserveManaged(instances)
	if err != nil {
		return err
	}
	defer reservation.Finish(false)
	if !s.publicationMu.TryLock() {
		return errors.New("settings are busy; try again")
	}
	defer s.publicationMu.Unlock()
	if !s.saveMu.TryLock() {
		return errors.New("settings are busy; try again")
	}
	if s.closed.Load() || s.configuration.RecoveryRequired {
		s.saveMu.Unlock()
		return errors.New("settings are unavailable; recover settings before changing runtimes")
	}
	old := s.current()
	next := old
	next.ManagedRuntimes = instances
	err = func() error {
		for _, prev := range old.ManagedRuntimes {
			for _, i := range instances {
				if prev.ID == i.ID && prev.Provider != i.Provider {
					return errors.New("runtime provider cannot change in place")
				}
			}
		}
		if store, ok := s.store.(interface {
			ConnectionCatalog() savedconnection.Catalog
		}); ok {
			for _, c := range store.ConnectionCatalog().Entries {
				if c.Details.ManagedInstanceID != "" && !c.BuiltIn {
					for _, p := range c.Uses {
						if _, _, err := config.ManagedContract(next, c.Details.ManagedInstanceID, roleForPurpose(p)); err != nil {
							return err
						}
					}
				}
			}
		}
		if store, ok := s.store.(interface {
			ApplySelectedConnections(config.Settings) config.Settings
		}); ok {
			next = store.ApplySelectedConnections(next)
		}
		if err := config.Validate(next); err != nil {
			return err
		}
		return s.store.Save(next)
	}()
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
	if err == nil {
		reservation.Finish(true)
		s.publishSettingsChange(old, next, result)
	} else if result.Configuration.RecoveryRequired && s.settingsChanged != nil {
		s.settingsChanged(result)
	}
	return err
}

func roleForPurpose(p savedconnection.Purpose) compatibility.Role {
	switch p {
	case savedconnection.Cleanup:
		return compatibility.PostProcessing
	case savedconnection.Speech:
		return compatibility.Speech
	default:
		return compatibility.Transcription
	}
}

func (s *Service) resolveManaged(v config.Settings, id string, role compatibility.Role) (managedruntime.ResolvedEndpoint, error) {
	i, c, err := config.ManagedContract(v, id, role)
	if err != nil || s.managedResolve == nil || s.closed.Load() || s.configuration.RecoveryRequired {
		return managedruntime.ResolvedEndpoint{}, ErrManagedUnavailable
	}
	e, err := s.managedResolve(i, role)
	if err != nil || e.InstanceID != i.ID || e.Provider != i.Provider || e.CatalogModel != i.Model || e.Generation == 0 || e.Model == "" || !reflect.DeepEqual(e.Contract, c) {
		return managedruntime.ResolvedEndpoint{}, ErrManagedUnavailable
	}
	u, err := url.Parse(e.BaseURL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return managedruntime.ResolvedEndpoint{}, ErrManagedUnavailable
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return managedruntime.ResolvedEndpoint{}, ErrManagedUnavailable
	}
	switch role {
	case compatibility.PostProcessing:
		err = config.ValidatePostProcessingConnection(e.BaseURL, true, e.Model)
	case compatibility.Speech:
		err = config.ValidateTextToSpeechConnection(e.BaseURL, true, config.AuthenticationModeNone, e.Model)
	default:
		err = config.ValidateSTTConnection(e.BaseURL, true, config.AuthenticationModeNone, e.Model, "", nil)
	}
	if err != nil {
		return managedruntime.ResolvedEndpoint{}, ErrManagedUnavailable
	}
	return e, nil
}

// Only the requested role is projected. Task-owned options are not reset.
func (s *Service) managedSettings(v config.Settings, voice bool) (config.Settings, error) {
	id, role := v.ManagedInstanceID, compatibility.Transcription
	if voice {
		id = v.VoiceTranscription.ManagedInstanceID
		if v.VoiceTranscription.Realtime {
			role = compatibility.Realtime
		}
	}
	if id == "" {
		return v, nil
	}
	e, err := s.resolveManaged(v, id, role)
	if err != nil {
		return config.Settings{}, err
	}
	if voice {
		r := &v.VoiceTranscription
		r.BaseURL, r.Model = e.BaseURL, e.Model
		r.CompatibilityProfile, r.ModelProfile = e.Contract.CompatibilityProfile, e.Contract.ModelProfile
		r.AllowInsecureHTTP = true
		r.AuthenticationMode = config.AuthenticationModeNone
		r.Headers = map[string]string{}
		r.HealthPath = ""
	} else {
		v.BaseURL, v.Model = e.BaseURL, e.Model
		v.CompatibilityProfile, v.ModelProfile = e.Contract.CompatibilityProfile, e.Contract.ModelProfile
		v.AllowInsecureHTTP = true
		v.AuthenticationMode = config.AuthenticationModeNone
		v.Headers = map[string]string{}
		v.HealthPath = ""
	}
	return v, nil
}
func (s *Service) managedCleanup(v config.Settings) (config.Settings, error) {
	p := &v.PostProcessing
	if !p.Enabled || p.ManagedInstanceID == "" {
		return v, nil
	}
	e, err := s.resolveManaged(v, p.ManagedInstanceID, compatibility.PostProcessing)
	if err != nil {
		p.BaseURL = ""
		return v, err
	}
	p.BaseURL, p.Model = e.BaseURL, e.Model
	p.AllowInsecureHTTP = true
	p.CompatibilityProfile = e.Contract.CompatibilityProfile
	p.Preset = config.PostProcessingPreset(e.Contract.ModelProfile)
	return v, nil
}
func (s *Service) managedSpeech(v config.Settings) (config.Settings, error) {
	p := &v.TextToSpeech
	if !p.Enabled || p.ManagedInstanceID == "" {
		return v, nil
	}
	e, err := s.resolveManaged(v, p.ManagedInstanceID, compatibility.Speech)
	if err != nil {
		p.BaseURL = ""
		p.AuthenticationMode = config.AuthenticationModeNone
		return v, err
	}
	p.BaseURL, p.Model = e.BaseURL, e.Model
	p.AllowInsecureHTTP = true
	p.AuthenticationMode = config.AuthenticationModeNone
	p.CompatibilityProfile, p.ModelProfile = e.Contract.CompatibilityProfile, e.Contract.ModelProfile
	return v, nil
}

// CurrentSource is admission-only; never destroy another task's manual transport.
func (s *Service) effectiveCurrent() config.Settings {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	v := s.current()
	for _, voice := range []bool{false, true} {
		next, err := s.managedSettings(v, voice)
		if err == nil {
			v = next
			continue
		}
		if voice {
			r := &v.VoiceTranscription
			r.BaseURL = ""
			r.HealthPath = ""
			r.Headers = map[string]string{}
			r.AuthenticationMode = config.AuthenticationModeNone
		} else {
			v.BaseURL = ""
			v.HealthPath = ""
			v.Headers = map[string]string{}
			v.AuthenticationMode = config.AuthenticationModeNone
		}
	}
	v, _ = s.managedCleanup(v)
	v, _ = s.managedSpeech(v)
	return v
}

type connectionResolver struct{ service *Service }

// ConnectionResolver exposes metadata resolution only through an unbound wrapper.
func ConnectionResolver(s *Service) *connectionResolver { return &connectionResolver{service: s} }
func (r *connectionResolver) ResolveSavedConnection(id string) (savedconnection.Connection, string, error) {
	s := r.service
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	store, ok := s.store.(interface {
		ResolveSavedConnection(string) (savedconnection.Connection, string, error)
	})
	if !ok {
		return savedconnection.Connection{}, "", errors.New("saved connections are unavailable")
	}
	c, key, err := store.ResolveSavedConnection(id)
	if err != nil {
		return savedconnection.Connection{}, "", err
	}
	if c.Details.ManagedInstanceID == "" {
		return c, key, nil
	}
	if len(c.Uses) == 0 {
		return savedconnection.Connection{}, "", ErrManagedUnavailable
	}
	e, err := s.resolveManaged(s.current(), c.Details.ManagedInstanceID, roleForPurpose(c.Uses[0]))
	if err != nil {
		return savedconnection.Connection{}, "", err
	}
	c.Details = savedconnection.Details{BaseURL: e.BaseURL, AllowInsecureHTTP: true, AuthenticationMode: config.AuthenticationModeNone, CompatibilityProfile: e.Contract.CompatibilityProfile, Headers: map[string]string{}}
	c.HasCredential = false
	for _, p := range c.Uses {
		if err := savedconnection.Validate(p, c.Details); err != nil {
			return savedconnection.Connection{}, "", ErrManagedUnavailable
		}
	}
	return c, "", nil
}
