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
	Name  string         `yaml:"name"`
	If    string         `yaml:"if"`
	Needs yaml.Node      `yaml:"needs"`
	Uses  string         `yaml:"uses"`
	With  map[string]any `yaml:"with"`
	Steps []struct {
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
	for _, name := range []string{"changes", "storage", "go", "frontend", "windows", "site"} {
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
	publication := release.Jobs["windows"]
	if err := publication.Needs.Decode(&needs); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(needs, "validation") {
		t.Fatal("publication can bypass validation")
	}
	found := false
	for _, step := range release.Jobs["windows"].Steps {
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
