<script lang="ts">
  import CheckIcon from "@lucide/svelte/icons/check";
  import ClipboardIcon from "@lucide/svelte/icons/clipboard";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import MicIcon from "@lucide/svelte/icons/mic";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import SlidersIcon from "@lucide/svelte/icons/sliders-horizontal";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import WandSparklesIcon from "@lucide/svelte/icons/wand-sparkles";
  import XIcon from "@lucide/svelte/icons/x";
  import TransportShell from "$lib/components/home/TransportShell.svelte";
  import Waveform from "$lib/components/home/Waveform.svelte";
  import { levels } from "$lib/stores/levels.svelte";
  import { cn } from "$lib/utils";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import {
    AutoStopState,
    RecordingMode,
    SegmentPhase,
    State,
    VADState,
    type Status,
  } from "$lib/state";
  import {
    canToggleRecording,
    isCopyRequired,
    isFailure,
    isRecording,
    railPhase,
    showIdleShortcutGuidance,
    type RailPhase,
  } from "$lib/utils/status";

  let {
    status,
    busy = false,
    toggleShortcut = "",
    model = "",
    processingModel = "",
    microphone = "default",
    onToggle,
    onCancel,
    onCopy,
    onOpenSettings,
  }: {
    status: Status;
    busy?: boolean;
    toggleShortcut?: string;
    model?: string;
    processingModel?: string;
    microphone?: string;
    onToggle: () => void;
    onCancel: () => void;
    onCopy: () => Promise<boolean>;
    onOpenSettings: () => void;
  } = $props();

  const recording = $derived(isRecording(status));
  const holdRecording = $derived(
    recording && status.recordingMode === RecordingMode.RecordingHold,
  );
  const vadSilence = $derived(recording && status.vadState === VADState.VADSilence);
  const vadSpeech = $derived(recording && status.vadState === VADState.VADSpeech);
  const autoStopCountdown = $derived(
    recording && status.autoStopState === AutoStopState.AutoStopCountdown,
  );
  const showShortcut = $derived(showIdleShortcutGuidance(status, toggleShortcut));
  // Copy-required is an outcome waiting on the user, not a fault: it keeps the
  // accent rather than a dead grey dot.
  const waiting = $derived(isCopyRequired(status));
  const failed = $derived(isFailure(status));
  const canToggle = $derived(canToggleRecording(status, busy));
  const showRecordControl = $derived(
    status.state === State.Idle || recording || (status.state === State.Failed && !waiting),
  );
  const manualCopy = $derived(waiting && status.message === "Transcript ready to copy");
  // Once a take is sent, the meter is a record of what was captured rather than
  // a live level, so it holds its shape at a lower weight.
  const held = $derived(
    status.state === State.Transcribing ||
      status.state === State.PostProcessing ||
      status.state === State.Ready ||
      status.state === State.Cancelling,
  );
  // The outcome states have no level left to show, so the stage stops being a
  // meter and carries words instead.
  const textStage = $derived(waiting || failed);

  // This is capture duration, not end-to-end job latency. Freeze it as soon as
  // recording ends so transcription and cleanup do not inflate the displayed
  // recording time. Failed and copy-required states can lose StartedAt, so the
  // final visible value remains available until the run returns to idle.
  let now = $state(Date.now());
  let visibleSeconds = $state(0);
  let clockWasRecording = false;
  $effect(() => {
    const state = status.state;
    const startedAt = status.startedAt;
    if (state === State.Idle) {
      visibleSeconds = 0;
      clockWasRecording = false;
      return;
    }
    const update = () => {
      now = Date.now();
      visibleSeconds = startedAt
        ? Math.max(0, Math.floor((now - Date.parse(startedAt)) / 1000))
        : 0;
    };

    if (!recording) {
      if (clockWasRecording && startedAt) update();
      clockWasRecording = false;
      return;
    }

    clockWasRecording = true;
    update();
    const timer = setInterval(update, autoStopCountdown ? 100 : 1000);
    return () => clearInterval(timer);
  });

  const clock = $derived.by(() => {
    const seconds = visibleSeconds;
    const minutes = Math.floor(seconds / 60);
    return `${String(minutes).padStart(2, "0")}:${String(seconds % 60).padStart(2, "0")}`;
  });

  // The ruler labels the span the meter is actually showing, so its ticks come
  // from the elapsed clock rather than being drawn as decoration.
  const ruler = $derived.by(() => {
    const span = Math.max(4, visibleSeconds);
    return [1, 2, 3].map((step) => `${Math.round((span * step) / 3)}s`);
  });

  const autoStopRemaining = $derived.by(() => {
    if (!autoStopCountdown || !status.autoStopDeadline) return "";
    const milliseconds = Math.max(0, Date.parse(status.autoStopDeadline) - now);
    return `${(Math.ceil(milliseconds / 100) / 10).toFixed(1)}s`;
  });

  const phase = $derived(railPhase(status));

  const checkpointGeneration = $derived(status.state === State.Recording ? status.generation : 0);
  const checkpointSegment = $derived(status.segmentNumber);
  const checkpointPhase = $derived(status.segmentPhase);

  // Silence splitting closes and transcribes segments while capture continues.
  // Reflect those real checkpoints without changing the recording state.
  let checkpointRail = $state<RailPhase>("hidden");
  let checkpointAnnouncement = $state("");
  $effect(() => {
    const generation = checkpointGeneration;
    const segment = checkpointSegment;
    const segmentPhase = checkpointPhase;
    if (!generation || !segment) {
      checkpointRail = "hidden";
      checkpointAnnouncement = "";
      return;
    }
    if (segmentPhase === SegmentPhase.SegmentTranscribing) {
      checkpointRail = "working";
      checkpointAnnouncement = `Transcribing segment ${segment}.`;
      return;
    }
    if (segmentPhase === SegmentPhase.SegmentCompleted) {
      checkpointRail = "done";
      checkpointAnnouncement = `Segment ${segment} transcribed.`;
      const railTimer = setTimeout(() => {
        checkpointRail = "hidden";
        checkpointAnnouncement = "";
      }, 1500);
      return () => clearTimeout(railTimer);
    }
    checkpointRail = "hidden";
    checkpointAnnouncement = "";
  });

  // A run that ends in insertion goes straight back to idle, which would snap
  // the rail off mid-travel. Hold it full for a beat so the work visibly
  // finishes instead of vanishing.
  let completed = $state(false);
  let previous: RailPhase = "hidden";
  $effect(() => {
    const next = phase;
    const was = previous;
    previous = next;
    if (was === "working" && next === "hidden") {
      completed = true;
      const timer = setTimeout(() => (completed = false), 1500);
      return () => clearTimeout(timer);
    }
    if (next !== "hidden") completed = false;
  });

  const postRecordingRail = $derived(phase === "hidden" && completed ? "done" : phase);
  const rail = $derived(postRecordingRail === "hidden" ? checkpointRail : postRecordingRail);

  let copied = $state(false);
  let stateAnnouncement = $state("");
  let previousState: State | undefined;
  let previousGeneration = -1;

  $effect(() => {
    const state = status.state;
    const generation = status.generation;
    const message = status.message?.trim() ?? "";
    if (previousState === undefined) {
      previousState = state;
      previousGeneration = generation;
      return;
    }
    if (state === previousState && generation === previousGeneration) return;

    if (state === State.Recording) stateAnnouncement = "Recording started.";
    else if (state === State.Transcribing)
      stateAnnouncement = "Recording stopped. Transcribing audio.";
    else if (state === State.PostProcessing)
      stateAnnouncement = "Raw transcript ready. Post-processing transcript.";
    else if (state === State.Ready)
      stateAnnouncement = "Transcript ready. Verifying the original focus target.";
    else if (state === State.Cancelling) stateAnnouncement = "Cancelling dictation.";
    else if (waiting) stateAnnouncement = message || "Transcript ready to copy.";
    else if (state === State.Failed)
      stateAnnouncement = message || "Dictation failed. Nothing was inserted.";
    else if (state === State.Idle && previousState !== State.Idle) {
      stateAnnouncement = message || "Dictation complete. Ready for another recording.";
    }

    previousState = state;
    previousGeneration = generation;
  });

  async function copyTranscript() {
    if (!(await onCopy())) return;
    copied = true;
    setTimeout(() => (copied = false), 1600);
  }
</script>

<TransportShell rail={rail} busy={held} state={status.state}>
  {#snippet control()}
    {#if waiting}
      <span
        class="grid size-[62px] place-items-center rounded-full border border-accent-edge bg-accent-wash text-accent-text"
        aria-hidden="true"
      >
        <ClipboardIcon class="size-[22px]" />
      </span>
    {:else if failed}
      <span
        class="grid size-[62px] place-items-center rounded-full border border-destructive/30 bg-destructive/10 text-destructive"
        aria-hidden="true"
      >
        <TriangleAlertIcon class="size-[22px]" />
      </span>
    {:else if held}
      <span
        class="grid size-[62px] place-items-center rounded-full border border-hairline bg-control-fill text-primary"
        aria-hidden="true"
      >
        {#if status.state === State.PostProcessing}
          <WandSparklesIcon class="size-[22px]" />
        {:else}
          <LoaderCircleIcon class="size-[22px] animate-spin motion-reduce:animate-none" />
        {/if}
      </span>
    {:else}
      <!-- The record glyph morphs circle to square while recording. -->
      <button
        type="button"
        class="rec"
        class:live={recording}
        class:invisible={!showRecordControl}
        disabled={!canToggle}
        aria-label={recording ? "Stop recording" : "Start recording"}
        title={recording ? "Stop recording" : "Start recording"}
        onclick={onToggle}
      >
        <span class="glyph"></span>
      </button>
    {/if}
  {/snippet}

  {#snippet stage()}
    {#if waiting}
      <span class="caption text-accent-text">Ready to copy</span>
      <p class="mt-1.5 text-[13.5px] font-medium">
        {manualCopy ? "Manual copy is selected for this profile." : "Focus moved before insertion."}
      </p>
      <p class="figure mt-1 text-[10.5px] text-muted-foreground">
        The transcript is held in memory · the audio has been discarded
      </p>
    {:else if failed}
      <p class="text-[13.5px] font-semibold">Dictation could not be completed</p>
      <p class="figure mt-1 truncate text-[11px] text-destructive" title={status.message}>
        {status.message || "The endpoint did not return a transcript."}
      </p>
      <p class="figure mt-1 text-[10.5px] text-muted-foreground">
        Nothing was inserted. The audio has been discarded.
      </p>
    {:else}
      <Waveform
        active={recording}
        quiet={vadSilence}
        {held}
        history={levels.history}
      />
      <div
        class="mt-2 flex min-h-5 items-center justify-between gap-3 border-t border-hairline pt-1.5"
      >
        {#if showShortcut}
          <span class="flex min-w-0 items-center gap-2 text-[11.5px] text-secondary-foreground">
            <span class="roomy shrink-0">Press</span>
            <ShortcutKeys value={toggleShortcut} label="Toggle recording shortcut" />
            <span class="roomy shrink-0">from any application to start</span>
          </span>
          <span class="roomy figure shrink-0 text-[9.5px] text-ink-quiet">
            {microphone}
          </span>
        {:else if autoStopCountdown}
          <span class="min-w-0 truncate text-[11.5px] font-medium text-warning">
            Silence detected — speak again to keep recording
          </span>
          <span class="figure shrink-0 text-[9.5px] text-ink-quiet">{clock}</span>
        {:else if recording}
          <span class="min-w-0 truncate text-[11.5px] text-secondary-foreground">
            {holdRecording ? "Release the shortcut to finish" : "Use the shortcut again to finish"}
          </span>
          <span class="figure flex shrink-0 items-center gap-3 text-[9.5px] text-ink-quiet">
            {#each ruler as tick (tick)}
              <span>{tick}</span>
            {/each}
          </span>
        {:else if status.state === State.Transcribing}
          <span class="figure min-w-0 truncate text-[11px] text-secondary-foreground">
            Waiting on {model || "the speech-to-text endpoint"}
          </span>
          <span class="figure shrink-0 text-[9.5px] text-success">audio sent ✓</span>
        {:else if status.state === State.PostProcessing}
          <span class="figure min-w-0 truncate text-[11px] text-secondary-foreground">
            Cleaning up with {processingModel || "the post-processor"}
          </span>
          <span class="figure shrink-0 text-[9.5px] text-success">raw transcript kept ✓</span>
        {:else if status.state === State.Ready}
          <span class="figure min-w-0 truncate text-[11px] text-secondary-foreground">
            Verifying the original focus target
          </span>
          <span class="figure shrink-0 text-[9.5px] text-success">transcript ready ✓</span>
        {:else if status.state === State.Cancelling}
          <span class="figure min-w-0 truncate text-[11px] text-secondary-foreground">
            Discarding this recording
          </span>
        {/if}
      </div>
    {/if}
  {/snippet}

  {#snippet readout()}
    <span
      class={cn(
        "figure text-[34px] leading-none font-medium tracking-[-0.02em]",
        status.state === State.Idle || textStage ? "text-ink-disabled" : "text-foreground",
      )}
    >
      {clock}
    </span>

    <div class="flex min-h-5 items-center gap-3.5">
      {#if autoStopCountdown}
        <span class="flex items-center gap-2">
          <span class="size-[7px] rounded-full bg-warning shadow-[0_0_8px_var(--warning)]"></span>
          <span class="caption text-warning">Stopping</span>
        </span>
        <span class="figure text-xs font-semibold text-warning">{autoStopRemaining}</span>
      {:else if recording}
        <span class="flex items-center gap-2">
          <span
            class={cn(
              "size-[7px] rounded-full",
              vadSpeech ? "bg-primary shadow-[0_0_8px_var(--primary)]" : "bg-meter-rest",
            )}
          ></span>
          <span class="caption text-secondary-foreground">
            {vadSpeech ? "Speech" : vadSilence ? "Silence" : "Live"}
          </span>
        </span>
        {#if checkpointSegment}
          <span class="flex items-center gap-1.5">
            <span class="caption">Seg</span>
            <span class="figure text-xs font-semibold">
              {String(checkpointSegment).padStart(2, "0")}
            </span>
          </span>
        {/if}
      {:else if held}
        <span class="flex items-center gap-2">
          <span class="size-[7px] rounded-full bg-primary shadow-[0_0_8px_var(--primary)]"></span>
          <span class="caption text-secondary-foreground">
            {status.state === State.PostProcessing
              ? "Cleanup"
              : status.state === State.Cancelling
                ? "Cancelling"
                : "In flight"}
          </span>
        </span>
      {:else if waiting}
        <span class="flex items-center gap-2">
          <span class="size-[7px] rounded-full bg-primary"></span>
          <span class="caption text-secondary-foreground">Waiting on you</span>
        </span>
      {:else if failed}
        <span class="flex items-center gap-2">
          <span class="size-[7px] rounded-full bg-destructive"></span>
          <span class="caption text-destructive">Failed</span>
        </span>
      {:else}
        <span class="flex items-center gap-2">
          <span class="size-[7px] rounded-full bg-success"></span>
          <span class="caption text-secondary-foreground">Ready</span>
        </span>
      {/if}
    </div>

    <!-- One action slot. It keeps its height in every state so the meter never
         shifts when a run begins or ends. -->
    <div class="flex min-h-[26px] items-center gap-1.5">
      {#if waiting}
        <button
          type="button"
          class="act primary"
          aria-label={copied ? "Transcript copied" : "Copy transcript"}
          onclick={() => void copyTranscript()}
        >
          {#if copied}<CheckIcon class="size-3.5" />{:else}<ClipboardIcon class="size-3.5" />{/if}
          {copied ? "Copied" : "Copy transcript"}
        </button>
      {:else if failed}
        <button type="button" class="act quiet flex-1" onclick={onToggle} disabled={!canToggle}>
          <RotateCcwIcon class="size-3" />
          Retry
        </button>
        <button type="button" class="act quiet flex-1" onclick={onOpenSettings}>
          <SlidersIcon class="size-3" />
          Settings
        </button>
      {:else}
        <button
          type="button"
          class={cn("act quiet w-full", !status.canCancel && "invisible")}
          disabled={!status.canCancel}
          onclick={onCancel}
        >
          <XIcon class="size-3" />
          {recording ? "Discard recording" : "Cancel"}
        </button>
      {/if}
    </div>

    <span class="sr-only" aria-live="polite">{checkpointAnnouncement}</span>
    <span class="sr-only" aria-live="polite">
      {autoStopCountdown
        ? "Silence detected. Automatic stop countdown started. Speaking again cancels it."
        : ""}
    </span>
    <span class="sr-only" role="status" aria-live="polite" aria-atomic="true">
      {stateAnnouncement}
    </span>
  {/snippet}
</TransportShell>

<style>
  /*
   * Wording that earns its place at full width and only crowds the meter in a
   * narrow window. The shortcut keys themselves always stay: they are the
   * instruction, where the surrounding words are only politeness.
   */
  @container (max-width: 699px) {
    .roomy {
      display: none;
    }
  }

  .rec {
    display: grid;
    place-items: center;
    width: 3.875rem;
    height: 3.875rem;
    flex-shrink: 0;
    border: 1px solid color-mix(in srgb, var(--record) 78%, #ffffff);
    border-radius: 999px;
    background-color: var(--record);
    transition:
      opacity 180ms ease,
      box-shadow 180ms ease;
  }
  .rec:disabled {
    opacity: 0.38;
    cursor: default;
  }
  .rec .glyph {
    display: block;
    width: 18px;
    height: 18px;
    border-radius: 999px;
    background-color: #ffffff;
    transition:
      width 220ms cubic-bezier(0.4, 0, 0.2, 1),
      height 220ms cubic-bezier(0.4, 0, 0.2, 1),
      border-radius 220ms cubic-bezier(0.4, 0, 0.2, 1);
  }
  .rec.live .glyph {
    width: 17px;
    height: 17px;
    border-radius: 3px;
  }
  .rec.live {
    animation: pulse 1.8s cubic-bezier(0.4, 0, 0.2, 1) infinite;
  }
  @keyframes pulse {
    0% {
      box-shadow: 0 0 0 0 color-mix(in srgb, var(--record) 42%, transparent);
    }
    70% {
      box-shadow: 0 0 0 10px color-mix(in srgb, var(--record) 0%, transparent);
    }
    100% {
      box-shadow: 0 0 0 0 color-mix(in srgb, var(--record) 0%, transparent);
    }
  }
  .rec:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 2px;
  }

  /* The transport's own action button. It is smaller and quieter than a
     registry button because the readout column is a dense instrument panel,
     not a form. */
  .act {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.375rem;
    height: 1.625rem;
    padding: 0 0.625rem;
    border-radius: var(--radius-md);
    font-size: 0.719rem;
    white-space: nowrap;
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .act:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .act:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 2px;
  }
  .act.quiet {
    border: 1px solid var(--border);
    background-color: var(--control-fill);
    color: var(--secondary-foreground);
  }
  .act.quiet:not(:disabled):hover {
    background-color: var(--control-fill-hover);
    color: var(--foreground);
  }
  .act.primary {
    width: 100%;
    border: 1px solid var(--primary-hover);
    background-color: var(--primary);
    color: var(--primary-foreground);
    font-weight: 600;
  }
  .act.primary:not(:disabled):hover {
    background-color: var(--primary-hover);
  }

  @media (prefers-reduced-motion: reduce) {
    .rec,
    .rec .glyph {
      transition: none;
    }
    .rec.live {
      animation: none;
    }
  }
</style>
