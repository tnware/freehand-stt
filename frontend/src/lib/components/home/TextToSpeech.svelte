<script lang="ts">
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Textarea } from "$lib/components/ui/textarea";
  import {
    TTSPhase,
    TTSSource,
    type Settings,
    type TTSStatus,
  } from "$lib/state";
  import { cn } from "$lib/utils";

  const maximumCharacters = 4096;

  let {
    text = $bindable(""),
    settings,
    status,
    unavailable = false,
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
    settings: Settings["textToSpeech"];
    status: TTSStatus;
    unavailable?: boolean;
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
  const showPlayback = $derived(
    isOwnSession &&
      status.phase !== TTSPhase.Idle &&
      status.phase !== TTSPhase.Cancelled,
  );
  const configured = $derived(
    settings.enabled &&
      Boolean(
        settings.baseURL.trim() &&
          settings.model.trim() &&
          settings.voice.trim(),
      ),
  );
  const canSpeak = $derived(
    configured &&
      !unavailable &&
      !working &&
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

  function compactLabel(value: string, fallback: string): string {
    const trimmed = value.trim();
    return trimmed ? (trimmed.split("/").at(-1) ?? trimmed) : fallback;
  }

  const failed = $derived(isOwnSession && status.phase === TTSPhase.Failed);
</script>

<div class="flex min-h-[360px] flex-1 flex-col gap-3">
  <section
    class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-hairline bg-layer-fill"
    aria-label="Speech composer"
  >
    <div class="flex flex-wrap items-start justify-between gap-3 px-5 pt-5">
      <div class="space-y-1">
        <h2 class="text-lg font-semibold tracking-tight">
          Write something to speak
        </h2>
        <p class="text-[13px] text-muted-foreground">
          Turn your text into audio with your selected voice.
        </p>
      </div>
      <span
        class="flex items-center gap-2 text-xs text-secondary-foreground"
        title="Local configuration only; connection checks appear in the footer."
      >
        <span
          class={cn(
            "size-1.5 rounded-full",
            failed
              ? "bg-destructive"
              : working
                ? "bg-primary"
                : configured
                  ? "bg-success"
                  : "bg-border",
          )}
        ></span>{stateLabel}
      </span>
    </div>
    <div class="flex min-h-0 flex-1 flex-col p-5">
      <Textarea
        bind:value={text}
        maxlength={maximumCharacters}
        disabled={working}
        class="field-sizing-fixed min-h-32 w-full flex-1 resize-none bg-well text-sm leading-relaxed"
        placeholder="Enter text for Freehand to read aloud…"
        aria-label="Text to speak"
      />
      <div
        class="mt-3 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground"
      >
        <span
          class="min-w-0 truncate"
          title={`${settings.model || "No model"} · ${settings.voice || "No voice"}`}
          >{compactLabel(settings.model, "No model")} · {compactLabel(
            settings.voice,
            "No voice",
          )}</span
        >
        <span class="shrink-0 tabular-nums"
          >{characterCount.toLocaleString()} / {maximumCharacters.toLocaleString()}</span
        >
      </div>
    </div>
    <div
      class="flex flex-wrap items-center justify-between gap-3 border-t border-hairline px-5 py-4"
    >
      <p class="max-w-sm flex-1 text-xs leading-relaxed text-muted-foreground">
        {configured
          ? "Audio is generated when you press Speak."
          : "Choose a speech connection, model, and voice to get started."}
      </p>
      <div class="flex items-center gap-2">
        {#if !configured}<Button variant="outline" onclick={onOpenSettings}
            ><SettingsIcon />Text to speech settings</Button
          >
        {:else}
          <Button
            variant="ghost"
            disabled={!text || working}
            onclick={() => (text = "")}>Clear</Button
          >
          <Button
            size="lg"
            class="min-w-28"
            disabled={!canSpeak}
            onclick={() => onSpeak(text)}
          >
            {#if working && status.phase === TTSPhase.Generating}<LoaderCircleIcon
                class="animate-spin motion-reduce:animate-none"
              />{:else}<Volume2Icon />{/if}
            {working && status.phase === TTSPhase.Generating
              ? "Generating…"
              : "Speak"}
          </Button>
        {/if}
      </div>
    </div>
  </section>

  {#if showPlayback}
    <PlaybackBar
      {status}
      {onPause}
      {onResume}
      {onRestart}
      {onStop}
      {onSave}
      {onClear}
    />
  {/if}
</div>
