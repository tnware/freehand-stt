package compatibility

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishedCatalogMatchesImplementedContracts(t *testing.T) {
	actual, err := os.ReadFile(filepath.Join("..", "..", "site", "src", "data", "compatibility.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := json.MarshalIndent(Profiles(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// Git may use CRLF on Windows; compare generated content independently of checkout line endings.
	actual = bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(actual, append(expected, '\n')) {
		t.Fatal("website compatibility catalog is stale; run go generate ./internal/compatibility")
	}
}
