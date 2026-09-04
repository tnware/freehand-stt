package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/releaseinfo"
)

func TestVersionedJSONDerivesFixedAndDisplayVersions(t *testing.T) {
	result, err := versionedJSON([]byte(`{
  "fixed": {"file_version": "9.9.9.9"},
  "info": {"0000": {"ProductName": "Freehand", "ProductVersion": "old"}}
}`), releaseinfo.Info{Version: "0.1.0-alpha.2", WindowsVersion: "0.1.0.2"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(result)
	for _, want := range []string{
		`"file_version": "0.1.0.2"`,
		`"ProductVersion": "0.1.0-alpha.2"`,
		`"FileVersion": "0.1.0-alpha.2"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("version resource does not contain %s:\n%s", want, text)
		}
	}
}

func TestVersionedJSONPreservesWindowsLineEndings(t *testing.T) {
	source := strings.ReplaceAll(`{
	"fixed": {
		"file_version": "0.1.0.2"
	},
	"info": {
		"0000": {
			"FileVersion": "0.1.0-alpha.2",
			"ProductVersion": "0.1.0-alpha.2"
		}
	}
}
`, "\n", "\r\n")
	result, err := versionedJSON([]byte(source), releaseinfo.Info{
		Version:        "0.1.0-alpha.2",
		WindowsVersion: "0.1.0.2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result, []byte(source)) {
		t.Fatalf("CRLF version resource was rewritten:\n%q", result)
	}
}

func TestReplaceOneRefusesAmbiguousAssets(t *testing.T) {
	pattern := regexp.MustCompile(`version="[^"]+"`)
	if _, err := replaceOne([]byte(`<a version="1"/><b version="2"/>`), pattern, `version="3"`, "version"); err == nil {
		t.Fatal("ambiguous generated asset was rewritten")
	}
	result, err := replaceOne([]byte(`<a version="1"/>`), pattern, `version="3"`, "version")
	if err != nil || string(result) != `<a version="3"/>` {
		t.Fatalf("single replacement = %q, %v", result, err)
	}
}
