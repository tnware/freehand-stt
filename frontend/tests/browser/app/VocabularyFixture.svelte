<script lang="ts">
  import { CancellablePromise } from "@wailsio/runtime";
  import type { VocabularyPreview, VocabularyPreviewRequest } from "$bindings/config";
  import { ID as BackendID } from "$bindings/compatibility";
  import VocabularySection from "$lib/components/settings/sections/VocabularySection.svelte";
  import SettingsNav from "$lib/components/settings/SettingsNav.svelte";
  import { settings } from "$lib/stores/session-fixtures-data";
  import { Button } from "$lib/components/ui/button";

  const scenario = new URLSearchParams(location.search).get("scenario");
  let draft = $state(structuredClone(settings));
  draft.voiceTranscription.compatibilityProfile = BackendID.NeMoSpeechV1;
  draft.voiceTranscription.model = "nvidia/nemotron-3.5-asr-streaming-0.6b";
  draft.model = "Example audio-file model";
  draft.vocabulary = {
    terms: scenario === "review" ? "Freehand\nFreehand\nA phrase to review" : "Freehand\nOpenAI",
    voice: true,
    files: false,
    boost: 3,
  };
  let attempts = 0;
  let saved = $state(false);
  function previewVocabulary(
    request: VocabularyPreviewRequest,
  ): CancellablePromise<VocabularyPreview> {
    attempts++;
    const failed = scenario === "failure" && attempts === 1;
    const review = request.vocabulary.terms.includes("A phrase to review");
    const words = request.vocabulary.terms.split("\n").filter(Boolean);
    return new CancellablePromise((resolve, reject) => {
      setTimeout(
        () => {
          if (failed) {
            reject(new Error("Synthetic preview failure"));
            return;
          }
          resolve({
            phraseCount: new Set(words).size,
            duplicateCount: words.length - new Set(words).size,
            issues: review
              ? [
                  { line: 2, duplicateOf: 1, voiceProblem: "", filesProblem: "" },
                  {
                    line: 3,
                    duplicateOf: 0,
                    voiceProblem: "Exceeds the selected model's phrase limit",
                    filesProblem: "",
                  },
                ]
              : [],
            voice: {
              mode: "speech-contexts",
              problem: review ? "Shorten this phrase before using vocabulary with this model." : "",
            },
            files:
              scenario === "unsupported"
                ? { mode: "", problem: "This model and backend do not support vocabulary hints." }
                : { mode: "hotwords", problem: "" },
          });
        },
        request.vocabulary.terms === "Old draft" ? 700 : 25,
      );
    });
  }
</script>

<div class="flex h-screen bg-background text-foreground">
  <SettingsNav active="vocabulary" onSelect={() => {}} />
  <div class="flex min-w-0 flex-1 flex-col">
    <main class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-5 sm:px-6">
      <div class="@container flex w-full max-w-[760px] flex-col gap-4">
        <header class="sticky top-0 z-10 space-y-1.5 border-b border-hairline bg-background py-4">
          <h1 class="text-xl font-semibold tracking-tight">Vocabulary</h1>
          <p class="text-[13px] text-muted-foreground">
            Shared names and phrases for supported transcription models.
          </p>
        </header>
        <VocabularySection
          settings={draft}
          {previewVocabulary}
          onChange={(patch) => {
            draft.vocabulary = { ...draft.vocabulary, ...patch };
            saved = false;
          }}
        />
      </div>
    </main>
    <footer
      class="flex h-14 shrink-0 items-center justify-between gap-3 border-t border-hairline bg-layer-fill px-4"
    >
      <span class="text-xs text-muted-foreground" role="status"
        >{saved ? "All changes saved" : "Unsaved changes"}</span
      >
      <Button size="sm" onclick={() => (saved = true)}>Save settings</Button>
    </footer>
  </div>
</div>
