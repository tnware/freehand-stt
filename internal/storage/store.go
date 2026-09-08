// Package storage owns durable non-secret configuration, generated queries,
// database lifecycle, and credential-reference persistence. It exposes no Wails API.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/credential"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

const applicationID = 1179796804 // FRHD: database identity, not a schema version.
const operationTimeout = 10 * time.Second

type Store struct {
	connections             connectionState
	pendingConnections      *connectionState
	pendingConnectionTarget string
	mu                      sync.Mutex
	path, legacy            string
	db                      *sql.DB
	lock                    *os.File
	closed                  bool
	uncertain               bool
	ctx                     context.Context
	cancel                  context.CancelFunc
	migrations              fs.FS
	refs                    map[string]string
	pending                 map[string]string
	vault                   Vault
}

// NewStore does not open or modify any files. Load owns initialization/recovery.
func NewStore() (*Store, error) {
	local, err := localDataDir()
	if err != nil {
		return nil, err
	}
	legacy, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return newStore(filepath.Join(local, "Freehand", "settings.db"), filepath.Join(legacy, "Freehand", "settings.json"), nativeVault{}), nil
}
func newStore(path, legacy string, vault Vault) *Store {
	migrations, _ := fs.Sub(embeddedMigrations, "migrations")
	ctx, cancel := context.WithCancel(context.Background())
	return &Store{ctx: ctx, cancel: cancel, path: path, legacy: legacy, migrations: migrations, vault: vault, refs: map[string]string{}}
}
func (s *Store) LoadReport() config.LoadReport { return config.LoadReport{} }
func (s *Store) Load() (config.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()
	if s.closed {
		return config.Default(), failure("closed", nil)
	}
	if err := s.open(ctx); err != nil {
		return config.Default(), err
	}
	if s.uncertain {
		if err := s.validateDatabase(ctx, s.db); err != nil {
			return config.Default(), err
		}
		if err := checkIntegrity(ctx, s.db); err != nil {
			return config.Default(), failure("corrupt", err)
		}
	}

	v, err := readSettings(ctx, dbgen.New(s.db))
	if err != nil {
		return config.Default(), failure("corrupt", err)
	}
	if err = config.Validate(v); err != nil {
		return config.Default(), failure("invalid_values", err)
	}
	if err = s.loadReferences(ctx); err != nil {
		return config.Default(), failure("corrupt", err)
	}
	state, err := readConnections(ctx, dbgen.New(s.db), v, s.refs)
	if err != nil {
		return config.Default(), failure("corrupt", err)
	}
	s.connections = state
	s.pendingConnections = nil
	s.pending = nil
	s.uncertain = false
	s.collectCredentials(ctx) // Failed deletions stay durably queued for a later attempt.
	return v, nil
}
func (s *Store) Save(v config.Settings) error {
	if err := config.Validate(v); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil || s.closed || s.uncertain {
		return failure("unavailable", nil)
	}
	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return failure("write_failed", err)
	}
	defer tx.Rollback()
	q := dbgen.New(tx)
	if err = writeSettings(ctx, q, v); err != nil {
		return failure("write_failed", err)
	}
	if s.pending != nil {
		for _, purpose := range purposes {
			next, old := s.pending[purpose], s.refs[purpose]
			if err = q.PutCredentialRef(ctx, dbgen.PutCredentialRefParams{Purpose: purpose, Account: next}); err != nil {
				return failure("write_failed", err)
			}
			if old != "" && old != next {
				if err = q.QueueCredentialGC(ctx, old); err != nil {
					return failure("write_failed", err)
				}
			}
			if next != "" {
				if err = q.CompleteCredentialGC(ctx, next); err != nil {
					return failure("write_failed", err)
				}
			}
		}
	}
	nextConnections, err := s.writeConnections(ctx, q, v)
	if err != nil {
		return failure("write_failed", err)
	}
	if err = writeRememberedModels(ctx, q, &nextConnections, v); err != nil {
		return failure("write_failed", err)
	}
	if err = tx.Commit(); err != nil {
		s.uncertain = true
		// A failed COMMIT may leave a driver transaction open. Drop the handle
		// so recovery must reopen and observe durable state, never that transaction.
		closeErr := s.db.Close()
		s.db = nil
		return failure("commit_uncertain", errors.Join(err, closeErr))
	}
	if s.pending != nil {
		s.refs = s.pending
		s.pending = nil
	}
	s.connections = nextConnections
	s.pendingConnections = nil
	s.collectCredentials(ctx)
	return nil
}
func (s *Store) Close() error {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return s.closeDatabase()
}
func (s *Store) closeDatabase() error {
	var err error
	if s.db != nil {
		err = s.db.Close()
		s.db = nil
	}
	if s.lock != nil {
		err = errors.Join(err, s.lock.Close())
		s.lock = nil
	}
	return err
}
func databaseURI(path, mode string) string {
	path = filepath.ToSlash(path)
	if len(path) > 1 && path[1] == ':' {
		path = "/" + path
	}
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {mode}, "_pragma": {"foreign_keys(1)", "busy_timeout(1500)", "synchronous(EXTRA)", "trusted_schema(0)"}, "_txlock": {"immediate"}}
	u.RawQuery = q.Encode()
	return u.String()
}
func openDatabase(path, mode string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", databaseURI(path, mode))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}
func (s *Store) open(ctx context.Context) (retErr error) {
	if s.db != nil {
		return nil
	}
	if err := secureDirectory(filepath.Dir(s.path)); err != nil {
		return failure("unreadable", err)
	}
	if s.lock == nil {
		lock, err := lockDatabase(s.path + ".lock")
		if err != nil {
			return failure("locked", err)
		}
		s.lock = lock
	}

	defer func() {
		if retErr != nil {
			_ = s.closeDatabase()
		}
	}()
	info, err := os.Stat(s.path)
	exists := err == nil
	if err != nil && !os.IsNotExist(err) {
		return failure("unreadable", err)
	}
	if exists && (!info.Mode().IsRegular() || info.Size() == 0) {
		return failure("corrupt", nil)
	}
	initial := config.Default()
	source := "defaults"

	if !exists && s.legacy != "" {
		legacy := &config.LegacyReader{Path: s.legacy}
		initial, err = legacy.Load()
		if err != nil {
			return failure("legacy_invalid", err)
		}
		if legacy.LoadReport().PreservedFieldCount > 0 {
			return failure("legacy_newer", nil)
		}
		if _, err = os.Stat(s.legacy); err == nil {
			source = "legacy"
		} else if !os.IsNotExist(err) {
			return failure("legacy_invalid", err)
		}
	}
	if !exists {
		// Build privately. Publishing the fully initialized file is the import commit point.
		temp, err := os.CreateTemp(filepath.Dir(s.path), "settings-init-*.db")
		if err != nil {
			return failure("write_failed", err)
		}
		tempPath := temp.Name()
		if err = temp.Close(); err != nil {
			return failure("write_failed", err)
		}
		defer os.Remove(tempPath)
		db, err := openDatabase(tempPath, "rw")
		if err != nil {
			return failure("write_failed", err)
		}
		if err = s.initialize(ctx, db, initial, source); err != nil {
			_ = db.Close()
			return failure("write_failed", err)
		}
		if err = db.Close(); err != nil {
			return failure("write_failed", err)
		}
		if err = syncFile(tempPath); err != nil {
			return failure("write_failed", err)
		}
		if err = os.Rename(tempPath, s.path); err != nil {
			return failure("write_failed", err)
		}
	}
	s.db, err = openDatabase(s.path, "rw")
	if err != nil {
		return failure("unreadable", err)
	}
	if err = s.validateDatabase(ctx, s.db); err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, s.db, s.migrations, goose.WithLogger(goose.NopLogger()))
	if err != nil {
		return failure("unavailable", err)
	}
	current, err := provider.GetDBVersion(ctx)
	if err != nil {
		return failure("corrupt", err)
	}
	sources := provider.ListSources()
	target := sources[len(sources)-1].Version
	if current < target {
		if _, err = s.backup(ctx, s.db); err != nil {
			return failure("backup_failed", err)
		}
		if _, err = provider.Up(ctx); err != nil {
			return failure("migration_failed", err)
		}
	}
	if err = checkIntegrity(ctx, s.db); err != nil {
		return failure("corrupt", err)
	}
	var journal string
	if err = s.db.QueryRowContext(ctx, "PRAGMA journal_mode=DELETE").Scan(&journal); err != nil || journal != "delete" {
		return failure("unavailable", err)
	}
	var fk, sync int
	if err = s.db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		return failure("unavailable", err)
	}
	if err = s.db.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&sync); err != nil || sync != 3 {
		return failure("unavailable", err)
	}
	return nil
}
func (s *Store) initialize(ctx context.Context, db *sql.DB, v config.Settings, source string) error {
	if source == "legacy" {
		v.VoiceTranscription = config.VoiceFromCompleted(v)
	}
	p, err := goose.NewProvider(goose.DialectSQLite3, db, s.migrations, goose.WithLogger(goose.NopLogger()))
	if err != nil {
		return err
	}
	if _, err = p.Up(ctx); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := dbgen.New(tx)
	if err = writeSettings(ctx, q, v); err != nil {
		return err
	}
	for _, purpose := range purposes {
		account := ""
		if source == "legacy" {
			account = legacyAccount(purpose)
			if purpose == "voice" {
				account = legacyAccount("stt")
			}
		}
		if source == "reset" {
			account = s.refs[purpose]
		}
		if err = q.PutCredentialRef(ctx, dbgen.PutCredentialRefParams{Purpose: purpose, Account: account}); err != nil {
			return err
		}
	}
	if err = seedConnections(ctx, q); err != nil {
		return err
	}
	if err = q.Initialize(ctx, source); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA application_id=%d", applicationID)); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) validateDatabase(ctx context.Context, db *sql.DB) error {
	var id int
	if err := db.QueryRowContext(ctx, "PRAGMA application_id").Scan(&id); err != nil {
		return failure("corrupt", err)
	}
	if id != applicationID {
		return failure("foreign_database", nil)
	}
	known := map[int64]bool{0: true}
	entries, err := fs.ReadDir(s.migrations, ".")
	if err != nil {
		return failure("unavailable", err)
	}
	for _, entry := range entries {
		var version int64
		if _, err = fmt.Sscanf(strings.SplitN(entry.Name(), "_", 2)[0], "%d", &version); err != nil {
			return failure("unavailable", err)
		}
		known[version] = true
	}
	rows, err := db.QueryContext(ctx, "SELECT version_id, is_applied FROM goose_db_version ORDER BY id")
	if err != nil {
		return failure("corrupt", err)
	}
	defer rows.Close()
	applied := map[int64]bool{}
	for rows.Next() {
		var version int64
		var ok bool
		if err = rows.Scan(&version, &ok); err != nil {
			return failure("corrupt", err)
		}
		if !known[version] {
			return failure("newer_schema", nil)
		}
		applied[version] = ok
	}
	if err = rows.Err(); err != nil {
		return failure("corrupt", err)
	}
	var latest int64
	for version, ok := range applied {
		if !ok {
			return failure("corrupt", nil)
		}
		if version > latest {
			latest = version
		}
	}
	for version := range known {
		if version <= latest && !applied[version] {
			return failure("corrupt", nil)
		}
	}
	if !applied[0] || latest == 0 {
		return failure("corrupt", nil)
	}
	return nil
}
func checkIntegrity(ctx context.Context, db *sql.DB) error {
	var result string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return errors.New("integrity check failed")
	}
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("foreign key check failed")
	}
	return rows.Err()
}
func syncFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

type storageError struct {
	kind  string
	cause error
}

func (e *storageError) Error() string { return e.ConfigurationFailure().Message }
func (e *storageError) Unwrap() error { return e.cause }

// DiagnosticKind exposes the category, never the driver error or database path.
func (e *storageError) DiagnosticKind() string { return e.kind }
func (e *storageError) ConfigurationFailure() config.LoadFailure {
	messages := map[string]string{
		"commit_uncertain": "The save outcome could not be confirmed. Reload saved settings before continuing.",
		"locked":           "The settings database is in use. Close other Freehand instances and retry.",
		"newer_schema":     "This settings database needs a newer Freehand version. Update Freehand or restore a compatible backup.",
		"foreign_database": "The selected file is not a Freehand settings database.",
		"corrupt":          "The settings database could not pass validation. Restore a backup or explicitly reset it.",
		"legacy_invalid":   "The previous settings file could not be imported. Repair it and retry, or explicitly reset settings.",
		"legacy_newer":     "The previous settings file contains unsupported fields. Use a compatible version or explicitly reset settings.",
		"write_failed":     "Settings could not be committed. Check available disk space and file access.",
		"backup_failed":    "A settings backup could not be created. The database was not upgraded.",
		"migration_failed": "The settings database could not be upgraded. Its backup remains available.",
		"unreadable":       "The settings database could not be opened. Check file access and retry.",
		"invalid_values":   "The saved settings contain unsupported values. Restore a backup or explicitly reset settings.",
	}
	message := messages[e.kind]
	if message == "" {
		message = "The settings database is unavailable."
	}
	return config.LoadFailure{Kind: e.kind, Message: message}
}
func failure(kind string, cause error) error { return &storageError{kind, cause} }

var _ credential.Store = (*credentialView)(nil)
