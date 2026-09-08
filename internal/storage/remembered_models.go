package storage

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

func (s *Store) RememberedModels() modelsettings.Catalog {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := modelsettings.Catalog{Entries: []modelsettings.Entry{}, Defaults: modelsettings.Defaults()}
	for _, e := range s.connections.models {
		if s.connections.selected[e.Purpose] == e.ConnectionID {
			out.Entries = append(out.Entries, e)
		}
	}
	return out
}
func (s *connectionState) forgetModels(id string) {
	s.models = slices.DeleteFunc(s.models, func(e modelsettings.Entry) bool { return e.ConnectionID == id })
}
func (s connectionState) restoreModel(v config.Settings, p savedconnection.Purpose, id string) config.Settings {
	for _, e := range s.models {
		if e.ConnectionID == id && e.Purpose == p && e.Selected {
			return modelsettings.Select(v, p, e.Model, e.Options)
		}
	}
	return modelsettings.Select(v, p, "", modelsettings.Defaults()[p])
}
func readRememberedModels(ctx context.Context, q *dbgen.Queries, state *connectionState) error {
	rows, err := q.ListRememberedModels(ctx)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, r := range rows {
		e := modelsettings.Entry{ConnectionID: r.ConnectionID, Purpose: savedconnection.Purpose(r.Purpose), Model: r.Model, Selected: r.Selected != 0, Options: modelsettings.Options{Realtime: modelprofile.NemotronOptions{Vocabulary: r.Vocabulary, Boost: r.Boost}, Profile: modelprofile.ID(r.Profile), Language: r.Language, Transcription: compatibility.TranscriptionOptions{Prompt: r.Prompt, Hotwords: r.Hotwords, TemperatureOverride: r.TemperatureOverride != 0, Temperature: r.Temperature}, Cleanup: compatibility.CleanupOptions{LimitOutputTokens: r.LimitOutputTokens != 0, MaxOutputTokens: int(r.MaxOutputTokens), DisableReasoning: r.DisableReasoning != 0}, SystemPrompt: r.SystemPrompt, Styling: r.Styling, Structure: r.Structure, Context: r.Context, Voice: r.Voice, Speed: r.Speed}}
		c, ok := state.entries[e.ConnectionID]
		counts[e.ConnectionID+":"+string(e.Purpose)]++
		if !ok || !c.Supports(e.Purpose) || counts[e.ConnectionID+":"+string(e.Purpose)] > modelsettings.MaxPerUse {
			return errors.New("invalid remembered model owner or count")
		}
		if err := modelsettings.Validate(e, c.Details); err != nil {
			return err
		}
		state.models = append(state.models, e)
	}
	return nil
}
func writeRememberedModels(ctx context.Context, q *dbgen.Queries, state *connectionState, v config.Settings) error {
	state.models = slices.DeleteFunc(state.models, func(e modelsettings.Entry) bool {
		c, ok := state.entries[e.ConnectionID]
		return !ok || !c.Supports(e.Purpose)
	})
	if err := rememberActiveModels(state, v); err != nil {
		return err
	}
	if err := q.ClearRememberedModels(ctx); err != nil {
		return err
	}
	for _, e := range state.models {
		o := e.Options
		if err := q.PutRememberedModel(ctx, dbgen.PutRememberedModelParams{ConnectionID: e.ConnectionID, Purpose: string(e.Purpose), Model: e.Model, Selected: boolean(e.Selected), Profile: string(o.Profile), Language: o.Language, Prompt: o.Transcription.Prompt, Hotwords: o.Transcription.Hotwords, TemperatureOverride: boolean(o.Transcription.TemperatureOverride), Temperature: o.Transcription.Temperature, LimitOutputTokens: boolean(o.Cleanup.LimitOutputTokens), MaxOutputTokens: int64(o.Cleanup.MaxOutputTokens), DisableReasoning: boolean(o.Cleanup.DisableReasoning), SystemPrompt: o.SystemPrompt, Styling: o.Styling, Structure: o.Structure, Context: o.Context, Voice: o.Voice, Speed: o.Speed, Vocabulary: o.Realtime.Vocabulary, Boost: o.Realtime.Boost}); err != nil {
			return err
		}
	}
	return nil
}

// BeginForgetModel runs inside the settings service save lock and credential transaction.
func (s *Store) BeginForgetModel(key modelsettings.Key, v config.Settings) (config.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil || s.pendingConnections != nil || s.connections.selected[key.Purpose] != key.ConnectionID || key.ConnectionID == "" {
		return v, errors.New("model selection changed; reload settings")
	}
	state := s.connections.clone()
	found := false
	state.models = slices.DeleteFunc(state.models, func(e modelsettings.Entry) bool {
		match := e.ConnectionID == key.ConnectionID && e.Purpose == key.Purpose && e.Model == key.Model
		found = found || match
		return match
	})
	if !found {
		return v, errors.New("remembered model is unavailable; reload settings")
	}
	if modelsettings.Model(v, key.Purpose) == key.Model {
		v = modelsettings.Select(savedconnection.ClearModel(v, key.Purpose), key.Purpose, "", modelsettings.Defaults()[key.Purpose])
	}
	s.pendingConnections = &state
	return v, nil
}

func rememberActiveModels(state *connectionState, v config.Settings) error {
	for _, p := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech, savedconnection.Realtime} {
		id := state.selected[p]
		if id == "" {
			continue
		}
		model := strings.TrimSpace(modelsettings.Model(v, p))
		if model == "" && !(p == savedconnection.Transcription && v.CompatibilityProfile == compatibility.WhisperCPP) {
			continue
		}
		e := modelsettings.Entry{ConnectionID: id, Purpose: p, Model: model, Selected: true, Options: modelsettings.Extract(v, p)}
		if err := modelsettings.Validate(e, state.entries[id].Details); err != nil {
			return err
		}
		found := false
		count := 0
		for i, old := range state.models {
			if old.ConnectionID == id && old.Purpose == p {
				count++
				state.models[i].Selected = false
				if old.Model == model {
					state.models[i] = e
					found = true
				}
			}
		}
		if !found {
			if count >= modelsettings.MaxPerUse {
				return errors.New("this connection already has 32 remembered models for this feature; forget an unused model first")
			}
			state.models = append(state.models, e)
		}
	}
	return nil
}

// BeginModelEdits stages all edited model drafts; Save commits them with the active selection.
func (s *Store) BeginModelEdits(edits []modelsettings.Edit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil || s.pendingConnections != nil || len(edits) > modelsettings.MaxPerUse*3 {
		return errors.New("model edits are unavailable or exceed the limit")
	}
	state := s.connections.clone()
	seen := map[modelsettings.Key]bool{}
	for _, edit := range edits {
		key := modelsettings.Key{ConnectionID: edit.ConnectionID, Purpose: edit.Purpose, Model: edit.Model}
		c, ok := state.entries[edit.ConnectionID]
		if !ok || !c.Supports(edit.Purpose) || state.selected[edit.Purpose] != edit.ConnectionID || seen[key] {
			return errors.New("model edit owner changed or is duplicated; reload settings")
		}
		seen[key] = true
		e := modelsettings.Entry{ConnectionID: edit.ConnectionID, Purpose: edit.Purpose, Model: edit.Model, Options: edit.Options}
		if err := modelsettings.Validate(e, c.Details); err != nil {
			return err
		}
		found := false
		count := 0
		for i, old := range state.models {
			if old.ConnectionID == e.ConnectionID && old.Purpose == e.Purpose {
				count++
				if old.Model == e.Model {
					e.Selected = old.Selected
					state.models[i] = e
					found = true
				}
			}
		}
		if !found {
			if count >= modelsettings.MaxPerUse {
				return errors.New("this connection already has 32 remembered models for this feature; forget an unused model first")
			}
			state.models = append(state.models, e)
		}
	}
	s.pendingConnections = &state
	return nil
}
