package dictation

import (
	"log/slog"

	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
	"github.com/tnware/freehand-stt/internal/postprocess"
	settingspkg "github.com/tnware/freehand-stt/internal/settings"
)

type testSettings interface{ Current() config.Settings }

type testRecorder struct {
	*recorder
	history *history.Store
}

// New preserves the compact setup used by the original state-machine tests
// while constructing the new package-owned recorder and history dependency.
func New(capture audio.Capture, platform insertion.Platform, client *inference.Client, keys credential.Store, source testSettings, changed func(Status)) *testRecorder {
	settingsSource := settingspkg.Source(source.Current)
	profiles := settingspkg.ProfileSource(func() (settingspkg.RequestProfile, error) {
		cfg := source.Current()
		profile := settingspkg.RequestProfile{Settings: cfg}
		if cfg.AuthenticationMode == config.AuthenticationModeAPIKey && keys != nil {
			key, err := keys.Get()
			if err != nil {
				return settingspkg.RequestProfile{}, err
			}
			profile.STTCredential = key
		}
		return profile, nil
	})
	store := history.NewStore(source.Current().HistoryEnabled, platform)
	return &testRecorder{
		recorder: newRecorder(capture, platform, client, nil, settingsSource, profiles, store, changed, nil),
		history:  store,
	}
}

func (r *testRecorder) Start() error                           { return r.start(RecordingToggle) }
func (r *testRecorder) StartWithMode(mode RecordingMode) error { return r.start(mode) }
func (r *testRecorder) Stop() error                            { return r.stopCurrent() }
func (r *testRecorder) Cancel() error                          { return r.cancelRecording() }
func (r *testRecorder) CopyPending() error                     { return r.copyPending() }
func (r *testRecorder) Status() Status                         { return r.currentStatus() }
func (r *testRecorder) History() []history.HistoryEntry        { return r.history.Entries() }

func (r *testRecorder) SetLogger(logger *slog.Logger) { r.logger = logger }

func (r *testRecorder) SetPostProcessor(processor *postprocess.Processor) {
	r.mu.Lock()
	r.processor = processor
	r.mu.Unlock()
}
