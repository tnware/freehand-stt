package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSQLBoundaryRejectsBypasses(t *testing.T) {
	for _, tc := range []struct {
		path, source string
		allowed      bool
	}{
		{"internal/storage/settings.go", `package storage; import "database/sql"; var _ = sql.ErrNoRows`, true},
		{"internal/storage/dbgen/query.go", `package dbgen; func f(q interface{ExecContext()}){q.ExecContext()}`, true},
		{"internal/settings/bypass.go", `package settings; import "database/sql"; var _ = sql.ErrNoRows`, false},
		{"internal/storage/bypass.go", `package storage; func f(q interface{ExecContext()}){q.ExecContext()}`, false},
		{"internal/storage/orm.go", `package storage; import _ "gorm.io/gorm"`, false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, tc.path)
			os.MkdirAll(filepath.Dir(path), 0700)
			os.WriteFile(path, []byte(tc.source), 0600)
			if err := checkBoundaries(root); (err == nil) != tc.allowed {
				t.Fatalf("boundary result: %v", err)
			}
		})
	}
}
