package managedruntime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInventoryReservationRejectsDrainingIDBeforeDurableWrite(t *testing.T) {
	i := Instance{ID: "draining", Name: "Runtime", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{i}})
	w := m.workers[i.ID]
	w.wg.Add(1)
	defer func() { w.wg.Done(); _ = m.ServiceShutdown() }()
	if err := m.DeleteInstance(InstanceRequest{InstanceID: i.ID}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "inventory.json")
	if err := os.WriteFile(path, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	save := func(next []Instance) error {
		r, err := ReserveInstances(m, next)
		if err != nil {
			return err
		}
		defer r.Finish(false)
		b, err := json.Marshal(next)
		if err != nil {
			return err
		}
		if err = os.WriteFile(path, b, 0600); err != nil {
			return err
		}
		r.Finish(true)
		return nil
	}
	if err := save([]Instance{i}); err == nil {
		t.Fatal("accepted draining directory reuse")
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "[]" || len(m.GetInstances()) != 0 {
		t.Fatalf("durable/runtime diverged: %s %v %+v", b, err, m.GetInstances())
	}
}

func TestInventoryReservationReentrantCommitAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[fail], func(t *testing.T) {
			old := Instance{ID: "voice", Name: "Before", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
			next := old
			next.Name = "After"
			durable := []Instance{old}
			var m *Manager
			m = NewManager(ManagerOptions{Directory: t.TempDir(), Instances: durable, SaveInstances: func(rows []Instance) error {
				r, err := ReserveInstances(m, rows)
				if err != nil {
					return err
				}
				defer r.Finish(false)
				conflict := next
				conflict.Model = "parakeet-tdt"
				if other, err := ReserveInstances(m, []Instance{conflict}); err == nil {
					other.Finish(false)
					t.Fatal("conflicting publication entered reserved save")
				}
				if err := m.DeleteInstance(InstanceRequest{InstanceID: old.ID}); err == nil {
					t.Fatal("mutation entered reserved save")
				}
				if fail {
					return errors.New("disk full")
				}
				durable = append([]Instance(nil), rows...)
				r.Finish(true)
				return nil
			}})
			defer m.ServiceShutdown()
			err := m.SetInstance(next)
			if (err != nil) != fail {
				t.Fatalf("save result: %v", err)
			}
			got := m.GetInstances()
			if len(got) != 1 || !reflect.DeepEqual(durable, []Instance{got[0].Instance}) {
				t.Fatalf("durable/runtime diverged: %+v %+v", durable, got)
			}
			want := next
			if fail {
				want = old
			}
			if got[0].Instance != want {
				t.Fatalf("got %+v want %+v", got[0].Instance, want)
			}
			r, err := ReserveInstances(m, durable)
			if err != nil {
				t.Fatal("reservation leaked", err)
			}
			r.Finish(false)
		})
	}
}
