package storage

import "testing"

func TestManagedRuntimeMigrationDefaultsAndConstraints(t *testing.T) {
 s := testStore(t)
 loadStore(t, s)
 var enabled, realtime int
 var model string
 if err := s.db.QueryRow(`SELECT enabled, model, realtime FROM managed_runtime_preferences WHERE id=1`).Scan(&enabled, &model, &realtime); err != nil { t.Fatal(err) }
 if enabled != 0 || model != "nemotron-3.5" || realtime != 1 { t.Fatalf("unsafe defaults: %d %q %d", enabled, model, realtime) }
 for _, query := range []string{
 `UPDATE managed_runtime_preferences SET enabled=2`,
 `UPDATE managed_runtime_preferences SET realtime=2`,
 `UPDATE managed_runtime_preferences SET model=''`,
 `INSERT INTO managed_runtime_preferences(id,enabled,model,realtime) VALUES(2,0,'nemotron-3.5',1)`,
 } { if _, err := s.db.Exec(query); err == nil { t.Fatalf("accepted invalid preferences: %s", query) } }
}
