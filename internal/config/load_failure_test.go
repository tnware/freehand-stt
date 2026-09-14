package config

import (
	"errors"
	"fmt"
	"testing"
)

type databaseFailureFixture struct{}

func (databaseFailureFixture) Error() string { return "private driver diagnostic" }
func (databaseFailureFixture) ConfigurationFailure() LoadFailure {
	return LoadFailure{Kind: "database_corrupt", Message: "The settings database is damaged."}
}

func TestLoadFailurePreservesWrappedDatabaseClassification(t *testing.T) {
	err := fmt.Errorf("load: %w", databaseFailureFixture{})
	if got, want := LoadFailureFor(err), (databaseFailureFixture{}).ConfigurationFailure(); got != want {
		t.Fatalf("database recovery classification = %#v; want %#v", got, want)
	}
}

func TestLoadFailureMasksUnclassifiedDiagnostics(t *testing.T) {
	got := LoadFailureFor(errors.New("private path and driver diagnostic"))
	want := LoadFailure{Kind: "unavailable", Message: "The saved configuration could not be loaded."}
	if got != want {
		t.Fatalf("unclassified failure leaked diagnostics: %#v", got)
	}
}
