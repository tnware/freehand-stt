<script lang="ts">
  import { Service as SettingsService } from "$bindings/settings";
  import type { VocabularyPreview, VocabularyPreviewRequest } from "$bindings/config";
  import type { Settings } from "$lib/state";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Textarea } from "$lib/components/ui/textarea";
  import { Switch } from "$lib/components/ui/switch";
  import SettingRow from "../SettingRow.svelte";
  import SettingsDisclosure from "../SettingsDisclosure.svelte";
  import FieldHelp from "../FieldHelp.svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import { Button } from "$lib/components/ui/button";
  import { cn } from "$lib/utils";

  let {
    settings,
    onChange,
    disabled = false,
    previewVocabulary = SettingsService.PreviewVocabulary,
  }: {
    settings: Settings;
    onChange: (patch: Partial<Settings["vocabulary"]>) => void;
    disabled?: boolean;
    previewVocabulary?: typeof SettingsService.PreviewVocabulary;
  } = $props();
  let preview = $state<VocabularyPreview | null>(null);
  let error = $state("");
  let checking = $state(true);
  let retry = $state(0);
  let input = $state<HTMLTextAreaElement | null>(null);
  function selectLine(line: number) {
    if (!input) return;
    const lines = input.value.split("\n");
    const start = lines.slice(0, line - 1).reduce((sum, value) => sum + value.length + 1, 0);
    input.focus();
    input.setSelectionRange(start, start + (lines[line - 1]?.length ?? 0));
    const lineHeight = Number.parseFloat(getComputedStyle(input).lineHeight) || 20;
    input.scrollTop = Math.max(0, (line - 2) * lineHeight);
  }
  const bytes = $derived(new TextEncoder().encode(settings.vocabulary.terms).length);
  const request = $derived<VocabularyPreviewRequest>({
    vocabulary: { ...settings.vocabulary },
    voice: {
      backend: settings.voiceTranscription.compatibilityProfile,
      modelProfile: settings.voiceTranscription.modelProfile,
      realtime: settings.voiceTranscription.realtime,
      context: settings.voiceTranscription.transcriptionOptions.prompt,
    },
    files: {
      backend: settings.compatibilityProfile,
      modelProfile: settings.modelProfile,
      realtime: false,
      context: settings.transcriptionOptions.prompt,
    },
  });
  $effect(() => {
    const value = request;
    retry;
    let stale = false;
    checking = true;
    error = "";
    const timer = setTimeout(() => {
      void previewVocabulary(value)
        .then((result) => {
          if (!stale) {
            preview = result;
            checking = false;
          }
        })
        .catch(() => {
          if (!stale) {
            preview = null;
            error = "Could not check vocabulary support.";
            checking = false;
          }
        });
    }, 150);
    return () => {
      stale = true;
      clearTimeout(timer);
    };
  });
  const modeLabels: Record<string, string> = {
    "speech-contexts": "Vocabulary boosting",
    hotwords: "Recognition hotwords",
    prompt: "Appended to the transcription context",
  };
  const lineProblems = $derived(
    preview?.issues?.some((issue) => issue.voiceProblem || issue.filesProblem),
  );
  const uses = $derived([
    {
      key: "voice" as const,
      label: "Voice transcription",
      backend: settings.voiceTranscription.compatibilityProfile,
      model: settings.voiceTranscription.model || "No model selected",
      support: preview?.voice,
    },
    {
      key: "files" as const,
      label: "Audio-file transcription",
      backend: settings.compatibilityProfile,
      model: settings.model || "No model selected",
      support: preview?.files,
    },
  ]);
</script>

<div class="overflow-hidden rounded-xl border border-hairline bg-layer-fill">
  <div class="space-y-3 p-4">
    <div class="flex items-center justify-between gap-3">
      <label for="vocabulary-terms" class="text-sm font-semibold">Names and phrases</label>
      <FieldHelp
        label="About vocabulary hints"
        text="Add one name or phrase per line, including spaces within a name. These are recognition hints, not guaranteed replacements or cleanup instructions. Your list is saved locally and sent only to the workflows you enable. Changing a connection or model keeps the list and your workflow choices. Cleanup and text to speech use their own instructions; they do not use this list."
      />
    </div>
    <p id="vocabulary-help" class="text-xs text-muted-foreground">One name or phrase per line.</p>
    <Textarea
      bind:ref={input}
      class="field-sizing-fixed h-44 min-h-24 resize-y font-mono text-sm"
      id="vocabulary-terms"
      bind:value={() => settings.vocabulary.terms, (terms) => onChange({ terms })}
      {disabled}
      rows={7}
      maxlength={16384}
      aria-describedby="vocabulary-help vocabulary-size"
      aria-invalid={bytes > 16384}
      placeholder="One name or phrase per line"
    />
    <div class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
      <span role="status">
        {#if checking}Checking vocabulary…
        {:else if preview && bytes <= 16384}{preview.phraseCount} unique {preview.phraseCount === 1
            ? "phrase"
            : "phrases"}
        {/if}
      </span>
      <span id="vocabulary-size" class={cn("tabular-nums", bytes > 16384 && "text-destructive")}
        >{bytes.toLocaleString()} / 16,384 bytes</span
      >
    </div>
    {#if bytes > 16384}
      <p class="text-xs text-destructive" role="alert">Shorten the list to 16,384 bytes to save.</p>
    {/if}
    {#if error}
      <div class="flex items-center justify-between gap-3 text-xs">
        <p role="alert" class="text-warning">{error}</p>
        <Button variant="outline" size="sm" disabled={disabled || checking} onclick={() => retry++}
          >Try again</Button
        >
      </div>
    {/if}
    {#if preview?.issues?.length}
      <details
        class="group/lines rounded-lg border border-hairline"
        aria-label="Vocabulary line feedback"
        aria-busy={checking}
      >
        <summary
          class="flex cursor-pointer list-none items-center gap-3 rounded-lg px-3 py-2.5 text-xs hover:bg-subtle-fill-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden"
        >
          <span
            class={cn(
              "min-w-0 flex-1",
              !checking && lineProblems ? "text-warning" : "text-muted-foreground",
            )}
          >
            {#if checking}Checking line feedback…{:else}{preview.issues.length}
              {preview.issues.length === 1 ? "line" : "lines"} to review
              {#if preview.duplicateCount}
                · {preview.duplicateCount}
                {preview.duplicateCount === 1 ? "duplicate" : "duplicates"}{/if}
            {/if}
          </span>
          <ChevronDownIcon
            class="size-4 shrink-0 text-muted-foreground group-open/lines:rotate-180"
          />
        </summary>
        <div class="border-t border-hairline">
          <p class="px-3 py-2 text-xs leading-relaxed text-muted-foreground">
            Select a line to edit it. Duplicates are sent once; your list stays unchanged.
          </p>
          <ul class="max-h-48 divide-y divide-hairline overflow-y-auto overscroll-contain">
            {#each preview.issues as issue (issue.line)}
              <li class="px-3 py-2 text-xs leading-relaxed">
                <button
                  type="button"
                  class="rounded-sm font-medium text-primary underline underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  disabled={disabled || checking}
                  onclick={() => selectLine(issue.line)}>Line {issue.line}</button
                >
                {#if issue.duplicateOf}<span class="ml-2 text-muted-foreground"
                    >Duplicate of line {issue.duplicateOf}</span
                  >{/if}
                {#if issue.voiceProblem}<p class="mt-1 text-warning">
                    Voice{settings.vocabulary.voice ? "" : " (off)"}: {issue.voiceProblem}
                  </p>{/if}
                {#if issue.filesProblem}<p class="mt-1 text-warning">
                    Audio file{settings.vocabulary.files ? "" : " (off)"}: {issue.filesProblem}
                  </p>{/if}
              </li>
            {/each}
          </ul>
        </div>
      </details>
    {/if}
  </div>

  <div class="border-t border-hairline" aria-label="Vocabulary workflows">
    <div class="px-4 pt-3 pb-1">
      <h3 class="text-sm font-semibold">Use vocabulary in</h3>
      <p class="mt-1 text-xs text-muted-foreground">Changes apply after saving.</p>
    </div>
    {#each uses as use (use.key)}
      <div class="flex items-center gap-3 px-4 py-3">
        <ProviderIcon profile={use.backend} />
        <div class="min-w-0 flex-1">
          <label for={`vocabulary-${use.key}`} class="text-sm font-medium">{use.label}</label>
          <p
            class={cn(
              "mt-0.5 text-xs",
              !checking && use.support?.problem && settings.vocabulary[use.key]
                ? "text-warning"
                : "text-muted-foreground",
            )}
            role="status"
          >
            {#if checking}Checking support…
            {:else if error}Support check unavailable
            {:else if use.support?.problem}{use.support.mode
                ? "Review vocabulary for this selection"
                : "Unavailable with this selection"}
            {:else if !settings.vocabulary[use.key]}Off
            {:else if !preview?.phraseCount}Add phrases above
            {:else}Hints enabled{/if}
          </p>
        </div>
        <FieldHelp
          label={`${use.label} vocabulary support`}
          text={`${use.model}\n\n${checking ? "Checking this selection…" : error || (use.support?.problem ? `${use.support.problem} ${use.support.mode ? "Shorten the list or turn vocabulary off for this workflow." : "Your preference is kept for when you select a supported model."}` : modeLabels[use.support?.mode ?? ""] || "Support is unavailable.")}`}
        />
        <Switch
          id={`vocabulary-${use.key}`}
          checked={settings.vocabulary[use.key]}
          {disabled}
          onCheckedChange={(value) => onChange({ [use.key]: value })}
        />
      </div>
    {/each}
  </div>
</div>

{#if preview?.voice.mode === "speech-contexts" || preview?.files.mode === "speech-contexts"}
  <SettingsDisclosure title="Vocabulary tuning" description="Recognition strength for Nemotron">
    <SettingRow
      title="Vocabulary strength"
      description="Shared by Voice and audio files using Nemotron. 3 is a starting point; stronger hints can increase incorrect matches. The server may cap this value."
    >
      {#snippet control()}<input
          aria-label="Vocabulary strength"
          type="number"
          min="0"
          max="5"
          step="0.5"
          bind:value={() => settings.vocabulary.boost, (boost) => onChange({ boost: boost ?? 0 })}
          {disabled}
          class="h-9 w-20 rounded-md border border-input bg-background px-3 text-sm"
        />{/snippet}
    </SettingRow>
    <p class="px-5 py-3 text-xs leading-relaxed text-muted-foreground">
      Nemotron accepts up to 32 phrases, 128 UTF-8 bytes per phrase, and 2,048 bytes total. Other
      models use their own supported hint format.
    </p>
  </SettingsDisclosure>
{/if}
