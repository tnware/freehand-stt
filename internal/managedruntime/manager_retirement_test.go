package managedruntime

import (
	"fmt"
	"testing"
	"time"
)

func TestManagerReapsDeletedOwnersAndAllowsCompletedIDReuse(t *testing.T) {
	m := NewManager(ManagerOptions{Directory: t.TempDir()})
	t.Cleanup(func() { _ = m.ServiceShutdown() })
	var previous uint64
	for n := 0; n < MaxInstances*3; n++ {
		i := Instance{ID: "reusable", Name: fmt.Sprintf("Runtime %d", n), Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
		deadline := time.Now().Add(time.Second)
		for {
			err := m.SetInstance(i)
			if err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("completed owner prevents recreation: %v", err)
			}
			time.Sleep(time.Millisecond)
		}
		generation := m.workers[i.ID].generation
		if generation <= previous {
			t.Fatal("recreated owner reused a lease generation")
		}
		previous = generation
		if err := m.DeleteInstance(InstanceRequest{InstanceID: i.ID}); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(time.Second)
	for {
		m.mu.Lock()
		retained := len(m.retired)
		m.mu.Unlock()
		if retained == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("retained %d completed owners", retained)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestManagerBoundsDrainingOwners(t *testing.T) {
	m := NewManager(ManagerOptions{Directory: t.TempDir()})
	var draining []*worker
	defer func() {
		for _, w := range draining {
			w.wg.Done()
		}
		_ = m.ServiceShutdown()
	}()
	for n := 0; n < MaxInstances; n++ {
		i := Instance{ID: fmt.Sprintf("draining-%d", n), Name: "Runtime", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
		if err := m.SetInstance(i); err != nil {
			t.Fatal(err)
		}
		w := m.workers[i.ID]
		w.wg.Add(1)
		draining = append(draining, w)
		if err := m.DeleteInstance(InstanceRequest{InstanceID: i.ID}); err != nil {
			t.Fatal(err)
		}
	}
	i := Instance{ID: "overflow", Name: "Runtime", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	if err := m.SetInstance(i); err == nil {
		t.Fatal("unbounded draining owners permit more process/file owners")
	}
}

func TestManagerDoesNotReuseIDWhileDeletedOwnerDrains(t *testing.T) {
	i := Instance{ID: "draining", Name: "Runtime", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{i}})
	w := m.workers[i.ID]
	w.wg.Add(1)
	defer func() { w.wg.Done(); _ = m.ServiceShutdown() }()
	if err := m.DeleteInstance(InstanceRequest{InstanceID: i.ID}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInstance(i); err == nil {
		t.Fatal("reused directory while old owner still drains")
	}
}
