//go:build windows

package storage

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/settings"
	"reflect"
	"testing"
)

type fixtureStartup struct {
	enabled bool
	fail    bool
}

func (f *fixtureStartup) Set(v bool) error {
	if f.fail {
		return errors.New("fixture startup failure")
	}
	f.enabled = v
	return nil
}

func TestSettingsServiceUsesCommittedSQLiteAndCredentialSnapshot(t *testing.T) {
	s := testStore(t)
	initial := config.Default()
	initial.BaseURL = "https://example.test/v1"
	initial.Model = "fixture"
	initial.AuthenticationMode = config.AuthenticationModeAPIKey
	writeLegacy(t, s, initial)
	initial = loadStore(t, s)
	startup := &fixtureStartup{}
	service := settings.NewService(s, initial, s.STTCredentials(), s.CleanupCredentials(), startup, func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil, settings.WithTextToSpeechCredential(s.SpeechCredentials()), settings.WithConfigurationLoad(s, nil, config.LoadReport{}))
	defer service.ServiceShutdown()
	next := initial
	next.BaseURL = "https://example.test/v1"
	next.Model = "fixture"
	next.AuthenticationMode = config.AuthenticationModeAPIKey
	next.StartWithWindows = true
	if _, err := service.SaveSettings(settings.SaveSettingsRequest{Settings: next, STTCredentialDraft: "first-fixture-key"}); err != nil {
		t.Fatal(err)
	}
	captured, err := settings.RequestProfiles(service).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if captured.STTCredential != "first-fixture-key" || !startup.enabled {
		t.Fatal("initial profile not applied")
	}
	if _, err = s.db.Exec(`CREATE TRIGGER fail_cleanup BEFORE UPDATE ON cleanup_settings BEGIN SELECT RAISE(ABORT, 'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	failed := next
	failed.BaseURL = "https://changed.test/v1"
	failed.StartWithWindows = false
	if _, err = service.SaveSettings(settings.SaveSettingsRequest{Settings: failed, STTCredentialDraft: "second-fixture-key"}); err == nil {
		t.Fatal("failed SQL save reported success")
	}
	profile, err := settings.RequestProfiles(service).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(profile, captured) || !startup.enabled {
		t.Fatal("failed save changed active profile/native state")
	}
	if got := loadStore(t, s); !reflect.DeepEqual(got, next) {
		t.Fatal("failed save changed persistent settings")
	}
	if _, err = s.db.Exec("DROP TRIGGER fail_cleanup"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.SaveSettings(settings.SaveSettingsRequest{Settings: failed, STTCredentialDraft: "second-fixture-key"}); err != nil {
		t.Fatal(err)
	}
	profile, err = settings.RequestProfiles(service).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if profile.Settings.BaseURL != next.BaseURL || profile.STTCredential != "second-fixture-key" || startup.enabled {
		t.Fatal("new profile did not commit coherently")
	}
	if captured.STTCredential != "first-fixture-key" {
		t.Fatal("in-flight snapshot changed")
	}
}

func TestUncertainCommitBlocksJobsAndPublishesRecovery(t *testing.T) {
	s := testStore(t)
	initial := loadStore(t, s)
	startup := &fixtureStartup{}
	var published settings.SettingsDTO
	service := settings.NewService(s, initial, s.STTCredentials(), s.CleanupCredentials(), startup, func() (bool, string) { return true, "" }, nil, nil, nil, nil, func(v settings.SettingsDTO) { published = v }, nil, settings.WithTextToSpeechCredential(s.SpeechCredentials()), settings.WithConfigurationLoad(s, nil, config.LoadReport{}))
	defer service.ServiceShutdown()
	if _, err := s.db.Exec(`CREATE TABLE fixture_deferred(id INTEGER REFERENCES preferences_settings(id) DEFERRABLE INITIALLY DEFERRED); CREATE TRIGGER fixture_commit_failure AFTER UPDATE ON cleanup_settings BEGIN INSERT INTO fixture_deferred VALUES(2); END;`); err != nil {
		t.Fatal(err)
	}
	next := initial
	next.Language = "de"
	next.StartWithWindows = true
	if _, err := service.SaveSettings(settings.SaveSettingsRequest{Settings: next, STTCredentialDraft: "uncommitted-fixture-key"}); err == nil {
		t.Fatal("commit failure accepted")
	}
	if !published.Configuration.RecoveryRequired || published.Configuration.ErrorKind != "commit_uncertain" {
		t.Fatal("uncertain outcome not published")
	}
	if _, err := settings.RequestProfiles(service).Capture(); err == nil {
		t.Fatal("new job allowed while commit outcome uncertain")
	}
	if startup.enabled {
		t.Fatal("native startup not rolled back")
	}
	recovered, err := service.RetryConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Configuration.RecoveryRequired || recovered.Language != initial.Language || recovered.CredentialConfigured {
		t.Fatal("retry exposed uncommitted settings or credential")
	}
	if len(s.vault.(*memoryVault).values) != 0 {
		t.Fatal("retry did not reclaim uncommitted key")
	}
}
