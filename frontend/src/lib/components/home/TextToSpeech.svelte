<script lang="ts">
  import type { SeekRequest } from "$bindings/tts";
  import type { Snippet } from "svelte";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
  import { TTSPhase, TTSSource, type Settings, type TTSStatus } from "$lib/state";
  import { cn } from "$lib/utils";

  const maximumCharacters = 4096;

  let {
    text = $bindable(""),
    quickSettings,
    settings,
    status,
    unavailable = false,
    submitting = false,
    seeking = false,
    saving = false,
    onSeek,
    onSpeak,
    onPause,
    onResume,
    onRestart,
    onStop,
    onSave,
    onClear,
    onOpenSettings,
  }: {
    text?: string;
    quickSettings?: Snippet;
    settings: Settings["textToSpeech"];
    status: TTSStatus;
    unavailable?: boolean;
    submitting?: boolean;
    seeking?: boolean;
    saving?: boolean;
    onSeek?: (request: SeekRequest) => Promise<void>;
    onSpeak: (text: string) => void;
    onPause: () => void;
    onResume: () => void;
    onRestart: () => void;
    onStop: () => void;
    onSave: () => void;
    onClear: () => void;
    onOpenSettings: () => void;
  } = $props();

  const characterCount = $derived(Array.from(text).length);
  const isOwnSession = $derived(status.source === TTSSource.SourceCompose);
  const working = $derived(
    isOwnSession &&
      (status.phase === TTSPhase.Generating ||
        status.phase === TTSPhase.Playing ||
        status.phase === TTSPhase.Paused),
  );
  const generating = $derived(status.phase === TTSPhase.Generating);
  const showPlayback = $derived(
    isOwnSession && status.phase !== TTSPhase.Idle && status.phase !== TTSPhase.Cancelled,
  );
  const configured = $derived(
    settings.enabled &&
      Boolean(settings.baseURL.trim() && settings.model.trim() && settings.voice.trim()),
  );
  const canSpeak = $derived(
    configured &&
      !unavailable &&
      !submitting &&
      !generating &&
      characterCount > 0 &&
      characterCount <= maximumCharacters,
  );
  const stateLabel = $derived.by(() => {
    if (!configured) return "Setup needed";
    if (!isOwnSession) return "Ready to generate";
    if (status.phase === TTSPhase.Generating) return "Generating";
    if (status.phase === TTSPhase.Playing) return "Speaking";
    if (status.phase === TTSPhase.Paused) return "Paused";
    if (status.phase === TTSPhase.Completed) return "Complete";
    if (status.phase === TTSPhase.Failed) return "Failed";
    return "Ready to generate";
  });

  function composerKey(event: KeyboardEvent) {
    if (
      event.key !== "Enter" ||
      !event.ctrlKey ||
      event.altKey ||
      event.metaKey ||
      event.shiftKey ||
      event.isComposing
    )
      return;
    event.preventDefault();
    if (!event.repeat && canSpeak) onSpeak(text);
  }

  const failed = $derived(isOwnSession && status.phase === TTSPhase.Failed);
</script>

<section
  class="@container flex min-h-0 flex-1 flex-col overflow-hidden rounded-2xl border border-hairline bg-card"
  aria-label="Speech composer"
>
  <div class="flex h-14 shrink-0 items-center gap-3 border-b border-hairline px-3">
    <h2 class="sr-only">Text to speech</h2>
    <div class="min-w-0 flex-1">
      {#if quickSettings}{@render quickSettings()}{:else}<span class="text-sm font-medium"
          >Text to speech</span
        >{/if}
    </div>
    <span
      class="flex shrink-0 items-center gap-2 text-xs text-muted-foreground"
      role="status"
      title="Local configuration only; connection checks appear in the footer."
    >
      <span
        class={cn(
          "size-1.5 rounded-full",
          failed ? "bg-destructive" : working ? "bg-primary" : "bg-border",
        )}
      ></span>
      {stateLabel}
    </span>
  </div>
  <div class="flex min-h-24 flex-1 flex-col overflow-y-auto">
    <label for="speech-composer-text" class="sr-only">Text to speak</label>
    <Textarea
      id="speech-composer-text"
      bind:value={text}
      aria-invalid={characterCount > maximumCharacters}
      aria-describedby="speech-character-count speech-compose-shortcut"
      aria-keyshortcuts="Control+Enter"
      onkeydown={composerKey}
      class="field-sizing-fixed min-h-24 flex-1 resize-none rounded-none border-0 bg-transparent px-6 py-4 text-[15px] leading-8 focus-visible:ring-2 focus-visible:ring-inset"
      placeholder="Write or paste text to speak…"
    />
  </div>
  <p id="speech-compose-shortcut" class="sr-only">
    Press Ctrl+Enter to speak. Enter adds a new line. Editing or clearing this draft does not change
    the current audio. Speak generates this draft and replaces the current audio.
  </p>
  <div class="flex h-14 shrink-0 items-center justify-between gap-3 border-t border-hairline px-4">
    <span
      id="speech-character-count"
      class={cn(
        "text-xs tabular-nums",
        characterCount > maximumCharacters ? "text-destructive" : "text-muted-foreground",
      )}
      aria-label="Character count"
    >
      {characterCount.toLocaleString()} / {maximumCharacters.toLocaleString()}
      {#if characterCount > maximumCharacters}<span role="status">
          · Shorten text to speak</span
        >{/if}
    </span>
    <div class="flex items-center gap-2">
      <Button variant="ghost" size="sm" disabled={!text} onclick={() => (text = "")}>Clear</Button>
      {#if !configured}
        <Button
          variant="outline"
          size="sm"
          aria-label="Text to speech settings"
          onclick={onOpenSettings}><SettingsIcon />Set up speech</Button
        >
      {:else}
        <Button
          size="sm"
          class="min-w-28"
          disabled={!canSpeak}
          title="Generate this text and replace the current audio"
          onclick={() => onSpeak(text)}
        >
          {#if working && status.phase === TTSPhase.Generating}<LoaderCircleIcon
              class="animate-spin motion-reduce:animate-none"
            />{:else}<Volume2Icon />{/if}
          {working && status.phase === TTSPhase.Generating
            ? "Generating…"
            : failed
              ? "Try again"
              : "Speak"}
          {#if !working}<kbd
              aria-hidden="true"
              class="ml-1 hidden text-[10px] opacity-70 @sm:inline">Ctrl+Enter</kbd
            >{/if}
        </Button>
      {/if}
    </div>
  </div>
  <div
    class="flex h-24 shrink-0 flex-col justify-center overflow-y-auto border-t border-hairline bg-layer-fill"
    aria-label="Generated audio"
  >
    {#if showPlayback}
      <PlaybackBar
        {status}
        {onPause}
        {onResume}
        {onRestart}
        {onSeek}
        {seeking}
        {saving}
        {onStop}
        {onSave}
        {onClear}
        {onOpenSettings}
        embedded
      />
    {:else}
      <div class="flex items-center gap-3 px-4 py-3">
        <Volume2Icon class="size-4 shrink-0 text-muted-foreground" />
        <div class="min-w-0 space-y-1">
          <p class="text-sm font-medium">
            {configured ? "Ready when you are" : "Choose a connection and voice"}
          </p>
          <p class="text-xs text-muted-foreground">
            {configured
              ? "Press Speak to generate audio. Playback and save controls appear here."
              : "Open Speech settings above to get started. You can write your text now."}
          </p>
        </div>
      </div>
    {/if}
  </div>
</section>
