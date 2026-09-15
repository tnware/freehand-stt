package managedruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInventoryChangeDistinguishesActivityFromSaveFailure(t *testing.T) {
	for _, active := range []bool{true, false} {
		t.Run(map[bool]string{true: "active work", false: "save failure"}[active], func(t *testing.T) {
			before := Instance{ID: "voice", Name: "Voice", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
			next := before
			next.Model = "parakeet-tdt"
			failed := true
			saved := false
			m := NewManager(ManagerOptions{
				Directory: t.TempDir(), Instances: []Instance{before},
				CheckIdle: func() error {
					if active && failed {
						return errors.New("private activity detail")
					}
					return nil
				},
				SaveInstances: func([]Instance) error {
					saved = true
					if failed {
						return errors.New("private storage detail")
					}
					return nil
				},
			})
			defer m.ServiceShutdown()
			err := m.SetInstance(next)
			if err == nil || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "Finish active work") != active {
				t.Fatalf("misleading or unsafe binding error: %v", err)
			}
			if saved == active || m.GetInstances()[0].Instance != before {
				t.Fatal("failed mutation persisted or published inventory")
			}
			failed = false
			if err := m.SetInstance(next); err != nil {
				t.Fatalf("failed mutation retained reservation: %v", err)
			}
			if m.GetInstances()[0].Instance != next {
				t.Fatal("retry did not publish inventory")
			}
		})
	}
}

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

// A terminal event must permit the next action, and must precede that action's
// initial event even when the consumer immediately submits it from the callback.
func TestTerminalPublicationAdmitsNextActionInOrder(t *testing.T) {
	var w *worker
	events := make(chan Operation, 4)
	admitted := make(chan error, 1)
	m := NewManager(ManagerOptions{
		Directory: t.TempDir(),
		Instances: []Instance{{ID: "test", Name: "Test", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}},
		Changed: func(st InstanceStatus) {
			events <- st.Status.Operation
			if st.Status.Operation.Kind == "first" && st.Status.Operation.Outcome == "succeeded" {
				admitted <- w.run("second", "", false, func(context.Context) error { return nil })
			}
		},
	})
	w = m.workers["test"]
	w.ctx = t.Context()
	w.status.Supported = true
	if err := w.run("first", "", false, func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	defer w.wg.Wait()
	select {
	case err := <-admitted:
		if err != nil {
			t.Fatalf("terminal status still rejected the next action: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("terminal publication missing")
	}
	for _, want := range [][2]string{{"first", "running"}, {"first", "succeeded"}, {"second", "running"}, {"second", "succeeded"}} {
		select {
		case got := <-events:
			if got.Kind != want[0] || got.Outcome != want[1] {
				t.Fatalf("publication reordered: %+v, want %v", got, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("publication missing")
		}
	}
}
