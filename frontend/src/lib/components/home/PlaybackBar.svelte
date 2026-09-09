<script lang="ts">
  import FeedbackDetails from "$lib/components/common/FeedbackDetails.svelte";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import PauseIcon from "@lucide/svelte/icons/pause";
  import PlayIcon from "@lucide/svelte/icons/play";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
  import SquareIcon from "@lucide/svelte/icons/square";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  import * as Menu from "$lib/components/ui/dropdown-menu";
  import { buttonVariants } from "$lib/components/ui/button";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import { Slider } from "$lib/components/ui/slider";
  import type { SeekRequest } from "$bindings/tts";
  import {
    PlaybackSeek,
    playbackSteps,
    playbackSliderPosition,
  } from "$lib/utils/playbackSeek.svelte";
  import { Progress } from "$lib/components/ui/progress";
  import { TTSPhase, TTSSource, type TTSStatus } from "$lib/state";

  let {
    embedded = false,
    onSeek,
    seeking = false,
    status,
    onPause,
    onResume,
    onRestart,
    onStop,
    onSave,
    onClear,
    onOpenSettings,
  }: {
    embedded?: boolean;
    onSeek?: (request: SeekRequest) => Promise<void>;
    seeking?: boolean;
    status: TTSStatus;
    onPause: () => void;
    onResume: () => void;
    onRestart: () => void;
    onStop: () => void;
    onSave: () => void;
    onClear: () => void;
    onOpenSettings?: () => void;
  } = $props();

  const seek = new PlaybackSeek();
  const position = $derived(seek.position(status));
  const steps = $derived(playbackSteps(status.durationMilliseconds));
  const sliderPosition = $derived(playbackSliderPosition(position, status.durationMilliseconds));
  const generating = $derived(status.phase === TTSPhase.Generating);
  const failed = $derived(status.phase === TTSPhase.Failed);
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
    if (status.phase === TTSPhase.Generating) return "Creating audio";
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

{#snippet timeline()}
  {#if status.canSeek && onSeek}
    {#key status.generation}
      <Slider
        type="single"
        value={sliderPosition}
        min={0}
        max={status.durationMilliseconds}
        step={steps}
        disabled={seeking || seek.pending}
        class="h-4 [&_[data-slot=slider-thumb]]:size-3 [&_[data-slot=slider-thumb]]:border-2 [&_[data-slot=slider-thumb]]:border-layer-fill [&_[data-slot=slider-thumb]]:bg-primary [&_[data-slot=slider-track]]:h-1"
        aria-label="Playback position"
        aria-valuetext={`${formatTime(position)} of ${formatTime(status.durationMilliseconds)}`}
        onValueChange={(value) => seek.change(status, value)}
        onValueCommit={() => void seek.commit(status, onSeek)}
      />
    {/key}
  {:else}
    <div class="flex h-4 items-center">
      <Progress value={percent} max={100} class="h-1" aria-label="Speech playback progress" />
    </div>
  {/if}
{/snippet}

<div
  class="shrink-0 border-hairline bg-layer-fill px-3 py-2"
  class:rounded-lg={!embedded}
  class:border={!embedded}
  aria-label="Speech playback"
>
  {#if !embedded}
    <div class="flex h-7 items-center gap-2" title={`${label} - ${phaseLabel}`}>
      <span class="sr-only" role="status">{label} - {phaseLabel}</span>
      {#if generating}
        <LoaderCircleIcon
          class="size-4 shrink-0 animate-spin text-muted-foreground motion-reduce:animate-none"
          aria-hidden="true"
        />
      {:else if status.canPause}
        <TooltipButton size="icon-sm" label="Pause speech playback" onclick={onPause}
          ><PauseIcon /></TooltipButton
        >
      {:else if status.canResume}
        <TooltipButton size="icon-sm" label="Resume speech playback" onclick={onResume}
          ><PlayIcon /></TooltipButton
        >
      {:else if status.canRestart}
        <TooltipButton size="icon-sm" label="Restart speech playback" onclick={onRestart}
          ><PlayIcon /></TooltipButton
        >
      {/if}
      <div class="min-w-0 flex-1">
        {#if generating}
          <p class="truncate text-xs text-muted-foreground">Creating audio...</p>
        {:else if failed}
          <p class="truncate text-xs text-destructive" role="alert">
            {status.message || "Speech could not be generated or played."}
          </p>
        {:else}
          {@render timeline()}
        {/if}
      </div>
      {#if failed}
        <FeedbackDetails
          title="Speech could not be completed"
          label="Speech error details"
          message={status.message || "Speech could not be generated or played."}
          actionLabel="Speech settings"
          onAction={onOpenSettings}
        />
      {:else if !generating}
        <span class="shrink-0 font-mono text-xs text-muted-foreground tabular-nums"
          >{formatTime(position)} / {formatTime(status.durationMilliseconds)}</span
        >
      {/if}
      {#if status.canStop}
        <TooltipButton size="icon-sm" label="Stop and release speech playback" onclick={onStop}
          ><SquareIcon /></TooltipButton
        >
      {/if}
      {#if status.canRestart || status.canSave || (status.canClear && !status.canStop)}
        <Menu.Root>
          <Menu.Trigger
            aria-label="Playback actions"
            class={buttonVariants({ variant: "ghost", size: "icon-sm" })}
            ><EllipsisIcon /></Menu.Trigger
          >
          <Menu.Content align="end" class="w-60 max-w-[calc(100vw-24px)]">
            {#if status.canRestart}<Menu.Item onclick={onRestart}
                ><RotateCcwIcon />Restart speech playback</Menu.Item
              >{/if}
            {#if status.canSave}<Menu.Item onclick={onSave}
                ><DownloadIcon />Save generated speech</Menu.Item
              >{/if}
            {#if status.canClear && !status.canStop}
              {#if status.canRestart || status.canSave}<Menu.Separator />{/if}
              <Menu.Item onclick={onClear}><Trash2Icon />Clear generated speech</Menu.Item>
            {/if}
          </Menu.Content>
        </Menu.Root>
      {/if}
    </div>
  {:else}
    <div class="flex items-center gap-2.5">
      <span
        class="grid size-5 shrink-0 place-items-center text-muted-foreground"
        aria-hidden="true"
      >
        {#if status.phase === TTSPhase.Generating}
          <LoaderCircleIcon class="size-4 animate-spin motion-reduce:animate-none" />
        {:else}
          <Volume2Icon class="size-4" />
        {/if}
      </span>
      <div class="min-w-0 flex-1">
        <div class="flex items-center justify-between gap-3 text-xs">
          <span class="truncate text-sm font-medium" role="status"
            >{label}<span class="text-xs font-normal text-muted-foreground"
              >&nbsp;· {phaseLabel}</span
            ></span
          >
          {#if failed}
            <FeedbackDetails
              title="Speech could not be completed"
              label="Speech error details"
              message={status.message || "Speech could not be generated or played."}
              actionLabel="Speech settings"
              onAction={onOpenSettings}
            />
          {:else if !generating}
            <span class="shrink-0 font-mono text-muted-foreground tabular-nums"
              >{formatTime(position)} / {formatTime(status.durationMilliseconds)}</span
            >
          {/if}
        </div>
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
        {#if status.canRestart}<TooltipButton
            variant="ghost"
            size="icon-sm"
            disabled={!status.canRestart}
            label="Restart speech playback"
            onclick={onRestart}><RotateCcwIcon /></TooltipButton
          >{/if}
        {#if status.canSave}
          <TooltipButton
            variant="ghost"
            size="icon-sm"
            label="Save generated speech"
            onclick={onSave}><DownloadIcon /></TooltipButton
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
    <div class="mt-1">
      {#if failed}
        <p class="truncate text-xs text-destructive" role="alert">
          {status.message || "Speech could not be generated or played."}
        </p>
      {:else if generating}
        <p class="h-4 text-xs text-muted-foreground">Audio will play when ready.</p>
      {:else}
        {@render timeline()}
      {/if}
    </div>
  {/if}
</div>
