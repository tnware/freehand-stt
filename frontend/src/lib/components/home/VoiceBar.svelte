<script lang="ts">
  import { Button } from "$lib/components/ui/button";
  import FeedbackDetails from "$lib/components/common/FeedbackDetails.svelte";
  import ShortcutKeys from "$lib/components/common/ShortcutKeys.svelte";
  import Waveform from "./Waveform.svelte";
  import { levels } from "$lib/stores/levels.svelte";
  import { CaptureClock } from "$lib/utils/captureClock.svelte";
  import {
    canToggleRecording,
    isCopyRequired,
    isFailure,
  } from "$lib/utils/status";
  import {
    AutoStopState,
    RecordingMode,
    SegmentPhase,
    State,
    VADState,
    type Status,
  } from "$lib/state";

  let {
    status,
    now,
    busy = false,
    availability = "",
    microphone = "System default",
    toggleShortcut = "",
    onToggle,
    onCancel,
    onOpenSettings,
  }: {
    status: Status;
    now: number;
    busy?: boolean;
    availability?: string;
    microphone?: string;
    toggleShortcut?: string;
    onToggle: () => Promise<void>;
    onCancel: () => Promise<void>;
    onOpenSettings: () => void;
  } = $props();

  let pending = $state(false);
  const recording = $derived(status.state === State.Recording);
  const failed = $derived(isFailure(status));
  const recovery = $derived(isCopyRequired(status));
  const captureClock = new CaptureClock();
  $effect(() => captureClock.update(status, now));
  const clock = $derived(
    `${String(Math.floor(captureClock.seconds / 60)).padStart(2, "0")}:${String(captureClock.seconds % 60).padStart(2, "0")}`,
  );
  const label = $derived.by(() => {
    if (recording)
      return status.recordingMode === RecordingMode.RecordingHold
        ? "Recording · release shortcut to finish"
        : "Recording";
    if (status.state === State.Transcribing) return "Transcribing audio";
    if (status.state === State.PostProcessing) return "Cleaning up transcript";
    if (status.state === State.Ready) return "Checking insertion target";
    if (status.state === State.Cancelling) return "Cancelling dictation";
    if (recovery) return "Ready to copy";
    if (failed)
      return status.startRejected
        ? "Recording could not start"
        : "Dictation could not be completed";
    return availability || "Ready to dictate";
  });
  const checkpoint = $derived(
    status.segmentNumber &&
      status.segmentPhase === SegmentPhase.SegmentTranscribing
      ? `Transcribing segment ${status.segmentNumber}`
      : status.segmentNumber &&
          status.segmentPhase === SegmentPhase.SegmentCompleted
        ? `Segment ${status.segmentNumber} transcribed`
        : "",
  );
  async function run(action: () => Promise<void>) {
    if (pending) return;
    pending = true;
    try {
      await action();
    } finally {
      pending = false;
    }
  }
</script>

<section
  class="flex flex-col gap-2 rounded-lg border border-hairline bg-card px-3 py-2.5"
  aria-label="Voice capture"
>
  <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
    <div class="min-w-0 flex-1">
      <p
        class="text-[13px] font-medium"
        class:text-destructive={failed}
        class:text-warning={recovery}
        role="status"
      >
        {label}
      </p>
      <div
        class="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground"
      >
        {#if recording || captureClock.seconds > 0}<span
            class="font-mono tabular-nums"
            aria-label="Capture duration">{clock}</span
          >{/if}
        {#if recording && status.autoStopState === AutoStopState.AutoStopCountdown}
          <span>Silence detected. Speak again to keep recording.</span>
        {:else if recording && checkpoint}<span>{checkpoint}</span>
        {:else if recording}<span
            >{status.vadState === VADState.VADSilence
              ? "Waiting for speech"
              : "Listening"}</span
          >
        {:else if status.state === State.Idle && !availability && toggleShortcut}
          <ShortcutKeys
            value={toggleShortcut}
            label="Toggle recording shortcut"
          />
          <span>to record from your application</span>
        {:else if availability}<button
            type="button"
            class="text-accent-text hover:underline"
            onclick={onOpenSettings}>Open transcription settings</button
          >{/if}
      </div>
    </div>
    <div class="flex shrink-0 items-center gap-2">
      {#if failed}<FeedbackDetails
          title={label}
          label="Dictation error details"
          message={status.message || "The recording could not be transcribed."}
          actionLabel="Transcription settings"
          onAction={onOpenSettings}
        />{/if}
      {#if status.canCancel}
        <Button
          variant="outline"
          size="sm"
          disabled={pending || status.state === State.Cancelling}
          onclick={() => run(onCancel)}
          >{status.state === State.Cancelling
            ? "Cancelling…"
            : "Cancel"}</Button
        >
      {/if}
      {#if recording || status.state === State.Idle || status.state === State.Failed}
        <Button
          size="sm"
          disabled={pending ||
            !canToggleRecording(status, !recording && (busy || !!availability))}
          onclick={() => run(onToggle)}
          aria-label={recording ? "Stop recording" : "Start recording"}
          >{recording
            ? "Stop recording"
            : failed || recovery
              ? "Record again"
              : "Record"}</Button
        >
      {/if}
    </div>
  </div>
  {#if recording}<div class="flex items-center gap-3">
      <div class="min-w-0 flex-1">
        <Waveform
          active
          quiet={status.vadState === VADState.VADSilence}
          history={levels.history}
        />
      </div>
      <span
        class="max-w-[40%] truncate text-[10px] text-muted-foreground"
        title={microphone}>{microphone}</span
      >
    </div>{/if}
</section>
