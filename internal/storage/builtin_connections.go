package storage

import (
	"errors"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func (state connectionState) manualCount() int {
	count := 0
	for _, c := range state.entries {
		if !c.BuiltIn {
			count++
		}
	}
	return count
}

// projectBuiltIns joins runtime-owned rows into the one connection state used by
// catalog reads, selection, credentials and persistence. On load this is a pure
// projection; the next ordinary settings transaction materializes the rows using
// the same generated queries and foreign keys as manual connections.
func (state *connectionState) projectBuiltIns(v config.Settings) error {
	wanted := make(map[string]bool, len(v.ManagedRuntimes))
	for _, instance := range v.ManagedRuntimes {
		wanted[savedconnection.BuiltInID(instance.ID)] = true
	}
	for id, c := range state.entries {
		if !c.BuiltIn || wanted[id] {
			continue
		}
		for _, selected := range state.selected {
			if selected == id {
				return errors.New("deselect the built-in connection before removing its runtime")
			}
		}
		delete(state.entries, id)
	}
	for _, instance := range v.ManagedRuntimes {
		c := savedconnection.BuiltIn(instance)
		if old, exists := state.entries[c.ID]; exists {
			if old.Details.ManagedInstanceID != instance.ID || old.account != "" || savedconnection.ValidateUses(old.Uses, old.Details) != nil {
				return errors.New("built-in connection identity conflicts with a saved connection")
			}
		}
		state.entries[c.ID] = storedConnection{Connection: c}
	}
	return nil
}
