package settings

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
)

type storeFake struct {
	log  *[]string
	fail bool
	err  error
}

type recoveryStoreFake struct {
	storeFake
	settings config.Settings
	loadErr  error
	report   config.LoadReport
	saved    *config.Settings
}

func (f *recoveryStoreFake) Load() (config.Settings, error) {
	if f.loadErr != nil {
		return config.Default(), f.loadErr
	}
	return f.settings, nil
}

func (f *recoveryStoreFake) LoadReport() config.LoadReport { return f.report }

func (f *recoveryStoreFake) Save(settings config.Settings) error {
	if err := f.storeFake.Save(settings); err != nil {
		return err
	}
	f.saved = &settings
	return nil
}

func (f *storeFake) Save(config.Settings) error {
	*f.log = append(*f.log, "config:save")
	if f.err != nil {
		return f.err
	}
	if f.fail {
		return errors.New("disk full")
	}
	return nil
}

type startupFake struct {
	log    *[]string
	on     bool
	calls  int
	failAt map[int]error
}

func (f *startupFake) Set(on bool) error {
	*f.log = append(*f.log, "startup:set")
	f.calls++
	if err := f.failAt[f.calls]; err != nil {
		return err
	}
	f.on = on
	return nil
}

type keyFake struct {
	log         *[]string
	value       string
	present     bool
	getErr      error
	setAt       map[int]error
	deleteAt    map[int]error
	setCalls    int
	deleteCalls int
}

func (f *keyFake) Get() (string, error) {
	*f.log = append(*f.log, "credential:get")
	if f.getErr != nil {
		return "", f.getErr
	}
	if !f.present {
		return "", credential.ErrNotFound
	}
	return f.value, nil
}
func (f *keyFake) Set(value string) error {
	*f.log = append(*f.log, "credential:set")
	f.setCalls++
	if err := f.setAt[f.setCalls]; err != nil {
		return err
	}
	f.value, f.present = value, true
	return nil
}
func (f *keyFake) Delete() error {
	*f.log = append(*f.log, "credential:delete")
	f.deleteCalls++
	if err := f.deleteAt[f.deleteCalls]; err != nil {
		return err
	}
	f.value, f.present = "", false
	return nil
}
func (f *keyFake) Configured() bool { return f.present }

func transactionalService(failSave bool) (*Service, *[]string, *startupFake, *keyFake) {
	log := &[]string{}
	startup := &startupFake{log: log}
	keys := &keyFake{log: log, value: "old-secret", present: true}
	processingKeys := &keyFake{log: log}
	service := NewService(
		&storeFake{log: log, fail: failSave}, config.Default(), keys, processingKeys,
		startup, func() (bool, string) { return true, "" },
		func(config.Settings) error { *log = append(*log, "shortcuts:configure"); return nil },
		nil, nil, nil, nil, nil,
	)
	return service, log, startup, keys
}

func request(settings config.Settings, credential string) SaveSettingsRequest {
	return SaveSettingsRequest{Settings: settings, STTCredentialDraft: credential}
}

type transactionHarness struct {
	service        *Service
	log            *[]string
	startup        *startupFake
	keys           *keyFake
	processingKeys *keyFake
	ttsKeys        *keyFake
	shortcutCalls  int
	shortcutFailAt map[int]error
}

func newTransactionHarness(store ConfigStore) *transactionHarness {
	log := &[]string{}
	h := &transactionHarness{
		log:            log,
		startup:        &startupFake{log: log},
		keys:           &keyFake{log: log, value: "old-secret", present: true},
		processingKeys: &keyFake{log: log, value: "old-processing-secret", present: true},
		ttsKeys:        &keyFake{log: log, value: "old-tts-secret", present: true},
		shortcutFailAt: map[int]error{},
	}
	h.service = NewService(
		store, config.Default(), h.keys, h.processingKeys,
		h.startup, func() (bool, string) { return true, "" },
		func(config.Settings) error {
			h.shortcutCalls++
			*log = append(*log, "shortcuts:configure")
			return h.shortcutFailAt[h.shortcutCalls]
		},
		nil, nil, nil, nil, nil,
		WithTextToSpeechCredential(h.ttsKeys),
	)
	return h
}

func transactionRequest() SaveSettingsRequest {
	next := config.Default()
	next.ToggleShortcut = "Ctrl+Alt+A"
	next.StartWithWindows = true
	return SaveSettingsRequest{
		Settings:                      next,
		STTCredentialDraft:            "new-secret",
		PostProcessingCredentialDraft: "new-processing-secret",
	}
}

type blockingStore struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *blockingStore) Save(config.Settings) error {
	s.once.Do(func() { close(s.started) })
	<-s.release
	return nil
}

func TestSaveSettingsOrdersNativeChangesBeforePersistence(t *testing.T) {
	service, log, _, _ := transactionalService(false)
	next := config.Default()
	next.ToggleShortcut = "Ctrl+Alt+A"
	next.StartWithWindows = true
	if _, err := service.SaveSettings(request(next, "new-secret")); err != nil {
		t.Fatal(err)
	}
	want := []string{"shortcuts:configure", "startup:set", "credential:get", "credential:set", "config:save"}
	if !reflect.DeepEqual(*log, want) {
		t.Fatalf("order = %v, want %v", *log, want)
	}
}

func TestSaveSettingsPublishesTheCommittedRendererSnapshot(t *testing.T) {
	service, _, _, _ := transactionalService(false)
	var published SettingsDTO
	service.settingsChanged = func(settings SettingsDTO) { published = settings }
	next := config.Default()
	next.BaseURL = "https://example.test/v1"
	next.Model = "updated-model"
	result, err := service.SaveSettings(request(next, ""))
	if err != nil {
		t.Fatal(err)
	}
	if published.Model != "updated-model" || !reflect.DeepEqual(published, result) {
		t.Fatalf("published settings = %#v, result = %#v", published, result)
	}
}

func TestAppearanceSnapshotKeepsLaunchStateUntilRestart(t *testing.T) {
	log := &[]string{}
	start := &startupFake{log: log}
	keys := &keyFake{log: log}
	processingKeys := &keyFake{log: log}
	initial := config.Default()
	service := NewService(
		&storeFake{log: log}, initial, keys, processingKeys, start,
		func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil,
	)

	next := initial
	next.AppearanceMode = config.AppearanceModeDark
	result, err := service.SaveSettings(request(next, ""))
	if err != nil {
		t.Fatal(err)
	}
	if result.AppearanceMode != config.AppearanceModeDark || result.AppearanceModeActive != config.AppearanceModeSystem {
		t.Fatalf("saved/active appearance = %q/%q, want dark/system", result.AppearanceMode, result.AppearanceModeActive)
	}

	micaInitial := config.Default()
	micaInitial.UseMica = true
	micaInitial.AppearanceMode = config.AppearanceModeDark
	micaService := NewService(
		&storeFake{log: log}, micaInitial, keys, processingKeys, start,
		func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil,
	)
	micaSnapshot := micaService.GetSettings()
	if !micaSnapshot.MicaActive || micaSnapshot.AppearanceModeActive != config.AppearanceModeSystem {
		t.Fatalf("Mica active appearance = mica %v mode %q, want true/system", micaSnapshot.MicaActive, micaSnapshot.AppearanceModeActive)
	}
}

func TestSaveSettingsRollsBackEveryEarlierStage(t *testing.T) {
	service, log, startup, keys := transactionalService(true)
	next := config.Default()
	next.ToggleShortcut = "Ctrl+Alt+A"
	next.StartWithWindows = true
	if _, err := service.SaveSettings(request(next, "new-secret")); err == nil {
		t.Fatal("save succeeded")
	}
	want := []string{"config:save", "credential:set", "startup:set", "shortcuts:configure"}
	got := (*log)[len(*log)-len(want):]
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rollback order = %v, want %v (all: %v)", got, want, *log)
	}
	if startup.on || !keys.present || keys.value != "old-secret" || service.current().ToggleShortcut != config.Default().ToggleShortcut {
		t.Fatal("transaction did not restore prior state")
	}
}

func TestSaveSettingsRejectsOversizedCredentialBeforeDependencies(t *testing.T) {
	service, log, _, keys := transactionalService(false)
	if _, err := service.SaveSettings(request(config.Default(), strings.Repeat("k", MaxAPIKeyBytes+1))); err == nil {
		t.Fatal("oversized API key accepted")
	}
	if len(*log) != 0 || keys.value != "old-secret" {
		t.Fatalf("oversized key reached dependencies: log=%v key=%q", *log, keys.value)
	}
}

func TestRecoveryBlocksOrdinarySettingsAndRequestProfiles(t *testing.T) {
	log := &[]string{}
	store := &recoveryStoreFake{storeFake: storeFake{log: log}, settings: config.Default()}
	failure := config.LoadFailure{Kind: "invalid_json", Message: "The settings file is invalid."}
	keys := &keyFake{log: log}
	service := NewService(
		store, config.Default(), keys, &keyFake{log: log}, &startupFake{log: log},
		func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil,
		WithConfigurationLoad(store, &failure, config.LoadReport{}),
	)

	snapshot := service.GetSettings()
	if !snapshot.Configuration.RecoveryRequired || snapshot.Configuration.ErrorKind != "invalid_json" {
		t.Fatalf("configuration status = %#v", snapshot.Configuration)
	}
	if _, err := service.SaveSettings(request(config.Default(), "")); err == nil {
		t.Fatal("ordinary settings save was accepted during recovery")
	}
	if _, err := RequestProfiles(service).Capture(); err == nil {
		t.Fatal("request profile escaped during recovery")
	}
	if len(*log) != 0 {
		t.Fatalf("blocked operations reached dependencies: %v", *log)
	}
}

func TestRetryConfigurationKeepsRecoveryUntilTheOriginalFileLoads(t *testing.T) {
	log := &[]string{}
	store := &recoveryStoreFake{
		storeFake: storeFake{log: log},
		settings:  config.Default(),
		loadErr:   errors.New("still invalid"),
	}
	failure := config.LoadFailure{Kind: "invalid_json", Message: "Initial failure."}
	service := NewService(
		store, config.Default(), &keyFake{log: log}, &keyFake{log: log}, &startupFake{log: log},
		func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil,
		WithConfigurationLoad(store, &failure, config.LoadReport{}),
	)

	snapshot, err := service.RetryConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Configuration.RecoveryRequired || snapshot.Configuration.ErrorKind != "unavailable" {
		t.Fatalf("retry status = %#v", snapshot.Configuration)
	}
	if store.saved != nil {
		t.Fatal("retry replaced the settings file")
	}

	store.loadErr = nil
	store.report = config.LoadReport{PreservedFieldCount: 1, PreservedFields: []string{"realtime"}}
	snapshot, err = service.RetryConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Configuration.RecoveryRequired || !reflect.DeepEqual(snapshot.Configuration.PreservedFields, []string{"realtime"}) {
		t.Fatalf("recovered status = %#v", snapshot.Configuration)
	}
	if store.saved != nil {
		t.Fatal("successful retry rewrote the valid settings file")
	}
}

func TestResetConfigurationExplicitlyReplacesSettingsButKeepsCredentials(t *testing.T) {
	log := &[]string{}
	store := &recoveryStoreFake{storeFake: storeFake{log: log}, settings: config.Default()}
	failure := config.LoadFailure{Kind: "invalid_values", Message: "A saved setting is invalid."}
	keys := &keyFake{log: log, value: "keep-me", present: true}
	service := NewService(
		store, config.Default(), keys, &keyFake{log: log}, &startupFake{log: log},
		func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil,
		WithConfigurationLoad(store, &failure, config.LoadReport{}),
	)

	snapshot, err := service.ResetConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Configuration.RecoveryRequired || store.saved == nil || !reflect.DeepEqual(*store.saved, config.Default()) {
		t.Fatalf("reset result = %#v saved=%#v", snapshot.Configuration, store.saved)
	}
	if !keys.present || keys.value != "keep-me" || keys.deleteCalls != 0 || keys.setCalls != 0 {
		t.Fatalf("reset changed credential state: %#v", keys)
	}
}

func TestRequestProfileCapturesSettingsAndCredentialsTogether(t *testing.T) {
	service, _, _, keys := transactionalService(false)
	service.cfg.Model = "captured-model"
	service.cfg.CompatibilityProfile = compatibility.Speaches
	service.cfg.PostProcessing.CompatibilityProfile = compatibility.LlamaCPP
	service.cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	keys.value = "captured-key"
	profile, err := RequestProfiles(service).Capture()
	if err != nil {
		t.Fatal(err)
	}
	service.cfg.CompatibilityProfile = compatibility.Generic
	service.cfg.PostProcessing.CompatibilityProfile = compatibility.Generic
	if profile.Settings.CompatibilityProfile != compatibility.Speaches || profile.Settings.PostProcessing.CompatibilityProfile != compatibility.LlamaCPP {
		t.Fatal("captured compatibility selection changed")
	}
	if profile.Settings.Model != "captured-model" || profile.STTCredential != "captured-key" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestTextToSpeechProfileCapturesDedicatedCredential(t *testing.T) {
	h := newTransactionHarness(&storeFake{log: &[]string{}})
	h.service.cfg.TextToSpeech = config.TextToSpeechSettings{
		CompatibilityProfile: compatibility.Speaches,
		Enabled:              true, BaseURL: "https://example.test/v1",
		AuthenticationMode: config.AuthenticationModeAPIKey,
		Model:              "tts-model", Voice: "voice", Speed: 1,
		TimeoutSeconds: config.DefaultTextToSpeechTimeoutSeconds,
	}
	profile, err := TextToSpeechProfiles(h.service).Capture()
	if err != nil {
		t.Fatal(err)
	}
	h.service.cfg.TextToSpeech.CompatibilityProfile = compatibility.Generic
	if profile.Settings.CompatibilityProfile != compatibility.Speaches {
		t.Fatal("captured speech profile changed")
	}
	if profile.Settings.Model != "tts-model" || profile.Credential != "old-tts-secret" {
		t.Fatalf("profile = %#v", profile)
	}
}

func TestTextToSpeechCredentialRollsBackWithSettingsTransaction(t *testing.T) {
	persistErr := errors.New("persistence unavailable")
	h := newTransactionHarness(&storeFake{log: &[]string{}, err: persistErr})
	request := transactionRequest()
	request.TextToSpeechCredentialDraft = "new-tts-secret"
	if _, err := h.service.SaveSettings(request); !errors.Is(err, persistErr) {
		t.Fatalf("save error = %v", err)
	}
	if !h.ttsKeys.present || h.ttsKeys.value != "old-tts-secret" || h.ttsKeys.setCalls != 2 {
		t.Fatalf("TTS credential was not restored: %#v", h.ttsKeys)
	}
}

func TestSaveSettingsRollsBackTheStageThatReportedFailure(t *testing.T) {
	stageErr := errors.New("stage failed")
	tests := []struct {
		name                  string
		fail                  func(*transactionHarness)
		wantShortcuts         int
		wantStartup           int
		wantSTTSets           int
		wantProcessingKeySets int
	}{
		{
			name:          "shortcuts",
			fail:          func(h *transactionHarness) { h.shortcutFailAt[1] = stageErr },
			wantShortcuts: 2,
		},
		{
			name:          "startup",
			fail:          func(h *transactionHarness) { h.startup.failAt = map[int]error{1: stageErr} },
			wantShortcuts: 2,
			wantStartup:   2,
		},
		{
			name:          "STT credential",
			fail:          func(h *transactionHarness) { h.keys.setAt = map[int]error{1: stageErr} },
			wantShortcuts: 2,
			wantStartup:   2,
			wantSTTSets:   2,
		},
		{
			name:                  "post-processing credential",
			fail:                  func(h *transactionHarness) { h.processingKeys.setAt = map[int]error{1: stageErr} },
			wantShortcuts:         2,
			wantStartup:           2,
			wantSTTSets:           2,
			wantProcessingKeySets: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &[]string{}
			h := newTransactionHarness(&storeFake{log: log})
			tt.fail(h)
			if _, err := h.service.SaveSettings(transactionRequest()); err == nil {
				t.Fatal("save succeeded")
			}
			if h.shortcutCalls != tt.wantShortcuts || h.startup.calls != tt.wantStartup || h.keys.setCalls != tt.wantSTTSets || h.processingKeys.setCalls != tt.wantProcessingKeySets {
				t.Fatalf("calls: shortcuts=%d startup=%d stt=%d processing=%d", h.shortcutCalls, h.startup.calls, h.keys.setCalls, h.processingKeys.setCalls)
			}
			if h.service.current().ToggleShortcut != config.Default().ToggleShortcut {
				t.Fatal("failed transaction changed the committed settings snapshot")
			}
		})
	}
}

func TestSaveSettingsPreservesEveryRollbackFailure(t *testing.T) {
	persistErr := errors.New("persistence unavailable")
	shortcutRollbackErr := errors.New("shortcut-sensitive-detail")
	startupRollbackErr := errors.New("startup-sensitive-detail")
	credentialRollbackErr := errors.New("credential-sensitive-detail")
	processingRollbackErr := errors.New("processing-sensitive-detail")
	log := &[]string{}
	h := newTransactionHarness(&storeFake{log: log, err: persistErr})
	h.shortcutFailAt[2] = shortcutRollbackErr
	h.startup.failAt = map[int]error{2: startupRollbackErr}
	h.keys.setAt = map[int]error{2: credentialRollbackErr}
	h.processingKeys.setAt = map[int]error{2: processingRollbackErr}

	_, err := h.service.SaveSettings(transactionRequest())
	if err == nil {
		t.Fatal("save succeeded")
	}
	for _, want := range []error{persistErr, shortcutRollbackErr, startupRollbackErr, credentialRollbackErr, processingRollbackErr} {
		if !errors.Is(err, want) {
			t.Fatalf("error %q does not preserve %q", err, want)
		}
	}
	for _, sensitive := range []string{"shortcut-sensitive-detail", "startup-sensitive-detail", "credential-sensitive-detail", "processing-sensitive-detail"} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("renderer-safe error exposes rollback detail %q: %q", sensitive, err)
		}
	}
	if h.shortcutCalls != 2 || h.startup.calls != 2 || h.keys.setCalls != 2 || h.processingKeys.setCalls != 2 {
		t.Fatalf("rollback calls: shortcuts=%d startup=%d stt=%d processing=%d", h.shortcutCalls, h.startup.calls, h.keys.setCalls, h.processingKeys.setCalls)
	}
}

func TestGetSettingsWaitsForTheCompleteSaveTransaction(t *testing.T) {
	store := &blockingStore{started: make(chan struct{}), release: make(chan struct{})}
	h := newTransactionHarness(store)
	saveResult := make(chan error, 1)
	go func() {
		_, err := h.service.SaveSettings(transactionRequest())
		saveResult <- err
	}()
	<-store.started

	getStarted := make(chan struct{})
	snapshotResult := make(chan SettingsDTO, 1)
	go func() {
		close(getStarted)
		snapshotResult <- h.service.GetSettings()
	}()
	<-getStarted
	select {
	case snapshot := <-snapshotResult:
		t.Fatalf("mixed snapshot escaped during save: %#v", snapshot)
	case <-time.After(50 * time.Millisecond):
	}

	close(store.release)
	if err := <-saveResult; err != nil {
		t.Fatal(err)
	}
	snapshot := <-snapshotResult
	if snapshot.ToggleShortcut != "Ctrl+Alt+A" || !snapshot.CredentialConfigured || !snapshot.PostProcessingCredentialConfigured {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

func TestSaveSettingsPublishesAfterReleasingTheTransaction(t *testing.T) {
	service, _, _, _ := transactionalService(false)
	published := make(chan SettingsDTO, 1)
	service.settingsChanged = func(SettingsDTO) { published <- service.GetSettings() }
	done := make(chan error, 1)
	go func() {
		_, err := service.SaveSettings(request(config.Default(), ""))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("settings callback deadlocked on the transaction lock")
	}
	select {
	case <-published:
	case <-time.After(time.Second):
		t.Fatal("settings callback did not publish")
	}
}

func TestEveryOverlayPreferenceParticipatesInLiveReconfiguration(t *testing.T) {
	for name, change := range map[string]func(*config.Settings){
		"enabled":    func(s *config.Settings) { s.OverlayEnabled = !s.OverlayEnabled },
		"layout":     func(s *config.Settings) { s.OverlayLayout = config.OverlayLayoutDetailed },
		"anchor":     func(s *config.Settings) { s.OverlayAnchor = config.OverlayAnchorBottomRight },
		"visibility": func(s *config.Settings) { s.OverlayVisibility = config.OverlayVisibilityRecording },
		"motion":     func(s *config.Settings) { s.OverlayMotion = config.OverlayMotionReduced },
		"surface":    func(s *config.Settings) { s.OverlaySurface = config.OverlaySurfaceSolid },
		"visualizer": func(s *config.Settings) { s.OverlayVisualizer = config.OverlayVisualizerEnvelope },
		"size":       func(s *config.Settings) { s.OverlaySizePercent = 125 },
		"opacity":    func(s *config.Settings) { s.OverlayOpacityPercent = 80 },
		"edge":       func(s *config.Settings) { s.OverlayTopOffset = 42 },
		"glow":       func(s *config.Settings) { s.OverlayGlowPercent = 50 },
	} {
		t.Run(name, func(t *testing.T) {
			old := config.Default()
			next := old
			change(&next)
			if !overlaySettingsDiffer(old, next) {
				t.Fatal("overlay preference change was not detected")
			}
		})
	}
	if settings := config.Default(); overlaySettingsDiffer(settings, settings) {
		t.Fatal("unchanged overlay settings triggered live reconfiguration")
	}
}
