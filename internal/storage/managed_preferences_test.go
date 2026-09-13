package storage

import (
 "testing"
 "reflect"
)

func TestManagedPreferencesRoundTripPreservesBYO(t *testing.T) {
 s := testStore(t)
 v := loadStore(t, s)
 v.ManagedRuntime.Enabled = true
 if err := s.Save(v); err != nil { t.Fatal(err) }
 got := loadStore(t, reopen(t, s))
 if !reflect.DeepEqual(got, v) { t.Fatalf("managed save/reopen changed settings: managed=%#v want=%#v", got.ManagedRuntime, v.ManagedRuntime) }
}
