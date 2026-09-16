<script lang="ts">
  import ButtonIcon from "$lib/components/ui/button/ButtonIcon.svelte";
  import PlayIcon from "@lucide/svelte/icons/play";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";

  import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
  import FileAudioIcon from "@lucide/svelte/icons/file-audio";
  import XIcon from "@lucide/svelte/icons/x";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import { Button } from "$lib/components/ui/button";
  import FeedbackDetails from "$lib/components/common/FeedbackDetails.svelte";
  import WorkflowSettingsButton from "./WorkflowSettingsButton.svelte";
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
    optionsDisabled = false,
    showSettings = true,
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
    optionsDisabled?: boolean;
    showSettings?: boolean;
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
  File selection and request actions remain together above the transcript.
-->
<section
  class="@container flex flex-col gap-2 border-b border-hairline bg-layer-fill px-3 py-2"
  aria-label="Audio file"
  data-state={status.phase}
>
  <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
    <div class="flex min-w-0 basis-56 flex-[2] flex-wrap items-center gap-2">
      {#if hasFile}
        <FileAudioIcon class="content-section-icon" aria-hidden="true" />
        <span class="min-w-0 basis-24 flex-1">
          <span class="content-value block truncate" title={status.fileName}
            >{status.fileName || "Audio file"}</span
          >
          <span class="content-meta block truncate tabular-nums">
            {formatBytes(size) || "size unknown"}{uploading
              ? ` · ${percent}% sent`
              : ""}
          </span>
        </span>
        {#if !working}
          <TooltipButton
            label="Clear selected audio file"
            onclick={onClear}
            aria-busy={clearing}
            disabled={busy}
            ><ButtonIcon icon={XIcon} busy={clearing} /></TooltipButton
          >
        {/if}
      {:else}
        <Button
          size="sm"
          onclick={onChoose}
          aria-busy={choosing}
          disabled={busy || !!blocked}
          title={blocked || "Choose an audio file"}
        >
          <ButtonIcon icon={FolderOpenIcon} busy={choosing} />
          Choose audio file
        </Button>
        <span class="content-meta">{blocked || "no microphone required"}</span>
      {/if}
      {#if hasFile}
        <Button
          variant="outline"
          size="sm"
          onclick={onChoose}
          aria-busy={choosing}
          disabled={busy || working || !!blocked}
          aria-label="Change audio file"
          ><ButtonIcon icon={FolderOpenIcon} busy={choosing} />Change</Button
        >
      {/if}
    </div>

    <div
      class="ml-auto flex max-w-full flex-wrap items-center justify-end gap-2"
    >
      {#if showSettings}<WorkflowSettingsButton
          label="Audio file settings"
          disabled={optionsDisabled}
          onclick={onOpenSettings}
        />{/if}
      {#if status.streamingUnavailable && !status.streamingProfileUnavailable && !working}
        <Button
          variant="outline"
          size="sm"
          onclick={onTryStreamingAgain}
          aria-busy={resettingStreaming}
          disabled={busy || !!blocked}
          ><ButtonIcon icon={RotateCcwIcon} busy={resettingStreaming} />Try
          streaming</Button
        >
      {/if}
      {#if working}
        <Button
          size="sm"
          variant="outline"
          onclick={onCancel}
          aria-busy={cancelling ||
            status.phase === FileTranscriptionPhase.FileTranscriptionCancelling}
          disabled={cancelling || !status.canCancel}
        >
          <ButtonIcon
            icon={XIcon}
            busy={cancelling ||
              status.phase ===
                FileTranscriptionPhase.FileTranscriptionCancelling}
          />
          {cancelling ||
          status.phase === FileTranscriptionPhase.FileTranscriptionCancelling
            ? "Cancelling…"
            : "Cancel"}
        </Button>
      {:else if hasFile}
        <Button
          size="sm"
          onclick={onStart}
          aria-busy={starting}
          disabled={busy || !status.canStart || !!blocked}
        >
          <ButtonIcon
            icon={failed || completed ? RotateCcwIcon : PlayIcon}
            busy={starting}
          />
          {starting
            ? "Starting…"
            : failed
              ? "Retry"
              : completed
                ? "Transcribe again"
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
    <p class="content-meta" role="status">
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
    <p class="content-meta">
      {status.streamingNotice}
    </p>
  {/if}
</section>
