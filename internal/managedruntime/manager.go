package managedruntime

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

type ManagerOptions struct {
	Directory     string
	Instances     []Instance
	SaveInstances func([]Instance) error
	Changed       func(InstanceStatus)
	Logger        *slog.Logger
	CheckIdle     func() error
}
type InstanceStatus struct {
	Instance    Instance `json:"instance"`
	Status      Status   `json:"status"`
	ActiveModel string   `json:"activeModel"`
}
type InstanceRequest struct {
	InstanceID string `json:"instanceID"`
}
type BackendRequest struct {
	InstanceID string `json:"instanceID"`
	Backend    string `json:"backend"`
}
type ModelRequest struct {
	InstanceID string `json:"instanceID"`
	Model      string `json:"model"`
}
type ResolvedEndpoint struct {
	InstanceID   string     `json:"instanceID"`
	Provider     ProviderID `json:"provider"`
	CatalogModel string     `json:"catalogModel"`
	Generation   uint64     `json:"generation"`
	BaseURL      string     `json:"baseURL"`
	Model        string     `json:"model"`
	Contract     Contract   `json:"contract"`
}

// Manager owns inventory and admission, not process state. Each instance uses
// the same worker engine as the temporary Service facade. Directory is the
// application-data root (unlike the legacy Options.Directory runtime root).
// Lock order is manager -> worker; callbacks and persistence run under neither.
type Manager struct {
	mu           sync.Mutex
	directory    string
	instances    []Instance
	workers      map[string]*worker
	retired      map[string]*worker
	processes    map[ProviderID]*providerProcess
	save         func([]Instance) error
	changed      func(InstanceStatus)
	logger       *slog.Logger
	checkIdle    func() error
	ctx          context.Context
	cancel       context.CancelFunc
	busy, closed bool
	transaction  *inventoryTransaction
	initErr      error
}

// NewManager is metadata-only. Invalid initial inventories fail closed at
// startup; callers should validate before committing a settings snapshot.
func NewManager(o ManagerOptions) *Manager {
	m := &Manager{directory: o.Directory, save: o.SaveInstances, changed: o.Changed, logger: o.Logger, checkIdle: o.CheckIdle, workers: map[string]*worker{}, retired: map[string]*worker{}}
	m.initErr = ValidateInstances(o.Instances)
	if m.initErr == nil {
		m.instances = slices.Clone(o.Instances)
		for _, i := range m.instances {
			m.workers[i.ID] = m.newWorker(i)
		}
	}
	return m
}
func instanceConfig(i Instance) workerConfig {
	_, realtimeErr := Qualify(i.Provider, i.Model, compatibility.Realtime)
	return workerConfig{Model: i.Model, Enabled: true, Realtime: realtimeErr == nil, AutoStart: i.AutoStart}
}
func (m *Manager) newWorker(i Instance) *worker {
	w := newWorker(instanceDirectory(m.directory, i.ID), providers[i.Provider], instanceConfig(i), m.logger, m.checkIdle)
	if m.processes == nil {
		m.processes = make(map[ProviderID]*providerProcess)
	}
	if m.processes[i.Provider] == nil {
		m.processes[i.Provider] = &providerProcess{}
	}
	w.providerProcess = m.processes[i.Provider]
	w.changed = func(Status) { m.notifyWorker(i.ID, w) }
	return w
}
func (m *Manager) GetProviders() []ProviderDescriptor {
	ids := make([]ProviderID, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	out := make([]ProviderDescriptor, 0, len(ids))
	for _, id := range ids {
		out = append(out, providers[id].descriptor())
	}
	return out
}
func instanceSnapshot(i Instance, w *worker) InstanceStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	return InstanceStatus{Instance: i, Status: w.snapshotLocked(), ActiveModel: w.endpoint.Model}
}
func (m *Manager) GetInstances() []InstanceStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]InstanceStatus, 0, len(m.instances))
	for _, i := range m.instances {
		out = append(out, instanceSnapshot(i, m.workers[i.ID]))
	}
	return out
}
func (m *Manager) notifyWorker(id string, w *worker) {
	m.mu.Lock()
	if m.closed || m.workers[id] != w || m.changed == nil {
		m.mu.Unlock()
		return
	}
	var st InstanceStatus
	for _, i := range m.instances {
		if i.ID == id {
			st = instanceSnapshot(i, w)
			break
		}
	}
	changed := m.changed
	m.mu.Unlock()
	changed(st)
}
func (m *Manager) notifyAll() {
	m.mu.Lock()
	pairs := make(map[string]*worker, len(m.workers))
	for id, w := range m.workers {
		pairs[id] = w
	}
	m.mu.Unlock()
	for id, w := range pairs {
		m.notifyWorker(id, w)
	}
}
func (m *Manager) ResolveFor(expected Instance, role compatibility.Role) (ResolvedEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.busy || m.initErr != nil {
		return ResolvedEndpoint{}, errNotReady
	}
	for _, i := range m.instances {
		if i.ID != expected.ID {
			continue
		}
		if i != expected {
			return ResolvedEndpoint{}, errNotReady
		}
		c, err := Qualify(i.Provider, i.Model, role)
		if err != nil {
			return ResolvedEndpoint{}, err
		}
		w := m.workers[i.ID]
		w.mu.Lock()
		defer w.mu.Unlock()
		ep, err := w.resolveLocked()
		if err != nil {
			return ResolvedEndpoint{}, err
		}
		return ResolvedEndpoint{InstanceID: i.ID, Provider: i.Provider, CatalogModel: i.Model, Generation: w.generation, BaseURL: ep.BaseURL, Model: ep.Model, Contract: c}, nil
	}
	return ResolvedEndpoint{}, errNotReady
}
func (m *Manager) admitLocked() error {
	if m.closed {
		return errNotReady
	}
	if m.initErr != nil {
		return m.initErr
	}
	if m.busy {
		return errBusy
	}
	return nil
}

// ApplyInstances applies metadata without persistence. Durable settings writers
// must instead ReserveInstances before committing and finish that reservation.
func ApplyInstances(m *Manager, instances []Instance) error {
	r, err := ReserveInstances(m, instances)
	if err != nil {
		return err
	}
	r.Finish(true)
	return nil
}

// publicationLocked only changes metadata and fences/cancels affected workers.
// Provider identity cannot change in-place: a different provider needs a new ID
// so no adapter can adopt another provider's installation or model cache.
func (m *Manager) publicationLocked(next []Instance) (kills []*ownedProcess, closing, starting []*worker) {
	nextIDs := map[string]bool{}
	for _, i := range next {
		nextIDs[i.ID] = true
	}
	for _, old := range m.instances {
		w := m.workers[old.ID]
		if !nextIDs[old.ID] {
			w.mu.Lock()
			w.closed = true
			w.mu.Unlock()
			closing = append(closing, w)
			m.retired[old.ID] = w
			delete(m.workers, old.ID)
		}
	}
	for _, i := range next {
		w := m.workers[i.ID]
		if w == nil {
			w = m.newWorker(i)
			m.workers[i.ID] = w
			if m.ctx != nil {
				starting = append(starting, w)
			}
		} else {
			var old Instance
			for _, row := range m.instances {
				if row.ID == i.ID {
					old = row
					break
				}
			}
			if old != i {
				w.mu.Lock()
				generation := w.generation
				if p := w.configureLocked(instanceConfig(i)); p != nil {
					kills = append(kills, p)
				}
				if w.generation == generation {
					w.generation = leaseGeneration.Add(1)
				}
				w.mu.Unlock()
			}
		}
	}
	m.instances = slices.Clone(next)
	m.initErr = nil
	return
}
func (m *Manager) validateUpdateLocked(next []Instance) error {
	if err := ValidateInstances(next); err != nil {
		return err
	}
	// Existing duplicate providers remain repairable without rewriting IDs,
	// directories or Connections. Only retained identities may coexist; a
	// publication may reduce that legacy set, never add or replace a copy.
	seenProviders := make(map[ProviderID]string)
	for _, i := range next {
		if m.workers[i.ID] == nil {
			for _, old := range m.instances {
				if old.Provider == i.Provider {
					return errors.New("This runtime provider is already configured. Use its existing installation.")
				}
			}
			if _, exists := seenProviders[i.Provider]; exists {
				return errors.New("Only one instance per runtime provider may be configured.")
			}
		}
		seenProviders[i.Provider] = i.ID
	}
	// Deleted owners still count while draining: neither their process trees
	// nor directory mutations may be replaced by an unbounded stream of IDs.
	owners := len(m.workers) + len(m.retired)
	for _, i := range next {
		if m.workers[i.ID] == nil && m.retired[i.ID] == nil {
			owners++
		}
	}
	if owners > MaxInstances {
		return errors.New("Wait for deleted runtime instances to finish stopping before adding more.")
	}
	for _, i := range next {
		if m.retired[i.ID] != nil {
			return errors.New("Wait for the deleted runtime instance to finish stopping before reusing its ID.")
		}
		for _, old := range m.instances {
			if old.ID == i.ID && old.Provider != i.Provider {
				return errors.New("Use a new instance ID for a different runtime provider.")
			}
		}
	}
	return nil
}
func (m *Manager) finish(next []Instance, publish bool) {
	m.mu.Lock()
	var kills []*ownedProcess
	var closing, starting []*worker
	if publish {
		kills, closing, starting = m.publicationLocked(next)
	}
	if m.closed {
		closing = append(closing, starting...)
		starting = nil
	}
	ctx := m.ctx
	m.busy = false
	m.mu.Unlock()
	for _, w := range closing {
		w.closeNow()
		// Admission is closed: reserve the directory until every old operation
		// and process monitor exits, then release the owner rather than leaking it.
		go func(w *worker) {
			w.wg.Wait()
			m.mu.Lock()
			for id, retired := range m.retired {
				if retired == w {
					delete(m.retired, id)
				}
			}
			m.mu.Unlock()
		}(w)
	}
	for _, p := range kills {
		p.kill()
	}
	for _, w := range starting {
		_ = w.startup(ctx)
	}
	m.notifyAll()
}
func (m *Manager) change(update func([]Instance) ([]Instance, error)) error {
	m.mu.Lock()
	if err := m.admitLocked(); err != nil {
		m.mu.Unlock()
		return err
	}
	next, err := update(slices.Clone(m.instances))
	if err != nil {
		m.mu.Unlock()
		return err
	}
	r, err := m.reserveLocked(next, false)
	m.mu.Unlock()
	if err != nil {
		return err
	}
	if m.checkIdle != nil {
		err = m.checkIdle()
	}
	if err == nil && m.save != nil {
		m.mu.Lock()
		r.transaction.joinable = true
		m.mu.Unlock()
		err = m.save(slices.Clone(next))
		m.mu.Lock()
		r.transaction.joinable = false
		m.mu.Unlock()
	}
	r.Finish(err == nil)
	if err != nil {
		return errors.New("Could not change managed runtimes. Finish active work and try again.")
	}
	return nil
}
func (m *Manager) SetInstance(i Instance) error {
	if err := ValidateInstance(i); err != nil {
		return err
	}
	return m.change(func(rows []Instance) ([]Instance, error) {
		for n := range rows {
			if rows[n].ID == i.ID {
				rows[n] = i
				return rows, nil
			}
		}
		return append(rows, i), nil
	})
}
func (m *Manager) DeleteInstance(r InstanceRequest) error {
	return m.change(func(rows []Instance) ([]Instance, error) {
		for n := range rows {
			if rows[n].ID == r.InstanceID {
				return slices.Delete(rows, n, n+1), nil
			}
		}
		return nil, errors.New("Choose an existing managed runtime instance.")
	})
}
func (m *Manager) operation(id string, f func(*worker) error) error {
	m.mu.Lock()
	if err := m.admitLocked(); err != nil {
		m.mu.Unlock()
		return err
	}
	w := m.workers[id]
	if w == nil {
		m.mu.Unlock()
		return errors.New("Choose an existing managed runtime instance.")
	}
	m.busy = true
	m.mu.Unlock()
	err := f(w)
	m.finish(nil, false)
	return err
}
func (m *Manager) Install(r InstanceRequest) error {
	return m.install(r.InstanceID, (*worker).Install)
}
func (m *Manager) InstallBackend(r BackendRequest) error {
	if r.Backend != "cpu" && r.Backend != "cuda" {
		return errors.New("Choose CPU or NVIDIA CUDA.")
	}
	return m.install(r.InstanceID, func(w *worker) error { return w.InstallBackend(r.Backend) })
}
func (m *Manager) install(id string, install func(*worker) error) error {
	return m.operation(id, func(w *worker) error {
		m.mu.Lock()
		for _, other := range m.workers {
			if other != w && other.providerProcess == w.providerProcess {
				m.mu.Unlock()
				return errors.New("Resolve duplicate runtime installations before installing this provider. Existing installations and Connections have been preserved.")
			}
		}
		for _, other := range m.retired {
			if other.providerProcess == w.providerProcess {
				m.mu.Unlock()
				return errors.New("Wait for the previous runtime installation to finish stopping.")
			}
		}
		m.mu.Unlock()
		return install(w)
	})
}
func (m *Manager) RefreshCatalog(r InstanceRequest) error {
	return m.operation(r.InstanceID, (*worker).RefreshCatalog)
}
func (m *Manager) Start(r InstanceRequest) error  { return m.operation(r.InstanceID, (*worker).Start) }
func (m *Manager) Stop(r InstanceRequest) error   { return m.operation(r.InstanceID, (*worker).Stop) }
func (m *Manager) Cancel(r InstanceRequest) error { return m.operation(r.InstanceID, (*worker).Cancel) }
func (m *Manager) Remove(r InstanceRequest) error { return m.operation(r.InstanceID, (*worker).Remove) }
func (m *Manager) DownloadModel(r ModelRequest) error {
	return m.operation(r.InstanceID, func(w *worker) error { return w.DownloadModel(r.Model) })
}
func (m *Manager) RemoveModel(r ModelRequest) error {
	return m.operation(r.InstanceID, func(w *worker) error { return w.RemoveModel(r.Model) })
}
func (m *Manager) startup(ctx context.Context) error {
	m.mu.Lock()
	if m.closed || m.initErr != nil {
		err := m.initErr
		m.mu.Unlock()
		if err != nil {
			return err
		}
		return errNotReady
	}
	if m.ctx != nil {
		m.mu.Unlock()
		return nil
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	owned := m.ctx
	workers := make([]*worker, 0, len(m.workers))
	for _, w := range m.workers {
		workers = append(workers, w)
	}
	m.mu.Unlock()
	for _, w := range workers {
		if err := w.startup(owned); err != nil {
			return err
		}
	}
	return nil
}

// ServiceShutdown cancels every live and retired owner before waiting against
// one overall eight-second deadline, never eight seconds per instance.
func (m *Manager) ServiceShutdown() error {
	deadline := time.Now().Add(8 * time.Second)
	m.mu.Lock()
	m.closed = true
	if m.cancel != nil {
		m.cancel()
	}
	workers := make([]*worker, 0, len(m.workers)+len(m.retired))
	for _, w := range m.workers {
		workers = append(workers, w)
	}
	for _, w := range m.retired {
		workers = append(workers, w)
	}
	m.mu.Unlock()
	for _, w := range workers {
		w.closeNow()
	}
	return waitWorkers(workers, time.Until(deadline))
}
