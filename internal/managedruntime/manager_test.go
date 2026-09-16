package managedruntime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func waitInstance(t *testing.T, m *Manager, id, state string) InstanceStatus {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		for _, s := range m.GetInstances() {
			if s.Instance.ID == id && s.Status.State == state && s.Status.Phase == "" {
				return s
			}
		}
		select {
		case <-deadline:
			t.Fatalf("instance %s never became %s: %+v", id, state, m.GetInstances())
		case <-time.After(time.Millisecond):
		}
	}
}

func TestManagerRealAdaptersIsolatePathsAndRestartGenerations(t *testing.T) {
	secondProvider := registerTestProvider(t)
	root := t.TempDir()
	one := Instance{ID: LegacyInstanceID, Name: "Legacy", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	two := Instance{ID: "second", Name: "Second", Provider: secondProvider, Model: "nemotron-3.5"}
	m := NewManager(ManagerOptions{Directory: root, Instances: []Instance{one, two}})
	if _, err := os.Stat(filepath.Join(root, "managed-runtime")); !os.IsNotExist(err) {
		t.Fatal("constructor performed I/O", err)
	}
	for _, i := range []Instance{one, two} {
		dir := filepath.Join(root, "managed-runtimes", i.ID)
		if i.ID == LegacyInstanceID {
			dir = filepath.Join(root, "managed-runtime")
		}
		a, _ := managedAdapterFixture(t, dir)
		if err := a.Pull(t.Context(), i.Model, nil); err != nil {
			t.Fatal(err)
		}
		m.workers[i.ID].adapter = a
		m.workers[i.ID].status.Supported = true
	}
	t.Cleanup(func() { _ = m.ServiceShutdown() })
	if err := m.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, i := range []Instance{one, two} {
		waitInstance(t, m, i.ID, "installed")
		if err := m.Start(InstanceRequest{InstanceID: i.ID}); err != nil {
			t.Fatal(err)
		}
		waitInstance(t, m, i.ID, "running")
	}
	before, err := m.ResolveFor(one, compatibility.Realtime)
	if err != nil {
		t.Fatal(err)
	}
	other, err := m.ResolveFor(two, compatibility.Transcription)
	if err != nil || other.BaseURL == before.BaseURL {
		t.Fatalf("distinct servers: %+v %v", other, err)
	}
	if before.Model != "actual GGUF name" {
		t.Fatal("lost authoritative API model", before)
	}
	if err := m.Stop(InstanceRequest{InstanceID: one.ID}); err != nil {
		t.Fatal(err)
	}
	waitInstance(t, m, one.ID, "stopped")
	if err := m.Start(InstanceRequest{InstanceID: one.ID}); err != nil {
		t.Fatal(err)
	}
	waitInstance(t, m, one.ID, "running")
	after, err := m.ResolveFor(one, compatibility.Transcription)
	if err != nil || after.Generation <= before.Generation {
		t.Fatalf("restart reused endpoint generation: before=%+v after=%+v err=%v", before, after, err)
	}
	if err := m.Remove(InstanceRequest{InstanceID: one.ID}); err != nil {
		t.Fatal(err)
	}
	waitInstance(t, m, one.ID, "not_installed")
	if _, err := os.Stat(filepath.Join(root, "managed-runtime")); !os.IsNotExist(err) {
		t.Fatal("legacy removal failed", err)
	}
	if _, err := os.Stat(filepath.Join(root, "managed-runtimes", two.ID, "runtime")); err != nil {
		t.Fatal("legacy removal deleted sibling", err)
	}
	client := loopbackClient()
	defer client.CloseIdleConnections()
	var models struct{ Data []struct{ ID string } }
	if err := metadataJSON(t.Context(), client, other.BaseURL+"/models", &models); err != nil || len(models.Data) != 1 {
		t.Fatal("other server lost", err)
	}
}

func TestManagerIndependentlySupervisesInstances(t *testing.T) {
	secondProvider := registerTestProvider(t)
	one := Instance{ID: LegacyInstanceID, Name: "Voice", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	two := Instance{ID: "second", Name: "Files", Provider: secondProvider, Model: "nemotron-3.5"}
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{one, two}})
	adapters := map[string]*workerAdapter{}
	for id, w := range m.workers {
		a := &workerAdapter{installed: true, downloaded: true}
		adapters[id] = a
		w.adapter = a
		w.status.Supported = true
	}
	if err := m.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.ServiceShutdown() })
	for _, i := range []Instance{one, two} {
		waitInstance(t, m, i.ID, "installed")
		if err := m.Start(InstanceRequest{InstanceID: i.ID}); err != nil {
			t.Fatal(err)
		}
		waitInstance(t, m, i.ID, "running")
	}
	before, err := m.ResolveFor(two, compatibility.Transcription)
	if err != nil || before.InstanceID != two.ID || before.Model != "authoritative" || before.CatalogModel != two.Model || before.Generation == 0 {
		t.Fatalf("resolve %+v %v", before, err)
	}
	if _, err := m.ResolveFor(one, compatibility.PostProcessing); err == nil {
		t.Fatal("speech instance resolved as cleanup")
	}
	if err := m.Stop(InstanceRequest{InstanceID: one.ID}); err != nil {
		t.Fatal(err)
	}
	waitInstance(t, m, one.ID, "stopped")
	after, err := m.ResolveFor(two, compatibility.Transcription)
	if err != nil || after.Generation != before.Generation || after.BaseURL != before.BaseURL {
		t.Fatalf("stopping one changed other: %+v %v", after, err)
	}
	select {
	case <-adapters[two.ID].process.Done():
		t.Fatal("stopped another process")
	default:
	}
	old := two
	two.Name = "Renamed"
	if err := m.SetInstance(two); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ResolveFor(old, compatibility.Transcription); err == nil {
		t.Fatal("stale definition resolved")
	}
	if _, err := m.ResolveFor(two, compatibility.Transcription); err != nil {
		t.Fatal(err)
	}
	one.Model = "parakeet-tdt"
	if err := m.SetInstance(one); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ResolveFor(two, compatibility.Transcription); err != nil {
		t.Fatal("changing one invalidated another", err)
	}
	if err := m.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	for _, a := range adapters {
		select {
		case <-a.process.Done():
		default:
			t.Fatal("shutdown left a child owned")
		}
	}
	if _, err := m.ResolveFor(two, compatibility.Transcription); err == nil {
		t.Fatal("resolved after shutdown")
	}
}
