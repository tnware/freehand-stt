<script lang="ts">
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import SquareIcon from "@lucide/svelte/icons/square";
  import XIcon from "@lucide/svelte/icons/x";
  import Stage, { type StageTone } from "./Stage.svelte";
  import Waveform from "$lib/components/home/Waveform.svelte";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import { levels } from "$lib/stores/levels.svelte";
  import { CaptureClock } from "$lib/utils/captureClock.svelte";
  import {
    AutoStopState,
    RecordingMode,
    State,
    VADState,
    type Status,
  } from "$lib/state";
  import { canToggleRecording, isRecording } from "$lib/utils/status";

  let {
    status,
    now,
    busy = false,
    toggleShortcut = "",
    microphone = "System default",
    availability = "",
    onToggle,
    onCancel,
    onOpenAudioSettings,
  }: {
    status: Status;
    now: number;
    busy?: boolean;
    toggleShortcut?: string;
    microphone?: string;
    /** Why capture is unavailable, empty when it is not. */
    availability?: string;
    onToggle: () => void;
    onCancel: () => void;
    onOpenAudioSettings: () => void;
  } = $props();

  const clock = new CaptureClock();
  $effect(() => clock.update(status, now));

  const recording = $derived(isRecording(status));
  const canToggle = $derived(canToggleRecording(status, busy));
  const quiet = $derived(recording && status.vadState === VADState.VADSilence);
  // Once a take is sent the meter is a record of what was captured rather than
  // a live level, so it holds its shape at a lower weight.
  const held = $derived(
    status.state === State.Transcribing ||
      status.state === State.PostProcessing ||
      status.state === State.Ready,
  );
  const tone = $derived<StageTone>(
    availability ? "bad" : recording ? "busy" : "ok",
  );
  const countdown = $derived(
    recording && status.autoStopState === AutoStopState.AutoStopCountdown,
  );
  const elapsed = $derived(
    `${String(Math.floor(clock.seconds / 60)).padStart(2, "0")}:${String(
      clock.seconds % 60,
    ).padStart(2, "0")}`,
  );
</script>

<Stage ordinal="01" label="Input" {tone}>
  <div class="flex items-center gap-2.5">
    <button
      type="button"
      class="grid size-8 shrink-0 place-items-center rounded-full border transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:opacity-40"
      class:is-recording={recording}
      disabled={!canToggle}
      onclick={onToggle}
      aria-label={recording ? "Stop recording" : "Start recording"}
      title={availability || (recording ? "Stop recording" : "Start recording")}
    >
      {#if recording}
        <SquareIcon class="size-3 fill-current" aria-hidden="true" />
      {:else}
        <span class="size-3 rounded-full bg-record" aria-hidden="true"></span>
      {/if}
    </button>
    <span class="min-w-0">
      <span class="figure block text-sm font-medium tabular-nums"
        >{elapsed}</span
      >
      <span class="block truncate text-[11px] text-muted-foreground"
        >{availability ||
          (countdown
            ? "Stopping on silence"
            : recording
              ? quiet
                ? "Silence"
                : "Capturing"
              : "Ready")}</span
      >
    </span>
    {#if recording}
      <button
        type="button"
        class="ml-auto grid size-6 shrink-0 place-items-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-subtle-fill-hover hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        onclick={onCancel}
        aria-label="Discard this recording"
        title="Discard this recording"
      >
        <XIcon class="size-3" aria-hidden="true" />
      </button>
    {/if}
  </div>

  <div
    class="rounded-md bg-well px-2.5 py-2"
    style="--meter-height: 2.75rem"
  >
    <Waveform
      active={recording}
      {quiet}
      {held}
      history={levels.history}
    />
  </div>

  <button
    type="button"
    class="flex w-full items-center justify-between gap-2 rounded-md px-1 py-0.5 text-left transition-colors hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
    onclick={onOpenAudioSettings}
  >
    <span class="truncate text-[13px] font-medium">{microphone}</span>
    <ChevronDownIcon
      class="size-3.5 shrink-0 text-muted-foreground"
      aria-hidden="true"
    />
  </button>

  {#snippet footer()}
    <span
      class="rounded-sm border border-border px-1.5 py-0.5 text-[11px] text-muted-foreground"
    >
      {status.recordingMode === RecordingMode.RecordingHold
        ? "Hold to talk"
        : "Toggle"}
    </span>
    {#if toggleShortcut}
      <ShortcutKeys value={toggleShortcut} label="Recording shortcut" />
    {/if}
  {/snippet}
</Stage>

<style>
  button.is-recording {
    border-color: var(--record-edge);
    background: var(--record-wash);
    color: var(--record);
  }
  button:not(.is-recording) {
    border-color: var(--border);
  }
  button:not(.is-recording):hover:not(:disabled) {
    background: var(--subtle-fill-hover);
  }
</style>
