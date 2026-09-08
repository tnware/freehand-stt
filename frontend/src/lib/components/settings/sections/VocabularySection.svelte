<script lang="ts">
  import { Service as SettingsService } from "$bindings/settings";
  import type { VocabularyPreview, VocabularyPreviewRequest } from "$bindings/config";
  import type { Settings } from "$lib/state";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Textarea } from "$lib/components/ui/textarea";
  import { Switch } from "$lib/components/ui/switch";
  import SettingRow from "../SettingRow.svelte";

  let {
    settings,
    onChange,
    disabled = false,
  }: {
    settings: Settings;
    onChange: (patch: Partial<Settings["vocabulary"]>) => void;
    disabled?: boolean;
  } = $props();
  let preview = $state<VocabularyPreview | null>(null);
  let error = $state("");
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
    let stale = false;
    preview = null;
    error = "";
    const timer = setTimeout(() => {
      void SettingsService.PreviewVocabulary(value)
        .then((result) => {
          if (!stale) preview = result;
        })
        .catch(() => {
          if (!stale) error = "Could not check vocabulary support. Reopen this page to try again.";
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

<div class="space-y-3 rounded-xl border border-hairline bg-layer-fill p-5">
  <label for="vocabulary-terms" class="text-sm font-semibold">Names and phrases</label>
  <p id="vocabulary-help" class="text-xs leading-relaxed text-muted-foreground">
    Keep one phrase per line, including spaces within a name. These are recognition hints, not
    guaranteed replacements or cleanup instructions.
  </p>
  <Textarea
    id="vocabulary-terms"
    bind:value={() => settings.vocabulary.terms, (terms) => onChange({ terms })}
    {disabled}
    rows={8}
    maxlength={16384}
    aria-describedby="vocabulary-help vocabulary-size"
    aria-invalid={bytes > 16384}
    placeholder="One name or phrase per line"
  />
  <div class="flex flex-wrap justify-between gap-2 text-xs text-muted-foreground">
    <p>Saved locally. Sent only to the workflows you enable below.</p>
    <p id="vocabulary-size">{bytes.toLocaleString()} / 16,384 bytes</p>
  </div>
</div>

<div class="overflow-hidden rounded-xl border border-hairline bg-layer-fill">
  {#each uses as use (use.key)}
    <div class="border-b border-hairline p-5 last:border-b-0">
      <div class="flex items-center gap-3">
        <ProviderIcon profile={use.backend} />
        <div class="min-w-0 flex-1">
          <label for={`vocabulary-${use.key}`} class="text-sm font-medium">{use.label}</label>
          <p class="truncate text-xs text-muted-foreground" title={use.model}>{use.model}</p>
        </div>
        <Switch
          id={`vocabulary-${use.key}`}
          checked={settings.vocabulary[use.key]}
          {disabled}
          onCheckedChange={(value) => onChange({ [use.key]: value })}
        />
      </div>
      <p class="mt-3 text-xs leading-relaxed text-muted-foreground" role="status">
        {#if use.support?.problem}{use.support.problem}
          {use.support.mode
            ? "Shorten the list before using it with this selection, or turn it off here."
            : "Your preference is kept for when you select a supported model."}
        {:else if use.support}{modeLabels[use.support.mode]}. {settings.vocabulary[use.key]
            ? "Used on the next transcription after saving."
            : "Turn on to use the shared list."}
        {:else}{error || "Checking this selection…"}{/if}
      </p>
    </div>
  {/each}
</div>

{#if preview?.voice.mode === "speech-contexts" || preview?.files.mode === "speech-contexts"}
  <div class="rounded-xl border border-hairline bg-layer-fill">
    <SettingRow
      title="Vocabulary strength"
      description="Nemotron boosting, shared by Voice and audio files. 3 is a starting point; stronger hints can increase incorrect matches. The server may cap this value."
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
    <p class="px-5 pb-4 text-xs text-muted-foreground">
      Nemotron accepts up to 32 phrases, 128 UTF-8 bytes per phrase, and 2,048 bytes total. Other
      models use their own supported hint format.
    </p>
  </div>
{/if}

<p class="text-xs leading-relaxed text-muted-foreground">
  Changing a connection or model keeps this list and your workflow choices. Cleanup and text to
  speech do not have a qualified vocabulary control; their instructions and pronunciation behavior
  stay separate.
</p>
