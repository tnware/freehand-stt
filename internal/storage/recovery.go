package storage

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
	"modernc.org/sqlite"
)

const retainedBackups = 3

func (s *Store) backup(ctx context.Context, db *sql.DB) (string, error) {
	dir := filepath.Join(filepath.Dir(s.path), "backups")
	if err := secureDirectory(dir); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(dir, "settings-*.db")
	if err != nil {
		return "", err
	}
	path := temp.Name()
	if err = temp.Close(); err != nil {
		return "", err
	}
	if err = backupDatabase(ctx, db, path); err != nil {
		os.Remove(path)
		return "", err
	}
	// Complete and synced before retention removes any previous successful backup.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return path, err
	}
	type item struct {
		path string
		time time.Time
	}
	var files []item
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".db" {
			continue
		}
		info, e := entry.Info()
		if e != nil {
			return path, e
		}
		files = append(files, item{filepath.Join(dir, entry.Name()), info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].time.After(files[j].time) })
	for i := retainedBackups; i < len(files); i++ {
		if err = os.Remove(files[i].path); err != nil {
			return path, err
		}
	}
	return path, nil
}
func backupDatabase(ctx context.Context, db *sql.DB, path string) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	err = conn.Raw(func(raw any) error {
		driver, ok := raw.(interface {
			NewBackup(string) (*sqlite.Backup, error)
		})
		if !ok {
			return errors.New("backup API unavailable")
		}
		backup, err := driver.NewBackup(databaseURI(path, "rw"))
		if err != nil {
			return err
		}
		for {
			if err = ctx.Err(); err != nil {
				return errors.Join(err, backup.Finish())
			}
			more, e := backup.Step(128)
			if e != nil {
				return errors.Join(e, backup.Finish())
			}
			if !more {
				break
			}
		}
		return backup.Finish()
	})
	if err != nil {
		return err
	}
	return syncFile(path)
}

// Reset is explicit recovery only. Preserve an unreadable database and sidecars
// together; do not overwrite evidence or delete credentials. No normal Save calls this.
func (s *Store) Reset(v config.Settings) error {
	if err := config.Validate(v); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return failure("closed", nil)
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
	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()
	temp, err := os.CreateTemp(filepath.Dir(s.path), "settings-reset-*.db")
	if err != nil {
		return failure("write_failed", err)
	}
	tempPath := temp.Name()
	temp.Close()
	defer os.Remove(tempPath)
	replacement, err := openDatabase(tempPath, "rw")
	if err != nil {
		return failure("write_failed", err)
	}
	err = s.initialize(ctx, replacement, v, "reset")
	err = errors.Join(err, replacement.Close())
	if err != nil {
		return failure("write_failed", err)
	}
	if err = syncFile(tempPath); err != nil {
		return failure("write_failed", err)
	}
	if s.db != nil {
		if err = s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	if err = s.archiveDatabase(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, s.path); err != nil {
		return failure("write_failed", err)
	}
	if err = s.open(ctx); err != nil {
		return err
	}
	s.uncertain = false
	if err := s.loadReferences(ctx); err != nil {
		return err
	}
	cfg, err := readSettings(ctx, dbgen.New(s.db))
	if err != nil {
		return err
	}
	state, err := readConnections(ctx, dbgen.New(s.db), cfg, s.refs)
	if err != nil {
		return err
	}
	s.connections = state
	s.pendingConnections = nil
	s.pending = nil
	return nil
}

func (s *Store) archiveDatabase() error {
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return failure("unreadable", err)
	}
	dir, err := os.MkdirTemp(filepath.Dir(s.path), "settings-recovery-")
	if err != nil {
		return failure("write_failed", err)
	}
	// The lock is held and all our SQLite handles are closed. Keep rollback/WAL sidecars.
	for _, suffix := range []string{"", "-journal", "-wal", "-shm"} {
		old := s.path + suffix
		if _, err = os.Stat(old); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return failure("write_failed", err)
		}
		if err = copyClosedFile(old, filepath.Join(dir, filepath.Base(s.path)+suffix)); err != nil {
			return failure("write_failed", err)
		}
	}
	for _, suffix := range []string{"-journal", "-wal", "-shm"} {
		if err := os.Remove(s.path + suffix); err != nil && !os.IsNotExist(err) {
			return failure("write_failed", err)
		}
	}
	return nil
}
func copyClosedFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return errors.Join(err, out.Sync(), out.Close())
}

// RestoreBackup is the narrow native recovery operation. Call only with a
// trusted, user-selected local path; no SQL or arbitrary paths are exposed to Wails.
func (s *Store) RestoreBackup(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return failure("closed", nil)
	}
	ctx, cancel := context.WithTimeout(s.ctx, operationTimeout)
	defer cancel()
	source, err := openDatabase(path, "ro")
	if err != nil {
		return failure("unreadable", err)
	}
	defer source.Close()
	if err = s.validateDatabase(ctx, source); err != nil {
		return err
	}
	if err = checkIntegrity(ctx, source); err != nil {
		return failure("corrupt", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.path), "settings-restore-*.db")
	if err != nil {
		return failure("write_failed", err)
	}
	tempPath := temp.Name()
	temp.Close()
	defer os.Remove(tempPath)
	if err = backupDatabase(ctx, source, tempPath); err != nil {
		return failure("backup_failed", err)
	}
	// Upgrade and validate the private candidate before replacing any live files.
	candidate, err := openDatabase(tempPath, "rw")
	if err != nil {
		return failure("unreadable", err)
	}
	err = func() error {
		defer candidate.Close()
		provider, err := goose.NewProvider(goose.DialectSQLite3, candidate, s.migrations, goose.WithLogger(goose.NopLogger()))
		if err != nil {
			return failure("unavailable", err)
		}
		if _, err = provider.Up(ctx); err != nil {
			return failure("migration_failed", err)
		}
		if err = checkIntegrity(ctx, candidate); err != nil {
			return failure("corrupt", err)
		}
		q := dbgen.New(candidate)
		v, err := readSettings(ctx, q)
		if err != nil {
			return failure("corrupt", err)
		}
		if err = config.Validate(v); err != nil {
			return failure("invalid_values", err)
		}
		refs, err := readReferences(ctx, q)
		if err != nil {
			return failure("corrupt", err)
		}
		if _, err = readConnections(ctx, q, v, refs); err != nil {
			return failure("corrupt", err)
		}
		return nil
	}()
	if err != nil {
		return err
	}
	if err = syncFile(tempPath); err != nil {
		return failure("write_failed", err)
	}
	if err = source.Close(); err != nil {
		return failure("unreadable", err)
	}

	if s.db != nil {
		if err = s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	if s.lock == nil {
		lock, e := lockDatabase(s.path + ".lock")
		if e != nil {
			return failure("locked", e)
		}
		s.lock = lock
	}
	if err = s.archiveDatabase(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, s.path); err != nil {
		return failure("write_failed", err)
	}
	if err = s.open(ctx); err != nil {
		return err
	}
	s.uncertain = false
	if err := s.loadReferences(ctx); err != nil {
		return err
	}
	cfg, err := readSettings(ctx, dbgen.New(s.db))
	if err != nil {
		return err
	}
	state, err := readConnections(ctx, dbgen.New(s.db), cfg, s.refs)
	if err != nil {
		return err
	}
	s.connections = state
	s.pendingConnections = nil
	s.pending = nil
	return nil
}
