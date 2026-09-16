<script lang="ts">
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import type { SeekRequest } from "$bindings/tts";
  import type { Snippet } from "svelte";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import SquarePenIcon from "@lucide/svelte/icons/square-pen";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import WorkflowSettingsButton from "./WorkflowSettingsButton.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
  import {
    TTSPhase,
    TTSSource,
    type Settings,
    type TTSStatus,
  } from "$lib/state";
  import { cn } from "$lib/utils";

  import { session } from "$lib/stores/session.svelte";
  import { shortcutKeyLabels, shortcutSpokenLabel } from "$lib/utils/shortcuts";

  const maximumCharacters = 4096;
  // Preserve the functional Control+Enter chord on both platforms.
  const composeShortcut = "Ctrl+Enter";
  const platform = $derived(session.editor.applied?.platform);
  const shortcutVisual = $derived(
    shortcutKeyLabels(composeShortcut, platform).join("+"),
  );
  const shortcutSpoken = $derived(
    shortcutSpokenLabel(composeShortcut, platform),
  );

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
    optionsDisabled = false,
    showSettings = true,
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
    optionsDisabled?: boolean;
    showSettings?: boolean;
  } = $props();

  const uid = $props.id();
  const setupGuidanceID = `${uid}-speech-setup`;
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
    isOwnSession &&
      status.phase !== TTSPhase.Idle &&
      status.phase !== TTSPhase.Cancelled,
  );
  const configured = $derived(
    settings.enabled &&
      Boolean(
        (settings.managedInstanceID || settings.baseURL.trim()) &&
        settings.model.trim() &&
        settings.voice.trim(),
      ),
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
  class="@container flex min-h-0 flex-1 flex-col overflow-hidden"
  aria-label="Speech composer"
>
  <div class="workbench-toolbar flex-wrap gap-y-1 py-1">
    <SquarePenIcon class="content-section-icon" aria-hidden="true" />
    <h2 class="content-title">Compose</h2>
    <div
      class="mr-auto min-w-0"
      role="status"
      title="Local configuration only; connection checks appear in the footer."
    >
      <StatusBadge
        tone={failed ? "danger" : working ? "accent" : "neutral"}
        class={working ? "font-semibold" : ""}>{stateLabel}</StatusBadge
      >
    </div>
    {#if quickSettings}
      <div class="min-w-0 shrink-0">{@render quickSettings()}</div>
    {/if}
    {#if showSettings}<WorkflowSettingsButton
        label="Text to speech settings"
        disabled={optionsDisabled}
        onclick={onOpenSettings}
      />{/if}
  </div>
  <div class="flex min-h-24 flex-1 flex-col overflow-y-auto">
    {#if !configured}
      <div
        class="content-callout mx-3 mt-3 flex shrink-0 flex-wrap items-center gap-x-4 gap-y-2 px-3 py-2.5"
      >
        <p
          id={setupGuidanceID}
          class="min-w-0 flex-1 text-xs leading-relaxed text-accent-text"
        >
          Choose a connection, model, and voice in speech settings. You can
          write your text now.
        </p>
        <Button
          variant="outline"
          size="sm"
          class="shrink-0 border-accent-edge bg-background text-accent-text"
          onclick={onOpenSettings}><SettingsIcon />Configure speech</Button
        >
      </div>
    {/if}
    <label for="speech-composer-text" class="sr-only">Text to speak</label>
    <Textarea
      id="speech-composer-text"
      bind:value={text}
      aria-invalid={characterCount > maximumCharacters}
      aria-describedby={`speech-character-count speech-compose-shortcut${!configured ? ` ${setupGuidanceID}` : ""}`}
      aria-keyshortcuts="Control+Enter"
      onkeydown={composerKey}
      class="field-sizing-fixed min-h-24 flex-1 resize-none rounded-none border-0 bg-transparent px-3 py-3.5 text-[15px] leading-[26px] text-foreground focus-visible:ring-2 focus-visible:ring-inset"
      placeholder="Write or paste text to speak…"
    />
  </div>
  <p id="speech-compose-shortcut" class="sr-only">
    Press {shortcutSpoken} to speak. Enter adds a new line. Editing or clearing this
    draft does not change the current audio. Speak generates this draft and replaces
    the current audio.
  </p>
  <div
    class="flex min-h-9 shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-1 border-t border-hairline px-3 py-1"
  >
    <span
      id="speech-character-count"
      class={cn(
        "text-xs tabular-nums",
        characterCount > maximumCharacters
          ? "text-destructive"
          : "text-muted-foreground",
      )}
      aria-label="Character count"
    >
      {characterCount.toLocaleString()} / {maximumCharacters.toLocaleString()}
      {#if characterCount > maximumCharacters}<span role="status">
          · Shorten text to speak</span
        >{/if}
    </span>
    <div class="ml-auto flex shrink-0 items-center gap-2">
      <Button
        variant="ghost"
        size="sm"
        disabled={!text}
        onclick={() => (text = "")}>Clear</Button
      >
      <Button
        size="sm"
        class="min-w-24"
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
            class="ml-1 hidden text-2xs opacity-70 @sm:inline"
            >{shortcutVisual}</kbd
          >{/if}
      </Button>
    </div>
  </div>
  {#if configured || showPlayback}
    <div
      class="flex min-h-20 max-h-32 shrink-0 flex-col justify-center overflow-y-auto border-t border-hairline bg-layer-fill"
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
        <div class="flex items-center gap-3 px-3 py-3">
          <Volume2Icon class="content-section-icon" aria-hidden="true" />
          <div class="min-w-0 space-y-1">
            <p class="content-section-title">Ready when you are</p>
            <p class="content-meta">
              Press Speak to generate audio. Playback and save controls appear
              here.
            </p>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</section>
