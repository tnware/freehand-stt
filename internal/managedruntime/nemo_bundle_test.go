package managedruntime

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

// Replace only artifact bytes and the child boundary; catalog, verification,
// argument construction, process ownership and HTTP readiness stay real.
func managedSpeechBundleFixture(t *testing.T, root string) (*nemoAdapter, *[]string) {
	t.Helper()
	a, calls := managedAdapterFixture(t, root)
	originalModel, originalCodec, originalArchive := modelSpecs["magpie-tts"], nemoCodecSpec, nemoTokenizerArchive
	originalMembers := slices.Clone(nemoTokenizerMembers)
	t.Cleanup(func() {
		modelSpecs["magpie-tts"], nemoCodecSpec, nemoTokenizerArchive = originalModel, originalCodec, originalArchive
		nemoTokenizerMembers = originalMembers
	})
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte("fixture")))
	model := originalModel
	model.size, model.sha256 = 7, digest
	modelSpecs["magpie-tts"] = model
	nemoCodecSpec.size, nemoCodecSpec.sha256 = 7, digest
	nemoTokenizerArchive.size, nemoTokenizerArchive.sha256 = 7, digest
	for n := range nemoTokenizerMembers {
		nemoTokenizerMembers[n].size, nemoTokenizerMembers[n].sha256 = 7, digest
	}
	originalLaunch := a.launch
	a.launch = func(ctx context.Context, exe string, args []string, dir string, env []string) (*ownedProcess, error) {
		if strings.Join(args, " ") != "--json model pull "+model.repo {
			return originalLaunch(ctx, exe, args, dir, env)
		}
		*calls = append(*calls, strings.Join(args, " "))
		for _, spec := range []modelSpec{model, nemoCodecSpec} {
			if err := os.MkdirAll(spec.directory(root), 0700); err != nil {
				return nil, err
			}
			if err := os.WriteFile(spec.path(root), []byte("fixture"), 0600); err != nil {
				return nil, err
			}
		}
		if err := os.MkdirAll(nemoTokenizerDirectory(root), 0700); err != nil {
			return nil, err
		}
		for _, member := range nemoTokenizerMembers {
			if err := os.WriteFile(filepath.Join(nemoTokenizerDirectory(root), member.name), []byte("fixture"), 0600); err != nil {
				return nil, err
			}
		}
		p := &ownedProcess{done: make(chan struct{}), closeJob: func() {}}
		close(p.done)
		return p, nil
	}
	return a, calls
}

func TestNeMoSpeechBundleAcquisitionVerificationAndRemoval(t *testing.T) {
	root := t.TempDir()
	a, calls := managedSpeechBundleFixture(t, root)
	if err := a.Pull(t.Context(), "nemotron-3.5", nil); err != nil {
		t.Fatal(err)
	}
	if err := a.Pull(t.Context(), "magpie-tts", nil); err != nil {
		t.Fatal(err)
	}
	if err := verifyNeMoModel(t.Context(), root, "magpie-tts"); err != nil {
		t.Fatal(err)
	}
	progress := nemoModelAcquiredBytes(root, "magpie-tts")
	if progress.Bytes != 21 || progress.TotalBytes != 21 {
		t.Fatalf("bundle bytes: %+v", progress)
	}
	if !slices.Contains(*calls, "--json model pull "+modelSpecs["magpie-tts"].repo) {
		t.Fatal("bundle did not use the qualified model manager")
	}
	for _, target := range []string{nemoCodecSpec.path(root), filepath.Join(nemoTokenizerDirectory(root), nemoTokenizerMembers[0].name)} {
		if err := os.WriteFile(target, []byte("corrupt"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := verifyNeMoModel(t.Context(), root, "magpie-tts"); !errors.Is(err, errIntegrity) {
			t.Fatalf("accepted corrupt companion: %v", err)
		}
		if err := os.WriteFile(target, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	extra := filepath.Join(nemoTokenizerDirectory(root), "unqualified.dict")
	if err := os.WriteFile(extra, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyNeMoModel(t.Context(), root, "magpie-tts"); !errors.Is(err, errIntegrity) {
		t.Fatalf("accepted extra tokenizer input: %v", err)
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveModel(t.Context(), "magpie-tts"); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{modelSpecs["magpie-tts"].directory(root), nemoCodecSpec.directory(root)} {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("companion not removed: %v", err)
		}
	}
	if err := verifyNeMoModel(t.Context(), root, "nemotron-3.5"); err != nil {
		t.Fatal("removal affected ASR", err)
	}
}

func TestNeMoCombinedStartUsesVerifiedInputsAndOneProcess(t *testing.T) {
	root := t.TempDir()
	a, calls := managedSpeechBundleFixture(t, root)
	if err := a.Pull(t.Context(), "nemotron-3.5", nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.StartModels(t.Context(), "nemotron-3.5", "magpie-tts"); err == nil {
		t.Fatal("started without the complete speech bundle")
	}
	for _, call := range *calls {
		if strings.HasPrefix(call, "serve ") {
			t.Fatal("launched with missing speech assets")
		}
	}
	if err := a.Pull(t.Context(), "magpie-tts", nil); err != nil {
		t.Fatal(err)
	}
	owner := &providerProcess{}
	p, endpoint, err := owner.startModels(t.Context(), a, "nemotron-3.5", "magpie-tts")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.kill)
	if endpoint.Model != "actual GGUF name" || endpoint.SpeechModel != "actual Magpie name" {
		t.Fatalf("wrong role identities: %+v", endpoint)
	}
	if _, _, err := owner.start(t.Context(), a, "nemotron-3.5"); !errors.Is(err, errProviderRunning) {
		t.Fatalf("admitted second process: %v", err)
	}
	launches := 0
	for _, call := range *calls {
		if !strings.HasPrefix(call, "serve ") {
			continue
		}
		launches++
		for _, value := range []string{"--access-log --log-format json", "--tts-model " + modelSpecs["magpie-tts"].path(root), "--codec-model " + nemoCodecSpec.path(root), "--tokenizer-dir " + nemoTokenizerDirectory(root)} {
			if !strings.Contains(call, value) {
				t.Fatalf("missing launch option %q", value)
			}
		}
		if strings.Contains(call, "--json") {
			t.Fatal("global JSON would suppress startup status")
		}
	}
	if launches != 1 {
		t.Fatalf("launches=%d", launches)
	}
}

func TestNeMoCombinedReadinessRejectsUnexpectedEngines(t *testing.T) {
	valid := []nemoLoadedModel{{"asr", "transcription"}, {"tts", "speech"}}
	if asr, speech, err := nemoModelIdentities(valid, true); err != nil || asr != "asr" || speech != "tts" {
		t.Fatalf("valid combined identity: %s %s %v", asr, speech, err)
	}
	for _, rows := range [][]nemoLoadedModel{nil, valid[:1], {valid[0], valid[0]}, {valid[0], {"extra", "translation"}}, {valid[0], {"asr", "speech"}}, {valid[0], {"\n", "speech"}}, {valid[0], {strings.Repeat("x", 513), "speech"}}, append(slices.Clone(valid), nemoLoadedModel{"extra", "speech"})} {
		if _, _, err := nemoModelIdentities(rows, true); err == nil {
			t.Fatalf("accepted unexpected inventory: %+v", rows)
		}
	}
	if _, _, err := nemoModelIdentities(valid, false); err == nil {
		t.Fatal("accepted an unselected speech engine")
	}
}

func TestNeMoCatalogRequiresPinnedSpeechCompanions(t *testing.T) {
	data, err := os.ReadFile("testdata/catalog-v0.1.0.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, old := range []string{nemoCodecSpec.revision, "tokenizer", nemoCodecSpec.repo} {
		changed := strings.ReplaceAll(string(data), old, "unqualified")
		if _, err := parseCatalog([]byte(changed)); err == nil {
			t.Fatalf("accepted a changed speech companion field: %s", old)
		}
	}
}

func TestManagerCombinedRoleLeasesAndSelectionFence(t *testing.T) {
	root := t.TempDir()
	i := Instance{ID: LegacyInstanceID, Name: "Speech", Provider: NeMoSpeechCPP, Model: "nemotron-3.5", SpeechModel: "magpie-tts"}
	m := NewManager(ManagerOptions{Directory: root, Instances: []Instance{i}})
	a, _ := managedSpeechBundleFixture(t, instanceDirectory(root, i.ID))
	for _, model := range []string{i.Model, i.SpeechModel} {
		if err := a.Pull(t.Context(), model, nil); err != nil {
			t.Fatal(err)
		}
	}
	m.workers[i.ID].adapter = a
	m.workers[i.ID].status.Supported = true
	t.Cleanup(func() { _ = m.ServiceShutdown() })
	if err := m.startup(t.Context()); err != nil {
		t.Fatal(err)
	}
	waitInstance(t, m, i.ID, "installed")
	if err := m.Start(InstanceRequest{InstanceID: i.ID}); err != nil {
		t.Fatal(err)
	}
	waitInstance(t, m, i.ID, "running")
	asr, err := m.ResolveFor(i, compatibility.Transcription)
	if err != nil {
		t.Fatal(err)
	}
	speech, err := m.ResolveFor(i, compatibility.Speech)
	if err != nil {
		t.Fatal(err)
	}
	if asr.BaseURL != speech.BaseURL || asr.Generation != speech.Generation || speech.Model != "actual Magpie name" || speech.CatalogModel != "magpie-tts" || speech.Contract.Role != compatibility.Speech {
		t.Fatalf("incoherent role leases: asr=%+v speech=%+v", asr, speech)
	}
	changed := i
	changed.SpeechModel = ""
	if err := m.SetInstance(changed); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ResolveFor(i, compatibility.Transcription); err == nil {
		t.Fatal("old combined lease survived selection change")
	}
	if _, err := m.ResolveFor(changed, compatibility.Speech); err == nil {
		t.Fatal("disabled speech remained available")
	}
	if st := m.GetInstances()[0]; st.Status.State != "stopped" || st.ActiveSpeechModel != "" {
		t.Fatalf("selection did not stop and clear combined process: %+v", st)
	}
}
