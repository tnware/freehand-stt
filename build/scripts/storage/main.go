// Command storage is the pinned generation and persistence-contract check entry point.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const sqlcVersion = "v1.31.1"
const generated = "internal/storage/dbgen"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	check := flag.Bool("check", false, "verify generated output without accepting drift")
	base := flag.String("base", "", "git revision whose published migrations must remain unchanged")
	flag.Parse()
	if err := checkBoundaries("."); err != nil {
		return err
	}
	if err := checkMigrations(*base); err != nil {
		return err
	}
	before, err := snapshot(generated)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "run", "github.com/sqlc-dev/sqlc/cmd/sqlc@"+sqlcVersion, "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		return err
	}
	if *check {
		after, err := snapshot(generated)
		if err != nil {
			return err
		}
		if len(before) != len(after) {
			return errors.New("generated query file set changed; regenerate and commit all output")
		}
		for path, content := range before {
			if !bytes.Equal(content, after[path]) {
				return fmt.Errorf("generated query drift: %s", path)
			}
		}
	}
	fmt.Println("Storage generation and ownership contract verified")
	return nil
}
func snapshot(dir string) (map[string][]byte, error) {
	result := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[path] = data
		return nil
	})
	return result, err
}
func checkBoundaries(root string) error {
	return filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		inStorage := strings.HasPrefix(rel, "internal/storage/")
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			name, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			dbImport := name == "database/sql" || strings.HasPrefix(name, "database/sql/") || strings.HasPrefix(name, "modernc.org/sqlite") || strings.HasPrefix(name, "github.com/pressly/goose/") || strings.HasSuffix(name, "/internal/storage/dbgen")
			if dbImport && !inStorage {
				return fmt.Errorf("database import outside storage: %s", rel)
			}
			for _, forbidden := range []string{"gorm.io/", "github.com/jmoiron/sqlx", "github.com/Masterminds/squirrel", "github.com/golang-migrate/", "github.com/mattn/go-sqlite3", "github.com/ncruces/go-sqlite3"} {
				if strings.HasPrefix(name, forbidden) {
					return fmt.Errorf("alternative persistence dependency in %s", rel)
				}
			}
		}
		infrastructure := rel == "internal/storage/store.go" || rel == "internal/storage/recovery.go" || strings.HasPrefix(rel, generated+"/")
		var violation error
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "Exec", "Query", "QueryRow", "Prepare":
				if inStorage && !infrastructure {
					violation = fmt.Errorf("handwritten SQL outside infrastructure/generated queries: %s", rel)
				}
			case "ExecContext", "QueryContext", "QueryRowContext", "PrepareContext":
				if !infrastructure {
					violation = fmt.Errorf("handwritten SQL outside infrastructure/generated queries: %s", rel)
				}
			}
			return true
		})
		return violation
	})
}
func checkMigrations(base string) error {
	dir := "internal/storage/migrations"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	valid := regexp.MustCompile(`^[0-9]{5}_[a-z0-9_]+\.sql$`)
	versions := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !valid.MatchString(entry.Name()) {
			return fmt.Errorf("invalid migration filename: %s", entry.Name())
		}
		version := entry.Name()[:5]
		if version == "00000" || versions[version] {
			return errors.New("invalid or duplicate migration version")
		}
		versions[version] = true
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("NO TRANSACTION")) || !bytes.Contains(data, []byte("-- +goose Up")) {
			return fmt.Errorf("migration must use goose transactions: %s", entry.Name())
		}
	}
	if base == "" {
		return nil
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`).MatchString(base) || strings.HasPrefix(base, "-") {
		return errors.New("invalid migration baseline")
	}
	paths, err := exec.Command("git", "ls-tree", "-r", "--name-only", base, "--", dir).Output()
	if err != nil {
		return err
	}
	for _, path := range strings.Fields(string(paths)) {
		old, err := exec.Command("git", "show", base+":"+path).Output()
		if err != nil {
			return err
		}
		current, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("published migration missing: %s", path)
		}
		if !bytes.Equal(old, current) {
			return fmt.Errorf("published migration changed: %s", path)
		}
	}
	return nil
}
