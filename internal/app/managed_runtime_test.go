package app

import (
	"github.com/tnware/freehand-stt/internal/activity"
	"github.com/tnware/freehand-stt/internal/config"
	"os"
	"path/filepath"

	"testing"
)

func TestManagedRuntimeOptionsRemainInertAndGuardActiveWork(t *testing.T) {
	for _, active := range []bool{false, true} {
		a := &App{settings: config.Default()}
		gate := activity.New(activity.Sources{DictationActive: func() bool { return active }})
		root := filepath.Join(t.TempDir(), "Freehand")
		options := a.managedRuntimeOptions(root, gate)
		if options.Directory != root {
			t.Fatal("managed runtime escaped app data root")
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
