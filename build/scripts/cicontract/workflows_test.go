package cicontract

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type job struct {
	RunsOn string            `yaml:"runs-on"`
	Env    map[string]string `yaml:"env"`
	Name   string            `yaml:"name"`
	If     string            `yaml:"if"`
	Needs  yaml.Node         `yaml:"needs"`
	Uses   string            `yaml:"uses"`
	With   map[string]any    `yaml:"with"`
	Steps  []struct {
		Run  string         `yaml:"run"`
		Uses string         `yaml:"uses"`
		With map[string]any `yaml:"with"`
	} `yaml:"steps"`
}

type workflow struct {
	On   map[string]map[string]any `yaml:"on"`
	Jobs map[string]job            `yaml:"jobs"`
}

func load(t *testing.T, name string) workflow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".github", "workflows", name+".yml"))
	if err != nil {
		t.Fatal(err)
	}
	var result workflow
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestValidationAlwaysReportsAndIncludesSelectedWorkloads(t *testing.T) {
	ci := load(t, "ci")
	for _, event := range []string{"pull_request", "push", "workflow_dispatch", "workflow_call"} {
		config, ok := ci.On[event]
		if !ok {
			t.Errorf("missing %s trigger", event)
		}
		if config["paths"] != nil || config["paths-ignore"] != nil {
			t.Errorf("%s can skip required check", event)
		}
	}
	gate := ci.Jobs["validation"]
	if gate.Name != "Validation" || gate.If != "always()" {
		t.Fatal("missing unconditional stable Validation gate")
	}
	var needs []string
	if err := gate.Needs.Decode(&needs); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"changes", "storage", "go", "frontend", "windows", "macos", "site"} {
		if !slices.Contains(needs, name) {
			t.Errorf("gate does not depend on %s", name)
		}
	}
}

func TestReleasePublishesOnlyArtifactsFromItsOwnValidatedTag(t *testing.T) {
	release := load(t, "release")
	validation := release.Jobs["validation"]
	if validation.Uses != "./.github/workflows/ci.yml" {
		t.Fatal("release does not reuse complete validation")
	}
	if validation.With["ref"] != "${{ needs.release-please.outputs.tag-name }}" {
		t.Fatal("validation does not check release tag")
	}
	var needs []string
	publication := release.Jobs["publish"]
	if err := publication.Needs.Decode(&needs); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(needs, "validation") {
		t.Fatal("publication can bypass validation")
	}
	found := false
	for _, step := range release.Jobs["publish"].Steps {
		if strings.HasPrefix(step.Uses, "actions/download-artifact@") {
			found = true
			if step.With["run-id"] != nil || step.With["repository"] != nil {
				t.Fatal("publication must not fetch cross-run artifacts")
			}
		}
		if strings.Contains(step.Run, "task package") {
			t.Fatal("release rebuilds instead of publishing its validated artifacts")
		}
	}
	if !found {
		t.Fatal("release does not consume validated artifacts")
	}
}

func TestMacOSNativeValidationAndBothPackages(t *testing.T) {
	mac := load(t, "ci").Jobs["macos"]
	if mac.RunsOn != "macos-15" || mac.If != "needs.changes.outputs.app == 'true'" {
		t.Fatal("macOS must run as a selected native workload")
	}
	for key, value := range map[string]string{"CGO_ENABLED": "1", "MACOSX_DEPLOYMENT_TARGET": "13.0", "CGO_CFLAGS": "-mmacosx-version-min=13.0", "CGO_LDFLAGS": "-mmacosx-version-min=13.0", "GOFLAGS": "-ldflags=-extldflags=-mmacosx-version-min=13.0"} {
		if mac.Env[key] != value {
			t.Errorf("missing macOS toolchain setting %s", key)
		}
	}
	var commands string
	var cache, upload bool
	for _, step := range mac.Steps {
		commands += step.Run + "\n"
		if step.Uses == "./.github/actions/setup-go" {
			cache = step.With["scope"] == "macos"
		}
		if strings.HasPrefix(step.Uses, "actions/checkout@") && step.With["ref"] != "${{ needs.changes.outputs.ref }}" {
			t.Fatal("macOS source is not pinned")
		}
		if strings.HasPrefix(step.Uses, "actions/upload-artifact@") {
			upload = step.With["name"] == "freehand-darwin" && step.With["if-no-files-found"] == "error"
			for _, arch := range []string{"arm64", "amd64"} {
				if !strings.Contains(step.With["path"].(string), "bin/freehand-darwin-"+arch+".zip") {
					t.Errorf("missing %s archive", arch)
				}
			}
		}
	}
	if !cache || !upload {
		t.Fatal("missing scoped cache or required artifact upload")
	}
	for _, command := range []string{"go test -race ./...", "go vet ./...", "go list -m", "wails3 task package ARCH=arm64 CI=true", "wails3 task package ARCH=amd64 CI=true", "go run ./build/scripts/releaseinfo -root . check"} {
		if !strings.Contains(commands, command) {
			t.Errorf("missing %s", command)
		}
	}
}

func TestReleaseAssemblesAndAttestsCompleteAssetsBeforePublication(t *testing.T) {
	publication := load(t, "release").Jobs["publish"]
	downloads := map[string]bool{}
	var commands, subjects string
	assembly, attestation, publish := -1, -1, -1
	for i, step := range publication.Steps {
		commands += step.Run + "\n"
		if strings.HasPrefix(step.Uses, "actions/download-artifact@") {
			downloads[step.With["name"].(string)] = true
		}
		if strings.Contains(step.Run, "node build/scripts/releaseassembly/assemble.mjs bin dist") {
			assembly = i
		}
		if strings.HasPrefix(step.Uses, "actions/attest@") {
			attestation = i
			subjects = step.With["subject-path"].(string)
		}
		if strings.Contains(step.Run, "gh release edit") {
			publish = i
		}
	}
	if len(downloads) != 2 || !downloads["freehand-windows-amd64"] || !downloads["freehand-darwin"] {
		t.Fatal("release must consume both validated platforms")
	}
	if assembly < 0 || attestation <= assembly || publish <= attestation {
		t.Fatal("publication must follow complete assembly and attestations")
	}
	for _, asset := range []string{"freehand-windows-amd64.exe", "freehand-windows-amd64-installer.exe", "freehand-darwin-arm64.zip", "freehand-darwin-amd64.zip", "SHA256SUMS"} {
		if !slices.Contains(strings.Fields(subjects), "dist/"+asset) {
			t.Errorf("missing attestation: %s", asset)
		}
	}
	guard := `gh release view "$RELEASE_TAG" --json assets | node build/scripts/releaseassembly/check.mjs`
	upload := strings.Index(commands, "gh release upload")
	check := strings.Index(commands, guard)
	readback := strings.Index(commands, "gh release download")
	edit := strings.Index(commands, "gh release edit")
	draft := strings.Index(commands, `test "$(gh release view "$RELEASE_TAG" --json isDraft --jq .isDraft)" = true`)
	if draft < 0 || upload <= draft || check <= upload || readback <= check || edit <= readback {
		t.Fatal("exact remote asset validation and checksum readback must occur while draft, before publication")
	}
	if strings.Contains(commands, "gh release delete-asset") {
		t.Fatal("unexpected remote assets must block publication, not be deleted")
	}
	for _, command := range []string{"sha256sum --check --strict SHA256SUMS", "gh release download", "cmp dist/SHA256SUMS uploaded/SHA256SUMS", "--draft=false --prerelease=false --latest=true"} {
		if !strings.Contains(commands, command) {
			t.Errorf("missing publication guard: %s", command)
		}
	}
}

func TestPagesPRValidationCannotCancelDeployment(t *testing.T) {
	pages := load(t, "pages")
	if _, ok := pages.On["pull_request"]; ok {
		t.Fatal("PR validation belongs to CI, not the deployment workflow")
	}
	if pages.Jobs["build"].Uses != "./.github/workflows/site.yml" {
		t.Fatal("deployment must reuse site validation")
	}
	if load(t, "ci").Jobs["site"].Uses != "./.github/workflows/site.yml" {
		t.Fatal("site validation is missing from CI")
	}
}

func TestPackagingUsesFrozenDependenciesInCI(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "build", "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `npm {{if eq .CI "true"}}ci{{else}}install{{end}}`) {
		t.Fatal("packaging does not use npm ci in CI")
	}
}
