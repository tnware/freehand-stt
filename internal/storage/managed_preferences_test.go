package storage

import (
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"reflect"
	"testing"
)

func TestManagedPreferencesRoundTripPreservesBYO(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "local", Name: "Local", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5", AutoStart: true}}
	if e := s.Save(v); e != nil {
		t.Fatal(e)
	}
	got := loadStore(t, reopen(t, s))
	if !reflect.DeepEqual(got, v) {
		t.Fatal("instance round trip changed settings")
	}
}
