package cicontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePleaseGraduatesToStableWithoutPinningFutureVersions(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "release-please-config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config["prerelease"] != false {
		t.Fatal("Release Please must explicitly create non-prereleases")
	}
	// This strategy graduates the existing alpha without bumping its base;
	// with prerelease=false, subsequent stable versions use ordinary SemVer.
	if config["versioning"] != "prerelease" || config["initial-version"] != "0.1.0" {
		t.Fatal("missing stable graduation policy")
	}
	for _, name := range []string{"release-as", "prerelease-type"} {
		if _, ok := config[name]; ok {
			t.Errorf("%s must not pin a version or alpha channel", name)
		}
	}
	if config["draft"] != true || config["force-tag-creation"] != true {
		t.Fatal("stable releases must retain draft/tag validation staging")
	}
}

func TestStablePublicationVerifiesReleaseFlagsAndLatest(t *testing.T) {
	var commands string
	for _, step := range load(t, "release").Jobs["publish"].Steps {
		commands += step.Run + "\n"
	}
	for _, required := range []string{
		`[[ "$RELEASE_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]`,
		`--draft=false --prerelease=false --latest=true`,
	} {
		if !strings.Contains(commands, required) {
			t.Errorf("missing stable publication guard: %s", required)
		}
	}
	publication := strings.Index(commands, "gh release edit")
	for _, readback := range []string{
		`test "$(gh release view "$RELEASE_TAG" --json isDraft,isPrerelease --jq '[.isDraft,.isPrerelease] | @json')" = '[false,false]'`,
		`test "$(gh api "repos/$GH_REPO/releases/latest" --jq .tag_name)" = "$RELEASE_TAG"`,
	} {
		if publication < 0 || strings.Index(commands, readback) <= publication {
			t.Errorf("missing post-publication equality check: %s", readback)
		}
	}
	guard := strings.Index(commands, `[[ "$RELEASE_TAG"`)
	upload := strings.Index(commands, "gh release upload")
	if guard < 0 || upload <= guard {
		t.Fatal("reject non-stable tags before uploading release assets")
	}
}
