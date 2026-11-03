package storage

import "testing"

func TestLiveInsertDefaultsOffAndPersistsUnavailablePreference(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	if v.LiveInsert {
		t.Fatal("live insertion must default off")
	}
	// No realtime profile is selected. Saving a preference must not enable a
	// transport or erase the preference merely because it is unavailable now.
	v.LiveInsert = true
	v.AutoInsert = false
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	got := loadStore(t, reopen(t, s))
	if !got.LiveInsert || got.AutoInsert || got.VoiceTranscription.Realtime {
		t.Fatal("saved unavailable preference changed delivery or transport settings")
	}
}
