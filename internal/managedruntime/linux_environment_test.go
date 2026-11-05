package managedruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestLinuxRecipeScope(t *testing.T) {
	for _, provider := range []ProviderID{NeMoSpeechCPP, LlamaCPP} {
		recipe, ok := recipeFor(provider, "linux", "amd64", "cpu")
		if !ok || recipe.Executable == "" || len(recipe.Archives) != 1 || recipe.Archives[0].Size <= 0 || len(recipe.Archives[0].SHA256) != 64 {
			t.Fatalf("missing Linux pin: %v", provider)
		}
		for _, backend := range []string{"cuda", "vulkan", "metal"} {
			if _, ok := recipeFor(provider, "linux", "amd64", backend); ok {
				t.Fatal("unqualified Linux accelerator admitted")
			}
		}
		if _, ok := recipeFor(provider, "linux", "arm64", "cpu"); ok {
			t.Fatal("unqualified architecture admitted")
		}
	}
	if _, ok := recipeFor(WhisperCPP, "linux", "amd64", "cpu"); ok {
		t.Fatal("no official Linux whisper server archive exists at pin")
	}
}

func TestLlamaConfigAdmission(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.ini")
	if err := requireNoLlamaSystemConfig(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("log-file=private.log"), 0600); err != nil {
		t.Fatal(err)
	}
	if requireNoLlamaSystemConfig(p) == nil {
		t.Fatal("ambient config admitted")
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(t.TempDir(), "config.ini")
		if err := os.Symlink(filepath.Join(t.TempDir(), "absent"), link); err != nil {
			t.Fatal(err)
		}
		if requireNoLlamaSystemConfig(link) == nil {
			t.Fatal("dangling config link admitted")
		}
	}
}

func TestLinuxManagedEnvironment(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux environment policy")
	}
	root := t.TempDir()
	env, err := managedGGMLEnvironment(root, WhisperCPP, []string{"LD_PRELOAD=foreign", "HOME=foreign", "XDG_CONFIG_HOME=foreign", "XDG_CACHE_HOME=foreign", "LLAMA_ARG_LOG_FILE=foreign", "HTTPS_PROXY=foreign"})
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 3 {
		t.Fatal("ambient overrides inherited", env)
	}
	for _, v := range env {
		if strings.Contains(v, "foreign") {
			t.Fatal("ambient override inherited")
		}
	}
	if !slices.Contains(env, "HOME="+filepath.Join(root, "private-home")) {
		t.Fatal("missing private home")
	}
	nemo := childEnvironment(root, []string{"LD_PRELOAD=foreign", "PATH=foreign", "HF_TOKEN=foreign", "HTTPS_PROXY=foreign"})
	if len(nemo) != 5 || !slices.Contains(nemo, "PATH=/usr/bin:/bin") {
		t.Fatal("unsafe NeMo environment", nemo)
	}
}

func TestNeMoLinuxRejectsPrivateCurlConfig(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux environment policy")
	}
	root := t.TempDir()
	if err := prepareNeMoLinuxEnvironment(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "private-home", ".curlrc"), []byte("proxy = foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	if prepareNeMoLinuxEnvironment(root) == nil {
		t.Fatal("curl config admitted")
	}
}
