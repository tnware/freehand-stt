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
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

type storedConnection struct {
	savedconnection.Connection
	account string
}
type connectionState struct {
	models   []modelsettings.Entry
	entries  map[string]storedConnection
	selected map[savedconnection.Purpose]string
}

func (state connectionState) clone() connectionState {
	next := connectionState{entries: map[string]storedConnection{}, selected: map[savedconnection.Purpose]string{}}
	next.models = append([]modelsettings.Entry{}, state.models...)
	for id, c := range state.entries {
		c.Uses = append([]savedconnection.Purpose{}, c.Uses...)
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
		c.Uses = append([]savedconnection.Purpose{}, c.Uses...)
		c.Details = savedconnection.CloneDetails(c.Details)
		c.HasCredential = stored.account != ""
		catalog.Entries = append(catalog.Entries, c)
	}
	sort.Slice(catalog.Entries, func(i, j int) bool {
		a, b := catalog.Entries[i], catalog.Entries[j]
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	for p, id := range s.connections.selected {
		catalog.Selected[p] = id
	}
	return catalog
}
func seedConnections(ctx context.Context, q *dbgen.Queries) error {
	for _, seed := range []func(context.Context) error{q.SeedTranscriptionConnection, q.SeedCleanupConnection, q.SeedSpeechConnection, q.SeedConnectionUses, q.SeedSelectedConnections, q.SeedConnectionHeaders} {
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
	for _, row := range rows {
		if savedconnection.ValidateName(row.Name) != nil || (row.CredentialAccount != "" && !connectionAccount(row.CredentialAccount)) {
			return state, errors.New("invalid saved connection")
		}
		state.entries[row.ID] = storedConnection{Connection: savedconnection.Connection{ID: row.ID, Name: row.Name, Uses: []savedconnection.Purpose{}, Details: savedconnection.Details{CompatibilityProfile: compatibility.ID(row.CompatibilityProfile), BaseURL: row.BaseUrl, AllowInsecureHTTP: row.AllowInsecureHttp != 0, AuthenticationMode: config.AuthenticationMode(row.AuthenticationMode), HealthPath: row.HealthPath, Headers: map[string]string{}}}, account: row.CredentialAccount}
	}
	uses, err := q.ListConnectionUses(ctx)
	if err != nil {
		return state, err
	}
	counts := map[savedconnection.Purpose]int{}
	for _, use := range uses {
		p := savedconnection.Purpose(use.Purpose)
		c, ok := state.entries[use.ConnectionID]
		counts[p]++
		if !ok || !savedconnection.ValidPurpose(p) || counts[p] > savedconnection.MaxPerPurpose {
			return state, errors.New("invalid connection uses")
		}
		c.Uses = append(c.Uses, p)
		state.entries[c.ID] = c
	}
	for _, c := range state.entries {
		if len(c.Uses) == 0 {
			return state, errors.New("connection has no supported uses")
		}
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
		if !ok || !c.Supports(savedconnection.Transcription) {
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
	if len(selected) > 3 {
		return state, errors.New("missing connection selections")
	}
	for _, row := range selected {
		p := savedconnection.Purpose(row.Purpose)
		c, ok := state.entries[row.ConnectionID]
		if !ok || !c.Supports(p) || c.account != refs[row.Purpose] || !reflect.DeepEqual(savedconnection.Project(c.Details, p), savedconnection.Extract(v, p)) {
			return state, errors.New("connection selection does not match committed settings")
		}
		state.selected[p] = row.ConnectionID
	}
	if err := readRememberedModels(ctx, q, &state); err != nil {
		return state, err
	}
	// Legacy import creates connections after schema migration; capture active choices too.
	if err := rememberActiveModels(&state, v); err != nil {
		return state, err
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
	if change.Action == savedconnection.Select && !savedconnection.ValidPurpose(change.Purpose) {
		return v, errors.New("invalid connection purpose")
	}
	state := s.connections.clone()
	p := change.Purpose
	current, ok := state.entries[change.ID]
	if change.Action != savedconnection.Create && !(change.Action == savedconnection.Select && change.ID == "") && (!ok || (change.Action == savedconnection.Select && !current.Supports(p))) {
		return v, errors.New("saved connection is unavailable; reload settings")
	}
	name := strings.TrimSpace(change.Name)
	if change.Action == savedconnection.Create || change.Action == savedconnection.Duplicate || change.Action == savedconnection.Rename || change.Action == savedconnection.Update {
		if err := savedconnection.ValidateName(name); err != nil {
			return v, err
		}
		for id, c := range state.entries {
			if strings.EqualFold(c.Name, name) && ((change.Action != savedconnection.Rename && change.Action != savedconnection.Update) || id != change.ID) {
				return v, errors.New("a connection with that name already exists")
			}
		}
	}

	target := ""
	switch change.Action {
	case savedconnection.Create, savedconnection.Duplicate:
		if len(state.entries) >= savedconnection.MaxPerPurpose*3 {
			return v, errors.New("connection limit reached")
		}
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			return v, failure("unavailable", err)
		}
		c := storedConnection{Connection: savedconnection.Connection{ID: "connection-" + hex.EncodeToString(id[:]), Uses: append([]savedconnection.Purpose{}, change.Uses...), Name: name}}
		if change.Action == savedconnection.Duplicate {
			c.Details = savedconnection.CloneDetails(current.Details)
			c.Uses = append([]savedconnection.Purpose{}, current.Uses...)
			c.account = current.account
		} else {
			if change.Details == nil {
				return v, errors.New("connection details are required")
			}
			c.Details = savedconnection.CloneDetails(*change.Details)
			if err := savedconnection.ValidateUses(c.Uses, c.Details); err != nil {
				return v, err
			}
			target = c.ID
		}
		state.entries[c.ID] = c
	case savedconnection.Update:
		if change.Details == nil {
			return v, errors.New("connection details are required")
		}
		if err := savedconnection.ValidateUses(change.Uses, *change.Details); err != nil {
			return v, err
		}
		modelIdentityChanged := current.Details.BaseURL != change.Details.BaseURL || current.Details.CompatibilityProfile != change.Details.CompatibilityProfile
		if modelIdentityChanged {
			state.forgetModels(current.ID)
		}
		current.Name = name
		current.Uses = append([]savedconnection.Purpose{}, change.Uses...)
		current.Details = savedconnection.CloneDetails(*change.Details)
		state.entries[current.ID] = current
		target = current.ID
		for role, id := range state.selected {
			if id == current.ID {
				if !current.Supports(role) {
					return v, errors.New("deselect this connection from the feature before removing its use")
				}
				v = savedconnection.Apply(v, role, current.Details)
				if modelIdentityChanged {
					v = state.restoreModel(savedconnection.ClearModel(v, role), role, current.ID)
				}
			}
		}
	case savedconnection.Select:
		previous := v
		selectionChanged := state.selected[p] != change.ID
		if selectionChanged {
			v = savedconnection.ClearModel(v, p)
		}
		if change.ID == "" {
			delete(state.selected, p)
			v = savedconnection.Apply(v, p, savedconnection.Extract(config.Default(), p))
		} else {
			state.selected[p] = current.ID
			v = savedconnection.Apply(v, p, current.Details)
		}
		if selectionChanged {
			v = state.restoreModel(v, p, change.ID)
			if modelsettings.Model(v, p) != "" {
				switch p {
				case savedconnection.Transcription:
					v.SetupCompleted = previous.SetupCompleted
				case savedconnection.Cleanup:
					v.PostProcessing.Enabled = previous.PostProcessing.Enabled
				case savedconnection.Speech:
					v.TextToSpeech.Enabled = previous.TextToSpeech.Enabled
				}
			}
		}
	case savedconnection.Rename:
		current.Name = name
		state.entries[current.ID] = current
	case savedconnection.Delete:
		for _, id := range state.selected {
			if id == current.ID {
				return v, errors.New("choose another connection or None in every active feature before deleting this connection")
			}
		}
		delete(state.entries, current.ID)
	default:
		return v, errors.New("invalid connection action")
	}
	counts := map[savedconnection.Purpose]int{}
	for _, c := range state.entries {
		for _, role := range c.Uses {
			counts[role]++
			if counts[role] > savedconnection.MaxPerPurpose {
				return v, errors.New("each feature supports at most 32 connections")
			}
		}
	}
	pending := map[string]string{}
	for _, purpose := range purposes {
		pending[purpose] = state.entries[state.selected[savedconnection.Purpose(purpose)]].account
	}
	s.pending = pending
	s.pendingConnections = &state
	s.pendingConnectionTarget = target
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
			continue
		}
		c.account = s.refs[purpose]
		if s.pending != nil {
			c.account = s.pending[purpose]
		}
		state.entries[id] = c
	}
	if err := q.ClearSelectedConnections(ctx); err != nil {
		return state, err
	}
	if err := q.ClearConnectionUses(ctx); err != nil {
		return state, err
	}
	ids := make([]string, 0, len(state.entries))
	for id := range state.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := state.entries[id]
		d := c.Details
		if err := q.PutSavedConnection(ctx, dbgen.PutSavedConnectionParams{ID: c.ID, Name: c.Name, CompatibilityProfile: string(d.CompatibilityProfile), BaseUrl: d.BaseURL, AllowInsecureHttp: boolean(d.AllowInsecureHTTP), AuthenticationMode: string(d.AuthenticationMode), HealthPath: d.HealthPath, CredentialAccount: c.account}); err != nil {
			return state, err
		}
		for _, p := range c.Uses {
			if err := q.PutConnectionUse(ctx, dbgen.PutConnectionUseParams{ConnectionID: id, Purpose: string(p)}); err != nil {
				return state, err
			}
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
	if err := q.ClearSelectedConnections(ctx); err != nil {
		return state, err
	}
	for _, purpose := range purposes {
		p := savedconnection.Purpose(purpose)
		if state.selected[p] == "" {
			continue
		}
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

// ApplySelectedConnections keeps connection-owned fields out of ordinary runtime saves.
func (s *Store) ApplySelectedConnections(v config.Settings) config.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech} {
		if c, ok := s.connections.entries[s.connections.selected[p]]; ok {
			v = savedconnection.Apply(v, p, c.Details)
		} else {
			v = savedconnection.Apply(v, p, savedconnection.Extract(config.Default(), p))
			v = savedconnection.ClearModel(v, p)
		}
	}
	return v
}

// StageConnectionCredential changes only the explicit connection edit's native reference.
func (s *Store) StageConnectionCredential(value string, clear bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingConnections == nil || s.pendingConnectionTarget == "" {
		return errors.New("credentials require an explicit connection edit")
	}
	c := s.pendingConnections.entries[s.pendingConnectionTarget]
	if clear {
		c.account = ""
	} else if strings.TrimSpace(value) != "" {
		account, err := s.stageCredential("connection", value)
		if err != nil {
			return err
		}
		c.account = account
	}
	if c.Details.AuthenticationMode == config.AuthenticationModeNone {
		c.account = ""
	}
	s.pendingConnections.entries[c.ID] = c
	for p, id := range s.pendingConnections.selected {
		if id == c.ID {
			s.pending[string(p)] = c.account
		}
	}
	return nil
}

// ResolveSavedConnection captures metadata-test details and credentials for an internal caller.
// Store is not registered with Wails; stored keys never become binding results.
func (s *Store) ResolveSavedConnection(id string) (savedconnection.Connection, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.connections.entries[id]
	if !ok || s.closed || s.uncertain {
		return savedconnection.Connection{}, "", errors.New("saved connection is unavailable")
	}
	c.Details = savedconnection.CloneDetails(c.Details)
	key := ""
	if c.account != "" && c.Details.AuthenticationMode == config.AuthenticationModeAPIKey {
		var err error
		key, err = s.vault.Get(c.account)
		if err != nil {
			return savedconnection.Connection{}, "", err
		}
	} else if c.Details.AuthenticationMode == config.AuthenticationModeAPIKey {
		return savedconnection.Connection{}, "", credential.ErrNotFound
	}
	return c.Connection, key, nil
}
