package managedruntime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// The pinned v0.1.0 CUDA ZIP uses backslashes, including trailing
// backslashes for directories with no directory attribute. CPU uses slashes.
func TestOfficialWindowsArchiveLayout(t *testing.T) {
	b := zipFixture(t, map[string]string{`bin\nemo-speech.exe`: "binary", `lib\cmake\`: "", `lib\cmake\NeMoSpeech\config.cmake`: "config"})
	hash := sha256.Sum256(b)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(b) }))
	defer server.Close()
	a := asset{backend: "cuda", url: server.URL, size: int64(len(b)), sha256: fmt.Sprintf("%x", hash)}
	root := t.TempDir()
	if err := installAsset(context.Background(), root, a, server.Client(), nil); err != nil {
		t.Fatal(err)
	}
	if err := verifyRuntime(context.Background(), root, a); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(root, "runtime", "lib", "cmake")); err != nil || !st.IsDir() {
		t.Fatalf("directory: %v", err)
	}
}

func TestWindowsArchiveSeparatorsDoNotPermitTraversalOrAliases(t *testing.T) {
	for _, names := range []map[string]string{
		{`..\escape`: "bad"}, {`bin\..\escape`: "bad"}, {`\absolute`: "bad"}, {`\\server\share`: "bad"}, {`C:\outside`: "bad"},
		{`bin\CON`: "bad"}, {`bin\evil:stream`: "bad"}, {`bin\trailing. `: "bad"}, {`bin\\empty`: "bad"},
		{`bin\same`: "one", "bin/same": "two"}, {`bin\SAME`: "one", "bin/same": "two"},
	} {
		root := t.TempDir()
		archive := filepath.Join(root, "fixture.zip")
		if err := os.WriteFile(archive, zipFixture(t, names), 0600); err != nil {
			t.Fatal(err)
		}
		if err := extractArchive(context.Background(), archive, filepath.Join(root, "out")); err == nil {
			t.Fatalf("accepted unsafe entries %v", names)
		}
	}
}
