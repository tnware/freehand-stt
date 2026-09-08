<script lang="ts">
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import PauseIcon from "@lucide/svelte/icons/pause";
  import PlayIcon from "@lucide/svelte/icons/play";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import SquareIcon from "@lucide/svelte/icons/square";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import { Progress } from "$lib/components/ui/progress";
  import { TTSPhase, TTSSource, type TTSStatus } from "$lib/state";

  let {
    embedded = false,
    status,
    onPause,
    onResume,
    onRestart,
    onStop,
    onSave,
    onClear,
  }: {
    embedded?: boolean;
    status: TTSStatus;
    onPause: () => void;
    onResume: () => void;
    onRestart: () => void;
    onStop: () => void;
    onSave: () => void;
    onClear: () => void;
  } = $props();

  const percent = $derived(
    status.durationMilliseconds > 0
      ? Math.min(100, (status.positionMilliseconds / status.durationMilliseconds) * 100)
      : 0,
  );
  const label = $derived(
    status.source === TTSSource.SourceCompose
      ? "Text to speech"
      : status.source === TTSSource.SourcePreview
        ? "Voice preview"
        : status.source === TTSSource.SourceFile
          ? "Audio file transcript"
          : status.source === TTSSource.SourceVoice
            ? "Voice transcript"
            : "Transcript playback",
  );
  const phaseLabel = $derived.by(() => {
    if (status.phase === TTSPhase.Generating) return "Generating";
    if (status.phase === TTSPhase.Paused) return "Paused";
    if (status.phase === TTSPhase.Completed) return "Complete";
    if (status.phase === TTSPhase.Failed) return "Failed";
    return "Playing";
  });
  const formatTime = (milliseconds: number): string => {
    const seconds = Math.max(0, Math.floor(milliseconds / 1000));
    return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
  };
</script>

<div
  class="shrink-0 border-hairline bg-secondary px-3 py-2"
  class:border-t={!embedded}
  aria-label="Speech playback"
  aria-live="polite"
>
  <div class="flex items-center gap-2.5">
    <span
      class="grid size-7 shrink-0 place-items-center rounded-full border border-accent-edge bg-accent-wash text-accent-text"
      aria-hidden="true"
    >
      {#if status.phase === TTSPhase.Generating}
        <LoaderCircleIcon class="size-3.5 animate-spin motion-reduce:animate-none" />
      {:else}
        <Volume2Icon class="size-3.5" />
      {/if}
    </span>
    <div class="min-w-0 flex-1">
      <div class="mb-1 flex items-center justify-between gap-3 text-xs">
        <span class="truncate font-medium">{label} · {phaseLabel}</span>
        <span class="shrink-0 font-mono text-muted-foreground tabular-nums">
          {formatTime(status.positionMilliseconds)} / {formatTime(status.durationMilliseconds)}
        </span>
      </div>
      <Progress value={percent} max={100} class="h-1" aria-label="Speech playback progress" />
    </div>
    <div class="flex shrink-0 items-center">
      {#if status.canPause}
        <TooltipButton
          variant="ghost"
          size="icon-sm"
          label="Pause speech playback"
          onclick={onPause}><PauseIcon /></TooltipButton
        >
      {:else if status.canResume}
        <TooltipButton
          variant="ghost"
          size="icon-sm"
          label="Resume speech playback"
          onclick={onResume}><PlayIcon /></TooltipButton
        >
      {/if}
      <TooltipButton
        variant="ghost"
        size="icon-sm"
        disabled={!status.canRestart}
        label="Restart speech playback"
        onclick={onRestart}><RotateCcwIcon /></TooltipButton
      >
      {#if status.canSave}
        <TooltipButton variant="ghost" size="icon-sm" label="Save generated speech" onclick={onSave}
          ><DownloadIcon /></TooltipButton
        >
      {/if}
      {#if status.canStop}
        <TooltipButton
          variant="ghost"
          size="icon-sm"
          label="Stop and release speech playback"
          onclick={onStop}><SquareIcon /></TooltipButton
        >
      {:else if status.canClear}
        <TooltipButton
          variant="ghost"
          size="icon-sm"
          label="Clear generated speech from memory"
          onclick={onClear}><Trash2Icon /></TooltipButton
        >
      {/if}
    </div>
  </div>
</div>
