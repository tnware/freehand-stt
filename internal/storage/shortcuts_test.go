package storage

import "testing"

func TestUnassignedShortcutsSurviveDatabaseReopen(t *testing.T) {
	store := testStore(t)
	settings := loadStore(t, store)
	settings.ToggleShortcut, settings.ShowShortcut, settings.HoldShortcut = "", "", ""
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	got := loadStore(t, reopen(t, store))
	if got.ToggleShortcut != "" || got.ShowShortcut != "" || got.HoldShortcut != "" {
		t.Fatal("an intentional empty shortcut was replaced with a default on restart")
	}
}
