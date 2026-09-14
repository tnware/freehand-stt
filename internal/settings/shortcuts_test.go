package settings

import (
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestSaveSettingsAcceptsAndPublishesUnassignedToggle(t *testing.T) {
	service, _, _, _ := transactionalService(false)
	next := config.Default()
	next.ToggleShortcut = ""
	result, err := service.SaveSettings(request(next, ""))
	if err != nil {
		t.Fatal(err)
	}
	if result.ToggleShortcut != "" || service.current().ToggleShortcut != "" {
		t.Fatal("cleared toggle did not reach committed settings")
	}
	next.ToggleShortcut = "Ctrl+Alt+K"
	result, err = service.SaveSettings(request(next, ""))
	if err != nil || result.ToggleShortcut != next.ToggleShortcut {
		t.Fatalf("replacement save failed: %v", err)
	}
}
