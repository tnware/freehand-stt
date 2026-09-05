package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"reflect"
	"sort"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

type storedConnection struct {
	savedconnection.Connection
	account string
}
type connectionState struct {
	entries  map[string]storedConnection
	selected map[savedconnection.Purpose]string
}

func (state connectionState) clone() connectionState {
	next := connectionState{entries: map[string]storedConnection{}, selected: map[savedconnection.Purpose]string{}}
	for id, c := range state.entries {
		c.Details = savedconnection.CloneDetails(c.Details)
		next.entries[id] = c
	}
	for p, id := range state.selected {
		next.selected[p] = id
	}
	return next
}
func (s *Store) ConnectionCatalog() savedconnection.Catalog {
	s.mu.Lock()
	defer s.mu.Unlock()
	catalog := savedconnection.Catalog{Entries: []savedconnection.Connection{}, Selected: map[savedconnection.Purpose]string{}}
	for _, stored := range s.connections.entries {
		c := stored.Connection
		c.Details = savedconnection.CloneDetails(c.Details)
		c.HasCredential = stored.account != ""
		catalog.Entries = append(catalog.Entries, c)
	}
	sort.Slice(catalog.Entries, func(i, j int) bool {
		a, b := catalog.Entries[i], catalog.Entries[j]
		if a.Purpose != b.Purpose {
			return a.Purpose < b.Purpose
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	for p, id := range s.connections.selected {
		catalog.Selected[p] = id
	}
	return catalog
}
func seedConnections(ctx context.Context, q *dbgen.Queries) error {
	for _, seed := range []func(context.Context) error{q.SeedTranscriptionConnection, q.SeedCleanupConnection, q.SeedSpeechConnection, q.SeedSelectedConnections, q.SeedConnectionHeaders} {
		if err := seed(ctx); err != nil {
			return err
		}
	}
	return nil
}
func readConnections(ctx context.Context, q *dbgen.Queries, v config.Settings, refs map[string]string) (connectionState, error) {
	state := connectionState{entries: map[string]storedConnection{}, selected: map[savedconnection.Purpose]string{}}
	rows, err := q.ListSavedConnections(ctx)
	if err != nil {
		return state, err
	}
	if len(rows) > savedconnection.MaxPerPurpose*3 {
		return state, errors.New("too many saved connections")
	}
	counts := map[savedconnection.Purpose]int{}
	for _, row := range rows {
		p := savedconnection.Purpose(row.Purpose)
		counts[p]++
		if !savedconnection.ValidPurpose(p) || counts[p] > savedconnection.MaxPerPurpose || savedconnection.ValidateName(row.Name) != nil {
			return state, errors.New("invalid saved connection")
		}
		if row.CredentialAccount != "" && row.CredentialAccount != legacyAccount(row.Purpose) && !ownedAccount(row.Purpose, row.CredentialAccount) {
			return state, errors.New("invalid connection credential reference")
		}
		state.entries[row.ID] = storedConnection{Connection: savedconnection.Connection{ID: row.ID, Name: row.Name, Purpose: p, Details: savedconnection.Details{
			CompatibilityProfile: compatibility.ID(row.CompatibilityProfile), BaseURL: row.BaseUrl, AllowInsecureHTTP: row.AllowInsecureHttp != 0, AuthenticationMode: config.AuthenticationMode(row.AuthenticationMode), Model: row.Model, HealthPath: row.HealthPath, CleanupPreset: config.PostProcessingPreset(row.CleanupPreset), Headers: map[string]string{},
		}}, account: row.CredentialAccount}
	}
	headers, err := q.ListConnectionHeaders(ctx)
	if err != nil {
		return state, err
	}
	if len(headers) > savedconnection.MaxPerPurpose*3*config.MaxHeaderCount {
		return state, errors.New("too many saved headers")
	}
	for _, h := range headers {
		c, ok := state.entries[h.ConnectionID]
		if !ok || c.Purpose != savedconnection.Transcription {
			return state, errors.New("invalid saved headers")
		}
		c.Details.Headers[h.Name] = h.Value
		if len(c.Details.Headers) > config.MaxHeaderCount {
			return state, errors.New("too many saved headers")
		}
		state.entries[h.ConnectionID] = c
	}
	selected, err := q.ListSelectedConnections(ctx)
	if err != nil {
		return state, err
	}
	if len(selected) != 3 {
		return state, errors.New("missing connection selections")
	}
	for _, row := range selected {
		p := savedconnection.Purpose(row.Purpose)
		c, ok := state.entries[row.ConnectionID]
		if !ok || c.Purpose != p || c.account != refs[row.Purpose] || !reflect.DeepEqual(c.Details, savedconnection.Extract(v, p)) {
			return state, errors.New("connection selection does not match committed settings")
		}
		state.selected[p] = row.ConnectionID
	}
	return state, nil
}

// BeginConnectionChange stages one action under the settings service's save lock.
// No native credential or SQL state changes until the enclosing settings save.
func (s *Store) BeginConnectionChange(change savedconnection.Change, v config.Settings) (config.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.db == nil || s.uncertain || s.pending != nil {
		return v, failure("unavailable", nil)
	}
	if !savedconnection.ValidPurpose(change.Purpose) {
		return v, errors.New("invalid connection purpose")
	}
	state := s.connections.clone()
	p := change.Purpose
	current, ok := state.entries[change.ID]
	if change.Action != savedconnection.Create && (!ok || current.Purpose != p) {
		return v, errors.New("saved connection is unavailable; reload settings")
	}
	name := strings.TrimSpace(change.Name)
	if change.Action == savedconnection.Create || change.Action == savedconnection.Duplicate || change.Action == savedconnection.Rename {
		if err := savedconnection.ValidateName(name); err != nil {
			return v, err
		}
		for id, c := range state.entries {
			if c.Purpose == p && strings.EqualFold(c.Name, name) && (change.Action != savedconnection.Rename || id != change.ID) {
				return v, errors.New("a connection with that name already exists")
			}
		}
	}
	switch change.Action {
	case savedconnection.Create, savedconnection.Duplicate:
		count := 0
		for _, c := range state.entries {
			if c.Purpose == p {
				count++
			}
		}
		if count >= savedconnection.MaxPerPurpose {
			return v, errors.New("each capability can save at most 32 connections")
		}
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			return v, failure("unavailable", err)
		}
		c := storedConnection{Connection: savedconnection.Connection{ID: "connection-" + hex.EncodeToString(id[:]), Purpose: p, Name: name, Details: savedconnection.Extract(v, p)}}
		if change.Action == savedconnection.Duplicate {
			c.Details = savedconnection.CloneDetails(current.Details)
			c.account = current.account
		}
		state.entries[c.ID] = c
		state.selected[p] = c.ID
		v = savedconnection.Apply(v, p, c.Details)
	case savedconnection.Select:
		state.selected[p] = current.ID
		v = savedconnection.Apply(v, p, current.Details)
	case savedconnection.Rename:
		current.Name = name
		state.entries[current.ID] = current
	case savedconnection.Delete:
		if state.selected[p] == current.ID {
			replacement, ok := state.entries[change.ReplacementID]
			if !ok || replacement.Purpose != p || replacement.ID == current.ID {
				return v, errors.New("select another connection before deleting this one")
			}
			state.selected[p] = replacement.ID
			v = savedconnection.Apply(v, p, replacement.Details)
		}
		delete(state.entries, current.ID)
	default:
		return v, errors.New("invalid connection action")
	}
	pending := map[string]string{}
	for _, purpose := range purposes {
		pending[purpose] = state.entries[state.selected[savedconnection.Purpose(purpose)]].account
	}
	s.pending = pending
	s.pendingConnections = &state
	return v, nil
}
func (s *Store) writeConnections(ctx context.Context, q *dbgen.Queries, v config.Settings) (connectionState, error) {
	state := s.connections.clone()
	if s.pendingConnections != nil {
		state = s.pendingConnections.clone()
	}
	for _, purpose := range purposes {
		p := savedconnection.Purpose(purpose)
		id := state.selected[p]
		c, ok := state.entries[id]
		if !ok {
			return state, errors.New("missing selected connection")
		}
		c.Details = savedconnection.Extract(v, p)
		c.account = s.refs[purpose]
		if s.pending != nil {
			c.account = s.pending[purpose]
		}
		state.entries[id] = c
	}
	ids := make([]string, 0, len(state.entries))
	for id := range state.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := state.entries[id]
		d := c.Details
		if err := q.PutSavedConnection(ctx, dbgen.PutSavedConnectionParams{ID: c.ID, Purpose: string(c.Purpose), Name: c.Name, CompatibilityProfile: string(d.CompatibilityProfile), BaseUrl: d.BaseURL, AllowInsecureHttp: boolean(d.AllowInsecureHTTP), AuthenticationMode: string(d.AuthenticationMode), Model: d.Model, HealthPath: d.HealthPath, CleanupPreset: string(d.CleanupPreset), CredentialAccount: c.account}); err != nil {
			return state, err
		}
		if err := q.ClearConnectionHeaders(ctx, id); err != nil {
			return state, err
		}
		names := make([]string, 0, len(d.Headers))
		for name := range d.Headers {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := q.PutConnectionHeader(ctx, dbgen.PutConnectionHeaderParams{ConnectionID: id, Name: name, Value: d.Headers[name]}); err != nil {
				return state, err
			}
		}
	}
	for _, purpose := range purposes {
		p := savedconnection.Purpose(purpose)
		if err := q.SelectSavedConnection(ctx, dbgen.SelectSavedConnectionParams{Purpose: purpose, ConnectionID: state.selected[p]}); err != nil {
			return state, err
		}
	}
	for id, old := range s.connections.entries {
		next, exists := state.entries[id]
		if !exists {
			if err := q.DeleteSavedConnection(ctx, id); err != nil {
				return state, err
			}
		}
		if old.account != "" && (!exists || old.account != next.account) {
			if err := q.QueueCredentialGC(ctx, old.account); err != nil {
				return state, err
			}
		}
	}
	return state, nil
}
