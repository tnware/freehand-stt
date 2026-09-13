package main

import (
	"fmt"
	"testing"
)

func TestMacBuildVersionRejectsNoncanonicalPrerelease(t *testing.T) {
	for _, version := range []string{"0.1.0-alpha.01", "0.1.0-beta.001", "0.1.0-rc.0001"} {
		t.Run(version, func(t *testing.T) {
			info, err := parseMacInfo([]byte(fmt.Sprintf("info:\n  productName: Freehand\n  productIdentifier: io.github.tnware.freehand\n  version: %s\n", version)))
			if err == nil {
				t.Fatalf("invalid source %q normalized to CFBundleVersion %q", version, info.BuildNumber)
			}
		})
	}
}

func TestMacBuildVersionFollowsReleasePleaseVersion(t *testing.T) {
	for _, tt := range []struct{ version, want string }{
		{"0.1.0-alpha.5", "0.1.5"},
		{"0.1.0-alpha.1+build.001", "0.1.1"},
		{"0.1.0-alpha.6", "0.1.6"},
		{"0.1.0-beta.1", "0.1.65537"},
		{"0.1.0-rc.1", "0.1.131073"},
		{"0.1.0", "0.1.196608"},
		{"0.1.1", "0.1.458752"},
		{"1.0.0", "1.0.196608"},
	} {
		t.Run(tt.version, func(t *testing.T) {
			info, err := parseMacInfo([]byte(fmt.Sprintf("info:\n  productName: Freehand\n  productIdentifier: io.github.tnware.freehand\n  version: %s\n", tt.version)))
			if err != nil {
				t.Fatal(err)
			}
			if info.BuildNumber != tt.want {
				t.Fatalf("build = %q, want %q", info.BuildNumber, tt.want)
			}
		})
	}
}
