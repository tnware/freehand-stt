// Package settings owns the single settings transaction, native rollback, and
// coherent settings and credential snapshots exposed through its Wails service.
package settings

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/diagnostics"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type HoldInfo func() (bool, string)
type Startup interface {
	Set(bool) error
}
type ConfigStore interface{ Save(config.Settings) error }
type ConfigLoader interface {
	ConfigStore
	Load() (config.Settings, error)
}

type Option func(*Service)

// WithConfigurationLoad connects the startup load result to the bound
// settings service. It is optional so isolated service tests and consumers
// that only need persistence do not acquire a recovery dependency.
func WithConfigurationLoad(loader ConfigLoader, failure *config.LoadFailure) Option {
	return func(service *Service) {
		service.loader = loader
		if failure != nil {
			service.configuration.RecoveryRequired = true
			service.configuration.ErrorKind = failure.Kind
			service.configuration.Message = failure.Message
		}
	}
}

func WithVoiceCredential(store credential.Store) Option {
	return func(service *Service) { service.voiceKeys = store }
}

func WithTextToSpeechCredential(store credential.Store) Option {
	return func(service *Service) { service.ttsKeys = store }
}

// WithUpdateChecks applies the persisted update-check preference after the
// settings transaction commits. Update checks are runtime policy rather than
// part of the native settings rollback transaction.
func WithUpdateChecks(apply func(bool)) Option {
	return func(service *Service) { service.updateChecksChanged = apply }
}

type Service struct {
	managedResolve         func(managedruntime.Instance, compatibility.Role) (managedruntime.ResolvedEndpoint, error)
	managedChanged         func([]managedruntime.Instance)
	managedReserve         func([]managedruntime.Instance) (*managedruntime.InventoryReservation, error)
	mu                     sync.RWMutex
	publicationMu          sync.Mutex // Serializes commits through runtime publication; callbacks may read settings.
	saveMu                 sync.Mutex
	cfg                    config.Settings
	store                  ConfigStore
	loader                 ConfigLoader
	configuration          ConfigurationStatus
	keys                   credential.Store
	processKeys            credential.Store
	ttsKeys                credential.Store
	voiceKeys              credential.Store
	startup                Startup
	hold                   HoldInfo
	holdRetry              func() error
	shortcutChanged        func(config.Settings) (func() error, error)
	overlaySettingsChanged func(config.Settings)
	historyEnabledChanged  func(bool)
	fileSettingsChanged    func(config.Settings)
	updateChecksChanged    func(bool)
	settingsChanged        func(SettingsDTO)
	logger                 *slog.Logger
	micaActive             bool
	appearanceModeActive   config.AppearanceMode
	closed                 atomic.Bool
}

// shortcutChanged returns the rollback for its pre-change native snapshot, even
// on failure if its own partial changes require recovery. Only this settings
// transaction invokes it; saved preferences are not a native binding snapshot.
func NewService(st ConfigStore, cfg config.Settings, k credential.Store, processKeys credential.Store, start Startup, hold HoldInfo, shortcutChanged func(config.Settings) (func() error, error), overlaySettingsChanged func(config.Settings), historyEnabledChanged func(bool), fileSettingsChanged func(config.Settings), settingsChanged func(SettingsDTO), logger *slog.Logger, options ...Option) *Service {
	if logger == nil {
		logger = diagnostics.DiscardLogger()
	}
	service := &Service{
		store: st, cfg: cfg, keys: k, processKeys: processKeys,
		startup: start, hold: hold, shortcutChanged: shortcutChanged,
		overlaySettingsChanged: overlaySettingsChanged,
		historyEnabledChanged:  historyEnabledChanged,
		fileSettingsChanged:    fileSettingsChanged,
		settingsChanged:        settingsChanged,
		logger:                 logger.With("component", "settings"),
		micaActive:             cfg.UseMica && runtime.GOOS == "windows",
		appearanceModeActive:   config.EffectiveAppearanceMode(cfg.UseMica && runtime.GOOS == "windows", cfg.AppearanceMode),
	}
	for _, option := range options {
		option(service)
	}
	return service
}

// WithHoldRetry injects a bounded native rearm operation; never runs on mount/save.
func WithHoldRetry(retry func() error) Option {
	return func(s *Service) { s.holdRetry = retry }
}

// RetryHoldShortcut retries the saved assignment without editing or persisting it.
func (s *Service) RetryHoldShortcut() (SettingsDTO, error) {
	if s.closed.Load() {
		return SettingsDTO{}, errors.New("application is shutting down")
	}
	if !s.publicationMu.TryLock() {
		return SettingsDTO{}, errors.New("settings are busy; try again")
	}
	defer s.publicationMu.Unlock()
	if !s.saveMu.TryLock() {
		return SettingsDTO{}, errors.New("settings are busy; try again")
	}
	if s.holdRetry == nil {
		s.saveMu.Unlock()
		return SettingsDTO{}, errors.New("hold-to-talk retry is unavailable")
	}
	err := s.holdRetry()
	result := s.settingsSnapshotLocked()
	s.saveMu.Unlock()
	if s.settingsChanged != nil {
		s.settingsChanged(result)
	}
	return result, err
}

func (s *Service) publishSettingsChange(old, next config.Settings, result SettingsDTO) {
	if s.managedChanged != nil {
		s.managedChanged(slices.Clone(next.ManagedRuntimes))
	}
	if overlaySettingsDiffer(old, next) && s.overlaySettingsChanged != nil {
		s.overlaySettingsChanged(next)
	}
	if s.historyEnabledChanged != nil {
		s.historyEnabledChanged(next.HistoryEnabled)
	}
	if s.fileSettingsChanged != nil {
		s.fileSettingsChanged(next)
	}
	if s.updateChecksChanged != nil {
		s.updateChecksChanged(next.CheckForUpdates)
	}
	if s.settingsChanged != nil {
		s.settingsChanged(result)
	}
}

func overlaySettingsDiffer(old, next config.Settings) bool {
	return old.ToggleShortcut != next.ToggleShortcut ||
		old.HoldShortcut != next.HoldShortcut ||
		old.OverlayEnabled != next.OverlayEnabled ||
		old.OverlaySizePercent != next.OverlaySizePercent ||
		old.OverlayOpacityPercent != next.OverlayOpacityPercent ||
		old.OverlayTopOffset != next.OverlayTopOffset ||
		old.OverlayGlowPercent != next.OverlayGlowPercent ||
		old.OverlayLayout != next.OverlayLayout ||
		old.OverlayAnchor != next.OverlayAnchor ||
		old.OverlayVisibility != next.OverlayVisibility ||
		old.OverlayMotion != next.OverlayMotion ||
		old.OverlaySurface != next.OverlaySurface ||
		old.OverlayVisualizer != next.OverlayVisualizer
}

func (s *Service) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return diagnostics.DiscardLogger()
}

func (s *Service) ServiceStartup(context.Context, application.ServiceOptions) error {
	s.closed.Store(false)
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.closed.Store(true)
	if closer, ok := s.store.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
