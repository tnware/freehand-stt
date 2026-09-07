package storage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/tnware/freehand-stt/internal/diagnostics"
)

func TestStorageDiagnosticKindsSurviveWrapping(t *testing.T) {
	for _, kind := range []string{"locked", "newer_schema", "foreign_database", "corrupt", "legacy_invalid", "legacy_newer", "write_failed", "backup_failed", "migration_failed", "unreadable", "invalid_values", "commit_uncertain"} {
		t.Run(kind, func(t *testing.T) {
			err := fmt.Errorf("settings were not persisted: %w", failure(kind, errors.New("private driver text and database path")))
			if got := diagnostics.ErrorKind(err); got != kind {
				t.Fatalf("ErrorKind = %q, want %q", got, kind)
			}
		})
	}
	if got := diagnostics.ErrorKind(failure("private unrecognized category", errors.New("private driver text"))); got != "operation" {
		t.Fatalf("unrecognized classification escaped: %q", got)
	}
}
