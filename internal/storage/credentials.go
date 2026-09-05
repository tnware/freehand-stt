package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

// Vault is the native credential boundary; only opaque account references enter SQL.
type Vault interface {
	Get(string) (string, error)
	Set(string, string) error
	Delete(string) error
}
type nativeVault struct{}

func (nativeVault) Get(account string) (string, error) {
	return (credential.Keyring{Account: account}).Get()
}
func (nativeVault) Set(account, value string) error {
	return (credential.Keyring{Account: account}).Set(value)
}
func (nativeVault) Delete(account string) error {
	return (credential.Keyring{Account: account}).Delete()
}

const maxPendingCredentials = 128

var purposes = []string{"stt", "cleanup", "speech"}

func legacyAccount(purpose string) string {
	switch purpose {
	case "stt":
		return credential.STTAccount
	case "cleanup":
		return credential.PostProcessingAccount
	case "speech":
		return credential.TextToSpeechAccount
	}
	return ""
}
func (s *Store) STTCredentials() credential.Store     { return &credentialView{s, "stt"} }
func (s *Store) CleanupCredentials() credential.Store { return &credentialView{s, "cleanup"} }
func (s *Store) SpeechCredentials() credential.Store  { return &credentialView{s, "speech"} }

type credentialView struct {
	s       *Store
	purpose string
}

func (v *credentialView) Get() (string, error) {
	v.s.mu.Lock()
	defer v.s.mu.Unlock()
	account := v.s.refs[v.purpose]
	if account == "" {
		return "", credential.ErrNotFound
	}
	return v.s.vault.Get(account)
}
func (v *credentialView) Configured() bool { _, err := v.Get(); return err == nil }
func (v *credentialView) Set(value string) error {
	if value == "" || len(value) > 16*1024 {
		return errors.New("invalid credential length")
	}
	s := v.s
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil || s.db == nil || s.closed {
		return errors.New("credential change requires a settings transaction")
	}
	account, err := s.stageCredential(v.purpose, value)
	if err == nil {
		s.pending[v.purpose] = account
	}
	return err
}

// stageCredential is called with the store mutex held. It creates a fresh account,
// with durable cleanup intent, so failed saves cannot overwrite a committed key.
func (s *Store) stageCredential(purpose, value string) (string, error) {
	if len(value) == 0 || len(value) > 16*1024 || s.db == nil || s.closed {
		return "", errors.New("invalid credential transaction")
	}
	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()
	count, err := dbgen.New(s.db).CountCredentialGC(ctx)
	if err != nil || count >= maxPendingCredentials {
		return "", errors.New("pending credential cleanup must complete before replacing a key")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	account := "sqlite-" + purpose + "-" + hex.EncodeToString(id[:])
	// Persist cleanup intent first, so a crash after the keyring write leaves a reclaimable secret.
	if err := dbgen.New(s.db).QueueCredentialGC(ctx, account); err != nil {
		return "", failure("write_failed", err)
	}
	if err := s.vault.Set(account, value); err != nil {
		return "", errors.New("credential could not be stored")
	}
	return account, nil
}

func (v *credentialView) Delete() error {
	s := v.s
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		return errors.New("credential change requires a settings transaction")
	}
	s.pending[v.purpose] = ""
	return nil
}
func (s *Store) BeginCredentialChanges() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.db == nil || s.pending != nil || s.uncertain {
		return failure("unavailable", nil)
	}
	s.pending = map[string]string{}
	for k, v := range s.refs {
		s.pending[k] = v
	}
	return nil
}
func (s *Store) DiscardCredentialChanges() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = nil
	s.pendingConnections = nil
	s.pendingConnectionTarget = ""
	if s.db != nil && !s.uncertain {
		ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
		defer cancel()
		s.collectCredentials(ctx)
	}
}
func (s *Store) loadReferences(ctx context.Context) error {
	refs, err := readReferences(ctx, dbgen.New(s.db))
	if err == nil {
		s.refs = refs
	}
	return err
}
func readReferences(ctx context.Context, q *dbgen.Queries) (map[string]string, error) {
	rows, err := q.GetCredentialRefs(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) != len(purposes) {
		return nil, errors.New("missing credential references")
	}
	refs := map[string]string{}
	for _, r := range rows {
		if legacyAccount(r.Purpose) == "" || (r.Account != "" && r.Account != legacyAccount(r.Purpose) && !ownedAccount(r.Purpose, r.Account)) {
			return nil, errors.New("invalid credential reference")
		}
		refs[r.Purpose] = r.Account
	}
	return refs, nil
}
func (s *Store) collectCredentials(ctx context.Context) {
	q := dbgen.New(s.db)
	accounts, err := q.PendingCredentialGC(ctx)
	if err != nil {
		return
	}
	for _, account := range accounts {
		if ctx.Err() != nil {
			return
		}
		owned := false
		for _, p := range purposes {
			owned = owned || ownedAccount(p, account) || account == legacyAccount(p)
		}
		if !owned {
			continue
		}
		if err = s.vault.Delete(account); err == nil {
			_ = q.CompleteCredentialGC(ctx, account)
		}
	}
}

func ownedAccount(purpose, account string) bool {
	prefix := "sqlite-" + purpose + "-"
	if !strings.HasPrefix(account, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(account, prefix)
	if len(suffix) != 32 {
		return false
	}
	_, err := hex.DecodeString(suffix)
	return err == nil
}
