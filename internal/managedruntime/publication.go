package managedruntime

import "slices"

// InventoryReservation protects a validated inventory until durable persistence
// completes. Finish must be called on every path; Finish(true) marks a successful
// durable commit, Finish(false) aborts. Repeated calls are harmless. No manager
// lock is held while the caller persists or publishes settings.
type InventoryReservation struct {
	manager     *Manager
	transaction *inventoryTransaction
	finished    bool // guarded by manager.mu
}

type inventoryTransaction struct {
	next      []Instance
	holders   int
	committed bool
	joinable  bool
}

// ReserveInstances is an ordinary Go integration API, not a renderer binding.
// Call before writing settings, defer Finish(false), then Finish(true) after the
// durable commit. Do not also call ApplyInstances. The matching settings save
// invoked by ManagerOptions.SaveInstances may join its manager-owned reservation
// once; competing inventories and worker operations are rejected, never waited on.
func ReserveInstances(m *Manager, next []Instance) (*InventoryReservation, error) {
	if m == nil {
		return nil, errNotReady
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, errNotReady
	}
	if tx := m.transaction; tx != nil && tx.joinable && slices.Equal(tx.next, next) {
		tx.joinable = false
		tx.holders++
		r := &InventoryReservation{manager: m, transaction: tx}
		m.mu.Unlock()
		return r, nil
	}
	r, err := m.reserveLocked(next, false)
	m.mu.Unlock()
	if err == nil && m.checkIdle != nil {
		if err = m.checkIdle(); err != nil {
			r.Finish(false)
			return nil, err
		}
	}
	return r, err
}

func (m *Manager) reserveLocked(next []Instance, joinable bool) (*InventoryReservation, error) {
	if err := m.admitLocked(); err != nil {
		return nil, err
	}
	if err := m.validateUpdateLocked(next); err != nil {
		return nil, err
	}
	for _, w := range m.workers {
		w.mu.Lock()
		busy := w.busy
		w.mu.Unlock()
		if busy {
			return nil, errBusy
		}
	}
	tx := &inventoryTransaction{next: slices.Clone(next), holders: 1, joinable: joinable}
	m.busy = true
	m.transaction = tx
	return &InventoryReservation{manager: m, transaction: tx}, nil
}

func (r *InventoryReservation) Finish(committed bool) {
	if r == nil || r.manager == nil {
		return
	}
	m := r.manager
	m.mu.Lock()
	if r.finished {
		m.mu.Unlock()
		return
	}
	r.finished = true
	tx := r.transaction
	tx.committed = tx.committed || committed
	tx.holders--
	if tx.holders != 0 {
		m.mu.Unlock()
		return
	}
	m.transaction = nil
	// busy remains reserved through publication, so no invalidating transition
	// can enter between precommit validation and publication.
	m.mu.Unlock()
	m.finish(tx.next, tx.committed)
}
