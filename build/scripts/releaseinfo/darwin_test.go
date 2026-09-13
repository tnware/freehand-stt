package main

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMacBuildNumberRejectsInvalidInput(t *testing.T) {
	base := "info:\n  productName: Freehand\n  productIdentifier: io.github.tnware.freehand\n  version: 0.1.0-alpha.5\n"
	for _, build := range []string{"", "0", "01", "1.2.3.4", "alpha.5", "10000", "-1"} {
		source := base + "macos:\n  buildNumber: \"" + build + "\"\n"
		if _, err := parseMacInfo([]byte(source)); err == nil {
			t.Errorf("accepted build number %q", build)
		}
	}
}
func TestCommittedMacMetadataSynchronized(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	source, err := os.ReadFile(filepath.Join(root, "build", "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := parseMacInfo(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, dev := range []bool{false, true} {
		name := "Info.plist"
		if dev {
			name = "Info.dev.plist"
		}
		actual, err := os.ReadFile(filepath.Join(root, "build", "darwin", name))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := macPlist(info, dev)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("%s is stale; run releaseinfo sync", name)
		}
	}
}
func TestMacPlistDerivesReleaseIdentity(t *testing.T) {
	source := []byte(`info:
  productName: "Freehand & Friends"
  productIdentifier: "io.github.tnware.freehand"
  version: "0.1.0-alpha.5"
  description: "Speech to text, anywhere you type."
  copyright: "Copyright 2026 Tyler Woods"
macos:
  buildNumber: "5"
`)
	info, err := parseMacInfo(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, dev := range []bool{false, true} {
		result, err := macPlist(info, dev)
		if err != nil {
			t.Fatal(err)
		}
		// Decode actual XML, not substring matching unescaped metadata.
		decoder := xml.NewDecoder(strings.NewReader(string(result)))
		values := map[string]string{}
		key := ""
		for {
			token, err := decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
			}
			if start, ok := token.(xml.StartElement); ok {
				if start.Name.Local == "key" {
					if err := decoder.DecodeElement(&key, &start); err != nil {
						t.Fatal(err)
					}
				}
				if start.Name.Local == "string" {
					var value string
					if err := decoder.DecodeElement(&value, &start); err != nil {
						t.Fatal(err)
					}
					values[key] = value
				}
			}
		}
		id := "io.github.tnware.freehand"
		if dev {
			id += ".dev"
		}
		for k, want := range map[string]string{"CFBundleDisplayName": "Freehand & Friends", "CFBundleName": "Freehand & Friends", "CFBundleIdentifier": id, "CFBundleExecutable": "freehand", "CFBundleShortVersionString": "0.1.0", "CFBundleVersion": "5", "FreehandReleaseVersion": "0.1.0-alpha.5", "LSMinimumSystemVersion": "13.0"} {
			if got := values[k]; got != want {
				t.Errorf("%s=%q, want %q", k, got, want)
			}
		}
		if values["NSMicrophoneUsageDescription"] == "" {
			t.Fatal("missing microphone purpose")
		}
		if dev != strings.Contains(string(result), "NSAllowsLocalNetworking") {
			t.Fatal("development network allowance leaked or missing")
		}
	}
}
