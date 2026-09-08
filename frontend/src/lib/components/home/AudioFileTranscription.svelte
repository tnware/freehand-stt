<script lang="ts">
  import FeedbackDetails from "$lib/components/common/FeedbackDetails.svelte";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import CheckIcon from "@lucide/svelte/icons/check";
  import FileAudioIcon from "@lucide/svelte/icons/file-audio";
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import XIcon from "@lucide/svelte/icons/x";
  import TransportShell from "$lib/components/home/TransportShell.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import { FileTranscriptionPhase, type FileTranscriptionStatus } from "$lib/state";
  import { cn } from "$lib/utils";

  let {
    status,
    choosing = false,
    voiceActive = false,
    onChoose,
    onStart,
    onTryStreamingAgain,
    onCancel,
    onClear,
    onOpenSettings,
  }: {
    status: FileTranscriptionStatus;
    choosing?: boolean;
    voiceActive?: boolean;
    onChoose: () => void;
    onStart: (stream: boolean) => void;
    onTryStreamingAgain: () => void;
    onCancel: () => void;
    onClear: () => void;
    onOpenSettings?: () => void;
  } = $props();

  let stream = $state(true);
  $effect(() => {
    if (status.streamingUnavailable) stream = false;
  });

  const hasFile = $derived(status.phase !== FileTranscriptionPhase.FileTranscriptionEmpty);
  const uploading = $derived(status.phase === FileTranscriptionPhase.FileTranscriptionUploading);
  const working = $derived(
    uploading ||
      status.phase === FileTranscriptionPhase.FileTranscriptionProcessing ||
      status.phase === FileTranscriptionPhase.FileTranscriptionStreaming ||
      status.phase === FileTranscriptionPhase.FileTranscriptionCancelling,
  );
  const completed = $derived(status.phase === FileTranscriptionPhase.FileTranscriptionCompleted);
  const failed = $derived(status.phase === FileTranscriptionPhase.FileTranscriptionFailed);
  const uploaded = $derived(status.bytesUploaded ?? 0);
  const fileSize = $derived(status.fileSize ?? 0);
  const uploadPercent = $derived(
    fileSize > 0 ? Math.min(100, Math.round((uploaded / fileSize) * 100)) : 0,
  );

  const formatBytes = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    const units = ["KB", "MB", "GB"];
    let value = bytes / 1024;
    let unit = units[0];
    for (let index = 1; index < units.length && value >= 1024; index++) {
      value /= 1024;
      unit = units[index];
    }
    return `${value >= 10 ? value.toFixed(0) : value.toFixed(1)} ${unit}`;
  };

  const stateLabel = $derived.by(() => {
    switch (status.phase) {
      case FileTranscriptionPhase.FileTranscriptionSelected:
        return "Selected";
      case FileTranscriptionPhase.FileTranscriptionUploading:
        return "Uploading";
      case FileTranscriptionPhase.FileTranscriptionProcessing:
        return "Transcribing";
      case FileTranscriptionPhase.FileTranscriptionStreaming:
        return "Receiving";
      case FileTranscriptionPhase.FileTranscriptionCancelling:
        return "Cancelling";
      case FileTranscriptionPhase.FileTranscriptionCompleted:
        return "Complete";
      case FileTranscriptionPhase.FileTranscriptionFailed:
        return "Failed";
      default:
        return "Ready";
    }
  });

  const phaseLabel = $derived.by(() => {
    switch (status.phase) {
      case FileTranscriptionPhase.FileTranscriptionUploading:
        return `Uploading audio · ${uploadPercent}%`;
      case FileTranscriptionPhase.FileTranscriptionProcessing:
        return status.message || "Waiting for the completed transcript…";
      case FileTranscriptionPhase.FileTranscriptionStreaming:
        return status.transcript ? "Receiving transcript…" : "Waiting for transcript…";
      case FileTranscriptionPhase.FileTranscriptionCancelling:
        return "Discarding this upload…";
      case FileTranscriptionPhase.FileTranscriptionCompleted:
        return status.buffered
          ? "The server returned the stream as one completed result."
          : "The current result is ready to inspect and copy.";
      case FileTranscriptionPhase.FileTranscriptionFailed:
        return status.message || "The file could not be transcribed.";
      default:
        return hasFile ? "Ready to transcribe" : "FLAC, MP3, MP4, M4A, OGG, WAV, or WebM";
    }
  });

  const rail = $derived(failed ? "error" : completed ? "done" : working ? "working" : "hidden");
  // Only the upload leg has a known length. Everything after it waits on the
  // endpoint, which reports no progress, so the rail stops claiming a share.
  const railPercent = $derived(uploading ? uploadPercent : undefined);

  let phaseAnnouncement = $state("");
  let previousPhase: FileTranscriptionPhase | undefined;
  let previousGeneration = -1;
  $effect(() => {
    const phase = status.phase;
    const generation = status.generation;
    if (previousPhase === undefined) {
      previousPhase = phase;
      previousGeneration = generation;
      return;
    }
    if (phase === previousPhase && generation === previousGeneration) return;

    if (phase === FileTranscriptionPhase.FileTranscriptionSelected) {
      phaseAnnouncement = `${status.fileName || "Audio file"} selected.`;
    } else if (phase === FileTranscriptionPhase.FileTranscriptionUploading) {
      phaseAnnouncement = "Uploading audio file.";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionProcessing) {
      phaseAnnouncement = "Upload complete. Transcribing audio file.";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionStreaming) {
      phaseAnnouncement = "Receiving transcript…";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionCancelling) {
      phaseAnnouncement = "Cancelling audio file transcription.";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionCompleted) {
      phaseAnnouncement = status.transcript
        ? "Audio file transcription complete. Result ready to copy."
        : "Audio file transcription complete. No speech detected.";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionFailed) {
      phaseAnnouncement = status.message || "Audio file transcription failed.";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionEmpty && previousPhase !== phase) {
      phaseAnnouncement = "Audio file cleared.";
    }

    previousPhase = phase;
    previousGeneration = generation;
  });
</script>

<TransportShell {rail} {railPercent} busy={working} state={status.phase} stageGrid={false}>
  {#snippet control()}
    <span
      class={cn(
        "grid size-[62px] place-items-center rounded-full border",
        failed
          ? "border-destructive/30 bg-destructive/10 text-destructive"
          : hasFile
            ? "border-accent-edge bg-accent-wash text-accent-text"
            : "border-hairline bg-control-fill text-muted-foreground",
      )}
      aria-hidden="true"
    >
      {#if working}<LoaderCircleIcon class="size-6 animate-spin motion-reduce:animate-none" />
      {:else if completed}<CheckIcon class="size-6" />
      {:else if hasFile}<FileTextIcon class="size-6" />
      {:else}<FileAudioIcon class="size-6" />{/if}
    </span>
  {/snippet}
  {#snippet stage()}
    <div class="flex h-8 min-w-0 items-center gap-2">
      <span
        class="min-w-0 flex-1 truncate text-sm font-semibold"
        title={status.fileName || undefined}
      >
        {status.fileName || "Choose an audio file"}
      </span>
      {#if hasFile}<span class="shrink-0 text-xs tabular-nums text-muted-foreground"
          >{formatBytes(fileSize)}</span
        >{/if}
      <TooltipButton
        variant="ghost"
        size="icon-xs"
        label="Clear selected audio file"
        disabled={!hasFile || working || choosing}
        onclick={onClear}><XIcon /></TooltipButton
      >
    </div>
    {#if failed}
      <div class="flex h-5 min-w-0 items-center justify-between gap-2">
        <span class="truncate text-xs text-destructive">Transcription failed</span>
        <FeedbackDetails
          title="Transcription failed"
          label="File transcription error details"
          message={phaseLabel}
          actionLabel="Transcription settings"
          onAction={onOpenSettings}
        />
      </div>
    {:else}
      <p class="min-h-5 truncate text-xs text-muted-foreground" title={phaseLabel}>{phaseLabel}</p>
    {/if}
    <div class="mt-2 flex h-6 min-w-0 items-center gap-2 border-t border-hairline pt-2">
      <Switch
        id="file-stream-toggle"
        size="sm"
        checked={stream && !status.streamingUnavailable}
        disabled={working || status.streamingUnavailable}
        onCheckedChange={(next) => (stream = next)}
      />
      <label
        class="truncate text-xs text-secondary-foreground"
        for="file-stream-toggle"
        title={status.streamingUnavailable
          ? "This connection currently returns completed transcripts"
          : "Show transcript updates as the server sends them"}
      >
        {status.streamingUnavailable ? "Completed transcript" : "Show text as it arrives"}
      </label>
      {#if !working && status.streamingUnavailable && !status.streamingProfileUnavailable}
        <Button
          variant="ghost"
          size="sm"
          class="ml-auto h-6 px-1 text-xs"
          onclick={onTryStreamingAgain}>Try streaming</Button
        >
      {/if}
    </div>
  {/snippet}
  {#snippet readout()}
    <div class="flex min-w-0 items-center gap-2 text-sm">
      <span
        class={cn(
          "size-1.5 shrink-0 rounded-full",
          failed
            ? "bg-destructive"
            : completed
              ? "bg-success"
              : working
                ? "bg-primary"
                : "bg-border",
        )}
      ></span>
      <span class="font-medium">{stateLabel}</span>
      {#if uploading}<span class="ml-auto text-xs tabular-nums text-muted-foreground"
          >{uploadPercent}%</span
        >{/if}
    </div>
    <div class="flex shrink-0 items-center gap-2">
      <Button
        variant={hasFile ? "outline" : "default"}
        size="sm"
        class="h-8 flex-1 px-2.5"
        disabled={choosing || voiceActive || working}
        aria-label={hasFile ? "Change audio file" : "Choose audio"}
        onclick={onChoose}
      >
        {#if choosing}<LoaderCircleIcon
            class="size-3.5 animate-spin motion-reduce:animate-none"
          />{:else}<FolderOpenIcon class="size-3.5" />{/if}
        {hasFile ? "Change" : "Choose"}
      </Button>
      {#if working}
        <Button
          variant="outline"
          size="sm"
          class="h-8 min-w-24 flex-1 px-2.5"
          disabled={!status.canCancel}
          onclick={onCancel}
        >
          {status.phase === FileTranscriptionPhase.FileTranscriptionCancelling
            ? "Cancelling…"
            : "Cancel"}
        </Button>
      {:else}
        <Button
          size="sm"
          class="h-8 min-w-24 flex-1 px-2.5"
          disabled={!status.canStart || voiceActive || choosing}
          onclick={() => onStart(stream)}
        >
          {failed ? "Retry" : completed ? "Again" : "Transcribe"}
        </Button>
      {/if}
    </div>
    <span class="sr-only" role="status" aria-live="polite" aria-atomic="true"
      >{phaseAnnouncement}</span
    >
  {/snippet}
</TransportShell>
