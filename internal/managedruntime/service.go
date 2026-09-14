package managedruntime

import (
	"errors"
	"log/slog"
)

type Options struct {
	Directory       string
	Preferences     Preferences
	SavePreferences func(Preferences) error
	Changed         func(Status)
	Logger          *slog.Logger
	CheckIdle       func() error
}

// Service is the temporary singleton settings/binding facade. Both it and
// Manager delegate process ownership and operations to the same worker engine.
type Service struct {
	*worker
	save    func(Preferences) error
	pending *Preferences
}

func NewService(o Options) *Service {
	s := &Service{worker: newWorker(o.Directory, providers[NeMoSpeechCPP], legacyConfig(o.Preferences), o.Logger, o.CheckIdle), save: o.SavePreferences}
	if o.Changed != nil {
		s.changed = func(snapshot workerSnapshot) { o.Changed(snapshot.Status) }
	}
	if err := Validate(o.Preferences); err != nil {
		s.status.State = "error"
		s.status.Error = err.Error()
	}
	return s
}
func legacyConfig(p Preferences) workerConfig {
	return workerConfig{Model: p.Model, Enabled: p.Enabled, Realtime: p.Realtime, AutoStart: p.Enabled}
}
func (s *Service) legacyPreferencesLocked() Preferences {
	return Preferences{Enabled: s.configuration.Enabled, Model: s.configuration.Model, Realtime: s.configuration.Realtime}
}
func (s *Service) GetPreferences() Preferences {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.legacyPreferencesLocked()
}
func (s *Service) GetStatus() Status { return s.worker.GetStatus() }
func (s *Service) Resolve() (Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolveLocked()
}
func (s *Service) ResolveFor(p Preferences) (Endpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p != s.legacyPreferencesLocked() {
		return Endpoint{}, errNotReady
	}
	return s.resolveLocked()
}
func (s *Service) applyLocked(p Preferences) *ownedProcess { return s.configureLocked(legacyConfig(p)) }

// ApplyPreferences accepts already-committed settings. During the save callback it
// stages the latest committed snapshot; SetPreferences completes the handshake.
// It never persists or calls an external callback while holding the runtime lock.
func ApplyPreferences(s *Service, p Preferences) {
	if s == nil || Validate(p) != nil {
		return
	}
	s.mu.Lock()
	if s.closed || !s.status.Supported {
		s.mu.Unlock()
		return
	}
	if s.saving {
		q := p
		s.pending = &q
		s.mu.Unlock()
		return
	}
	kill := s.applyLocked(p)
	s.mu.Unlock()
	if kill != nil {
		kill.kill()
	}
	s.notify()
}
func (s *Service) SetPreferences(p Preferences) error {
	if err := Validate(p); err != nil {
		return err
	}
	s.mu.Lock()
	if err := s.admitLocked(); err != nil {
		s.mu.Unlock()
		return err
	}
	old := s.legacyPreferencesLocked()
	s.busy = true
	s.saving = true
	s.mu.Unlock()
	var err error
	if old != p {
		err = s.idle()
	}
	if err == nil && s.save != nil {
		err = s.save(p)
	}
	s.mu.Lock()
	var kill *ownedProcess
	if !s.closed {
		if s.pending != nil {
			kill = s.applyLocked(*s.pending)
		} else if err == nil {
			kill = s.applyLocked(p)
		}
	}
	s.pending = nil
	s.saving = false
	s.busy = false
	s.mu.Unlock()
	if kill != nil {
		kill.kill()
	}
	s.notify()
	if err != nil {
		return errors.New("Could not change managed speech preferences. Finish active speech work and try again.")
	}
	return nil
}

func (s *Service) Install() error                { return s.worker.Install() }
func (s *Service) RefreshCatalog() error         { return s.worker.RefreshCatalog() }
func (s *Service) DownloadModel(id string) error { return s.worker.DownloadModel(id) }
func (s *Service) RemoveModel(id string) error   { return s.worker.RemoveModel(id) }
func (s *Service) Remove() error                 { return s.worker.Remove() }
func (s *Service) Start() error                  { return s.worker.Start() }
func (s *Service) Stop() error                   { return s.worker.Stop() }
func (s *Service) Cancel() error                 { return s.worker.Cancel() }
func (s *Service) ServiceShutdown() error        { return s.worker.ServiceShutdown() }
