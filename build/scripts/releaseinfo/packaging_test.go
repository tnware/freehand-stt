package main

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMacPackagingContract(t *testing.T) {
	data, err := os.ReadFile("../../darwin/Taskfile.yml")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Tasks map[string]struct {
			Env       map[string]string `yaml:"env"`
			Cmds      []any             `yaml:"cmds"`
			Deps      []any             `yaml:"deps"`
			Platforms []string          `yaml:"platforms"`
		} `yaml:"tasks"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	native := file.Tasks["build:native"]
	for key, want := range map[string]string{"MACOSX_DEPLOYMENT_TARGET": "13.0", "CGO_CFLAGS": "-mmacosx-version-min=13.0", "CGO_LDFLAGS": "-mmacosx-version-min=13.0"} {
		if native.Env[key] != want {
			t.Errorf("native %s=%q want %q", key, native.Env[key], want)
		}
	}
	encode := func(v any) string {
		b, err := yaml.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if !strings.Contains(encode(file.Tasks["create:app:bundle"]), "common:sync:release-info") {
		t.Error("bundle must synchronize its release metadata")
	}
	if !strings.Contains(encode(file.Tasks["package"].Cmds), "create:archive") {
		t.Error("package must produce updater archive")
	}
	archive := encode(file.Tasks["create:archive"])
	for _, want := range []string{"ditto", "--keepParent", "freehand-darwin-", ".zip", "codesign --verify", "lipo", "-verify_arch"} {
		if !strings.Contains(archive, want) {
			t.Errorf("archive lacks %s", want)
		}
	}
	if !strings.Contains(encode(file.Tasks["run"]), "common:sync:release-info") {
		t.Error("development bundle must synchronize release metadata")
	}
}
