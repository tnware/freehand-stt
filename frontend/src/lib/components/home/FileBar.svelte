<script lang="ts">
  import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import XIcon from "@lucide/svelte/icons/x";
  import { Button } from "$lib/components/ui/button";
  import FeedbackDetails from "$lib/components/common/FeedbackDetails.svelte";
  import {
    FileTranscriptionPhase,
    type FileTranscriptionStatus,
  } from "$lib/state";

  let {
    status,
    choosing = false,
    starting = false,
    cancelling = false,
    clearing = false,
    blocked = "",
    resettingStreaming = false,
    onChoose,
    onStart,
    onCancel,
    onClear,
    onTryStreamingAgain,
    onOpenSettings,
  }: {
    status: FileTranscriptionStatus;
    choosing?: boolean;
    starting?: boolean;
    cancelling?: boolean;
    clearing?: boolean;
    blocked?: string;
    resettingStreaming?: boolean;
    onChoose: () => void;
    onStart: () => void;
    onCancel: () => void;
    onClear: () => void;
    onTryStreamingAgain: () => void;
    onOpenSettings: () => void;
  } = $props();

  const hasFile = $derived(
    status.phase !== FileTranscriptionPhase.FileTranscriptionEmpty,
  );
  const uploading = $derived(
    status.phase === FileTranscriptionPhase.FileTranscriptionUploading,
  );
  const working = $derived(
    uploading ||
      status.phase === FileTranscriptionPhase.FileTranscriptionProcessing ||
      status.phase === FileTranscriptionPhase.FileTranscriptionStreaming ||
      status.phase === FileTranscriptionPhase.FileTranscriptionCancelling,
  );
  const busy = $derived(choosing || starting || clearing || resettingStreaming);
  const failed = $derived(
    status.phase === FileTranscriptionPhase.FileTranscriptionFailed,
  );
  const completed = $derived(
    status.phase === FileTranscriptionPhase.FileTranscriptionCompleted,
  );
  const uploaded = $derived(status.bytesUploaded ?? 0);
  const size = $derived(status.fileSize ?? 0);
  const percent = $derived(
    size > 0 ? Math.min(100, Math.round((uploaded / size) * 100)) : 0,
  );

  function formatBytes(bytes: number): string {
    if (!bytes) return "";
    const units = ["KB", "MB", "GB"];
    let value = bytes / 1024;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
      value /= 1024;
      unit += 1;
    }
    return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${units[unit]}`;
  }
</script>

<!--
  A file has no microphone, no live stream and no insertion target, so it does
  not need four stages standing on screen. One row states where the audio comes
  from, which model reads it, and what to press.
-->
<section
  class="flex flex-col gap-2 rounded-lg border border-hairline bg-card px-3 py-2.5"
  aria-label="Audio file"
  data-state={status.phase}
>
  <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
    <div class="flex min-w-[13rem] flex-[2] items-center gap-2.5">
      {#if hasFile}
        <span class="min-w-0 flex-1">
          <span
            class="block truncate text-[13px] font-medium"
            title={status.fileName}>{status.fileName || "Audio file"}</span
          >
          <span class="block truncate font-mono text-[10px] text-ink-quiet">
            {formatBytes(size) || "size unknown"}{uploading
              ? ` · ${percent}% sent`
              : ""}
          </span>
        </span>
        {#if !working}
          <button
            type="button"
            class="grid size-6 shrink-0 place-items-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-subtle-fill-hover hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:opacity-40"
            onclick={onClear}
            disabled={busy}
            aria-label="Clear selected audio file"
            title="Remove this file"><XIcon class="size-3" /></button
          >
        {/if}
      {:else}
        <Button
          variant="outline"
          size="sm"
          onclick={onChoose}
          disabled={busy || !!blocked}
          title={blocked || "Choose an audio file"}
        >
          {#if choosing}
            <LoaderCircleIcon class="size-3.5 animate-spin" />
          {:else}
            <FolderOpenIcon class="size-3.5" />
          {/if}
          Choose audio file
        </Button>
        <span class="font-mono text-[10px] text-ink-quiet"
          >{blocked || "no microphone required"}</span
        >
      {/if}
      {#if hasFile}
        <Button
          variant="outline"
          size="sm"
          onclick={onChoose}
          disabled={busy || working || !!blocked}
          aria-label="Change audio file"
          ><FolderOpenIcon class="size-3.5" />Change</Button
        >
      {/if}
    </div>

    <div class="flex shrink-0 items-center gap-2">
      {#if status.streamingUnavailable && !status.streamingProfileUnavailable && !working}
        <Button
          variant="outline"
          size="xs"
          onclick={onTryStreamingAgain}
          disabled={busy || !!blocked}>Try streaming</Button
        >
      {/if}
      {#if working}
        <Button
          size="sm"
          variant="outline"
          onclick={onCancel}
          disabled={cancelling || !status.canCancel}
        >
          {cancelling ||
          status.phase === FileTranscriptionPhase.FileTranscriptionCancelling
            ? "Cancelling…"
            : "Cancel"}
        </Button>
      {:else if hasFile}
        <Button
          size="sm"
          onclick={onStart}
          disabled={busy || !status.canStart || !!blocked}
        >
          {starting
            ? "Starting…"
            : failed
              ? "Retry"
              : completed
                ? "Again"
                : "Transcribe"}
        </Button>
      {/if}
    </div>
  </div>

  {#if failed}
    <div
      class="flex items-center justify-between gap-2 text-xs text-destructive"
      role="status"
    >
      <span>Transcription failed</span><FeedbackDetails
        title="Transcription failed"
        label="File transcription error details"
        message={status.message || "The endpoint did not return a transcript."}
        actionLabel="Transcription settings"
        onAction={onOpenSettings}
      />
    </div>
  {:else if status.message || blocked}
    <p class="text-xs text-muted-foreground" role="status">
      {blocked || status.message}
    </p>
  {/if}

  {#if uploading}
    <div
      class="h-1 overflow-hidden rounded-full bg-meter-rest"
      role="progressbar"
      aria-valuenow={percent}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-label="Upload progress"
    >
      <span
        class="block h-full rounded-full bg-meter transition-[width]"
        style:width={`${percent}%`}
      ></span>
    </div>
  {/if}

  {#if status.streamingNotice}
    <p class="text-[11px] leading-snug text-muted-foreground">
      {status.streamingNotice}
    </p>
  {/if}
</section>
