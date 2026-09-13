package app

import (
	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedRuntimeOptionsRemainInertAndGuardActiveWork(t *testing.T) {
	for _, active := range []bool{false, true} {
		a := &App{settings: config.Default()}
		gate := activity.New(activity.Sources{DictationActive: func() bool { return active }})
		root := filepath.Join(t.TempDir(), "Freehand")
		options := a.managedRuntimeOptions(root, gate)
		if options.Directory != filepath.Join(root, "managed-runtime") {
			t.Fatal("managed runtime escaped app data root")
		}
		if options.Preferences != config.Default().ManagedRuntime {
			t.Fatal("wrong default runtime policy")
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Fatal("composition created runtime files")
		}
		if err := options.CheckIdle(); (err != nil) != active {
			t.Fatalf("activity guard: %v", err)
		}
		gate.Close()
		if err := options.CheckIdle(); err == nil {
			t.Fatal("closed admission accepted runtime mutation")
		}
	}
}

func TestManagedRuntimeIsRegisteredBetweenSettingsAndConsumers(t *testing.T) {
	data, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	settings := strings.Index(source, "application.NewService(a.settingsService)")
	managed := strings.Index(source, "application.NewService(a.managedRuntime)")
	dictation := strings.Index(source, "application.NewService(a.dictation)")
	if settings < 0 || managed <= settings || dictation <= managed {
		t.Fatal("runtime lifecycle must start after settings and stop after consumers")
	}
}
