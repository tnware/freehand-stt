package managedruntime

import (
	"context"
	"slices"
	"testing"
)

// A second registered test provider exercises cross-provider independence
// without treating production NeMo copies as separate providers.
func registerTestProvider(t *testing.T) ProviderID {
	t.Helper()
	id := ProviderID("test-speech")
	providers[id] = nemoProvider{}
	t.Cleanup(func() { delete(providers, id) })
	return id
}

type delayedProviderAdapter struct {
	workerAdapter
	done chan struct{}
}

func (a *delayedProviderAdapter) Start(context.Context, string) (processHandle, Endpoint, error) {
	return &fakeProcess{done: a.done, closeJob: func() {}}, Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:12345", Model: "authoritative"}, nil
}

func TestProviderProcessAdmissionRetainsStoppingOwner(t *testing.T) {
	one := Instance{ID: LegacyInstanceID, Name: "One", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	two := one
	two.ID = "copy"
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{one, two}})
	done := make(chan struct{})
	defer func() { close(done); _ = m.ServiceShutdown() }()
	for _, w := range m.workers {
		w.status.Supported = true
		w.adapter = &workerAdapter{installed: true, downloaded: true}
	}
	if err := m.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, i := range []Instance{one, two} {
		waitInstance(t, m, i.ID, "installed")
	}
	first, second := m.workers[one.ID], m.workers[two.ID]
	first.adapter = &delayedProviderAdapter{done: done}
	if err := first.startProcess(t.Context()); err != nil {
		t.Fatal(err)
	}
	// A cancelled/removed endpoint is not proof that the child has exited.
	if err := second.startProcess(t.Context()); err == nil {
		t.Fatal("started concurrent provider process")
	}
	first.mu.Lock()
	killed := first.configureLocked(workerConfig{Enabled: true, Model: "parakeet-tdt"})
	first.mu.Unlock()
	killed.Kill()
	if err := second.startProcess(t.Context()); err == nil {
		t.Fatal("forgot provider owner before process exit")
	}
}

func TestProviderInstallRejectsLegacyDuplicates(t *testing.T) {
	one := Instance{ID: LegacyInstanceID, Name: "One", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	two := one
	two.ID = "copy"
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{one, two}})
	defer m.ServiceShutdown()
	for _, w := range m.workers {
		w.status.Supported = true
		w.adapter = &workerAdapter{}
	}
	if err := m.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, i := range []Instance{one, two} {
		waitInstance(t, m, i.ID, "not_installed")
	}
	if err := m.Install(InstanceRequest{InstanceID: two.ID}); err == nil {
		t.Fatal("installed duplicate provider")
	}
}

func TestProviderInventoryRejectsNewCopiesPreservesLegacyRecovery(t *testing.T) {
	one := Instance{ID: LegacyInstanceID, Name: "Existing", Provider: NeMoSpeechCPP, Model: "nemotron-3.5"}
	two := one
	two.ID = "existing-copy"
	m := NewManager(ManagerOptions{Directory: t.TempDir(), Instances: []Instance{one, two}})
	defer m.ServiceShutdown()
	if m.initErr != nil {
		t.Fatal("repairable inventory rejected", m.initErr)
	}
	if err := ApplyInstances(m, []Instance{one, two}); err != nil {
		t.Fatal("unchanged legacy inventory rejected", err)
	}
	three := one
	three.ID = string(NeMoSpeechCPP)
	if err := m.SetInstance(three); err == nil {
		t.Fatal("created third copy of provider")
	}
	if r, err := ReserveInstances(m, []Instance{one, two, three}); err == nil {
		r.Finish(false)
		t.Fatal("reserved duplicate provider for persistence")
	}
	if !slices.Equal(m.instances, []Instance{one, two}) {
		t.Fatal("rejected update changed existing identities")
	}
	if err := m.DeleteInstance(InstanceRequest{InstanceID: two.ID}); err != nil {
		t.Fatal("legacy duplicate cannot be explicitly removed", err)
	}
	if err := m.SetInstance(three); err == nil {
		t.Fatal("created duplicate of retained provider")
	}
	fresh := NewManager(ManagerOptions{Directory: t.TempDir()})
	defer fresh.ServiceShutdown()
	if r, err := ReserveInstances(fresh, []Instance{one, two}); err == nil {
		r.Finish(false)
		t.Fatal("fresh duplicate inventory accepted")
	}
}
