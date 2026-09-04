<script lang="ts">
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import PauseIcon from "@lucide/svelte/icons/pause";
  import PlayIcon from "@lucide/svelte/icons/play";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import SquareIcon from "@lucide/svelte/icons/square";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import { Button } from "$lib/components/ui/button";
  import { Progress } from "$lib/components/ui/progress";
  import { TTSPhase, TTSSource, type TTSStatus } from "$lib/state";

  let { status, onPause, onResume, onRestart, onStop, onSave, onClear }: {
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

<div class="shrink-0 border-t border-hairline bg-secondary px-3 py-2" aria-label="Speech playback" aria-live="polite">
  <div class="flex items-center gap-2.5">
    <span class="grid size-7 shrink-0 place-items-center rounded-full border border-accent-edge bg-accent-wash text-accent-text" aria-hidden="true">
      {#if status.phase === TTSPhase.Generating}
        <LoaderCircleIcon class="size-3.5 animate-spin motion-reduce:animate-none" />
      {:else}
        <Volume2Icon class="size-3.5" />
      {/if}
    </span>
    <div class="min-w-0 flex-1">
      <div class="mb-1 flex items-center justify-between gap-3 text-[10px]">
        <span class="truncate font-medium">{label} · {phaseLabel}</span>
        <span class="shrink-0 font-mono text-muted-foreground tabular-nums">
          {formatTime(status.positionMilliseconds)} / {formatTime(status.durationMilliseconds)}
        </span>
      </div>
      <Progress value={percent} max={100} class="h-1" aria-label="Speech playback progress" />
    </div>
    <div class="flex shrink-0 items-center">
      {#if status.canPause}
        <Button variant="ghost" size="icon-xs" aria-label="Pause speech playback" onclick={onPause}><PauseIcon /></Button>
      {:else if status.canResume}
        <Button variant="ghost" size="icon-xs" aria-label="Resume speech playback" onclick={onResume}><PlayIcon /></Button>
      {/if}
      <Button variant="ghost" size="icon-xs" disabled={!status.canRestart} aria-label="Restart speech playback" onclick={onRestart}><RotateCcwIcon /></Button>
      {#if status.canSave}
        <Button variant="ghost" size="icon-xs" aria-label="Save generated speech" onclick={onSave}><DownloadIcon /></Button>
      {/if}
      {#if status.canStop}
        <Button variant="ghost" size="icon-xs" aria-label="Stop and release speech playback" onclick={onStop}><SquareIcon /></Button>
      {:else if status.canClear}
        <Button variant="ghost" size="icon-xs" aria-label="Clear generated speech from memory" onclick={onClear}><Trash2Icon /></Button>
      {/if}
    </div>
  </div>
</div>
