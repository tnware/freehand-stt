<script lang="ts">
  import CheckIcon from "@lucide/svelte/icons/check";
  import FileAudioIcon from "@lucide/svelte/icons/file-audio";
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RotateCcwIcon from "@lucide/svelte/icons/rotate-ccw";
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
  }: {
    status: FileTranscriptionStatus;
    choosing?: boolean;
    voiceActive?: boolean;
    onChoose: () => void;
    onStart: (stream: boolean) => void;
    onTryStreamingAgain: () => void;
    onCancel: () => void;
    onClear: () => void;
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
          : "The completed result is available in History.";
      case FileTranscriptionPhase.FileTranscriptionFailed:
        return status.message || "The file could not be transcribed.";
      default:
        return "";
    }
  });

  const fileDescription = $derived.by(() => {
    if (!hasFile) return "FLAC, MP3, MP4, M4A, OGG, WAV, or WebM";
    if (working) return "Follow the live result in History";
    if (completed)
      return status.transcript ? "Transcript retained in History" : "No speech detected";
    if (failed) return "Ready to retry or choose another file";
    return stream && !status.streamingUnavailable
      ? "Transcript will appear progressively"
      : "Wait for one completed result";
  });

  const footerStatus = $derived.by(() => {
    switch (status.phase) {
      case FileTranscriptionPhase.FileTranscriptionUploading:
        return `${uploadPercent}% sent`;
      case FileTranscriptionPhase.FileTranscriptionProcessing:
        return "audio sent ✓";
      case FileTranscriptionPhase.FileTranscriptionStreaming:
        return "history updates live";
      case FileTranscriptionPhase.FileTranscriptionCancelling:
        return "discarding";
      case FileTranscriptionPhase.FileTranscriptionCompleted:
        return "history updated ✓";
      case FileTranscriptionPhase.FileTranscriptionFailed:
        return "nothing added";
      case FileTranscriptionPhase.FileTranscriptionSelected:
        return "file ready";
      default:
        return "stored locally";
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
      phaseAnnouncement = "Receiving transcript.";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionCancelling) {
      phaseAnnouncement = "Cancelling audio file transcription.";
    } else if (phase === FileTranscriptionPhase.FileTranscriptionCompleted) {
      phaseAnnouncement = status.transcript
        ? "Audio file transcription complete. History updated."
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

<TransportShell {rail} {railPercent} busy={working} state={status.phase}>
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
      {#if working}
        <LoaderCircleIcon class="size-[24px] animate-spin motion-reduce:animate-none" />
      {:else if completed}
        <CheckIcon class="size-[24px]" />
      {:else if hasFile}
        <FileTextIcon class="size-[24px]" />
      {:else}
        <FileAudioIcon class="size-[24px]" />
      {/if}
    </span>
  {/snippet}

  {#snippet stage()}
    <div class="flex min-w-0 items-center gap-2.5">
      <span
        class="min-w-0 truncate text-[13.5px] font-semibold"
        title={status.fileName || undefined}
      >
        {status.fileName || "Choose an audio recording"}
      </span>
      {#if hasFile && fileSize}
        <span class="figure shrink-0 text-[10px] text-ink-quiet">{formatBytes(fileSize)}</span>
      {/if}
    </div>

    <p
      class={cn(
        "mt-1.5 min-h-4 text-[11.5px]",
        failed ? "text-destructive" : "text-secondary-foreground",
      )}
    >
      {phaseLabel}
    </p>

    <div
      class="mt-2 flex min-h-5 items-center justify-between gap-3 border-t border-hairline pt-1.5"
    >
      {#if !hasFile}
        <span class="figure min-w-0 truncate text-[10.5px] text-muted-foreground">
          FLAC, MP3, MP4, M4A, OGG, WAV, or WebM
        </span>
      {:else if !working && !completed && !failed}
        <label
          class="flex min-w-0 items-center gap-2.5 text-[11.5px] text-secondary-foreground"
          for="file-stream-toggle"
        >
          <Switch
            id="file-stream-toggle"
            size="sm"
            checked={stream && !status.streamingUnavailable}
            disabled={status.streamingUnavailable}
            onCheckedChange={(next) => (stream = next)}
          />
          Stream the transcript as it arrives
        </label>
      {:else}
        <span class="figure min-w-0 truncate text-[10.5px] text-muted-foreground">
          {fileDescription}
        </span>
      {/if}
      <span class="figure shrink-0 text-[9.5px] text-ink-quiet">{footerStatus}</span>
    </div>
  {/snippet}

  {#snippet readout()}
    <span
      class={cn(
        "figure text-[34px] leading-none font-medium tracking-[-0.02em]",
        working ? "text-foreground" : "text-ink-disabled",
      )}
    >
      {#if uploading}
        {uploadPercent}<span class="text-[20px] text-muted-foreground">%</span>
      {:else if hasFile}
        {formatBytes(fileSize)}
      {:else}
        0 <span class="text-[20px] text-muted-foreground">MB</span>
      {/if}
    </span>

    <div class="flex min-h-5 items-center gap-2">
      <span
        class={cn(
          "size-[7px] rounded-full",
          failed
            ? "bg-destructive"
            : completed
              ? "bg-success"
              : working
                ? "bg-primary shadow-[0_0_8px_var(--primary)]"
                : hasFile
                  ? "bg-primary"
                  : "bg-border",
        )}
      ></span>
      <span class="caption text-secondary-foreground">{stateLabel}</span>
    </div>

    <div class="flex min-h-[26px] items-center gap-1.5">
      {#if completed}
        <Button
          variant="outline"
          size="sm"
          class="h-[26px] flex-1 px-2.5 text-[11.5px]"
          disabled={!status.canStart || voiceActive}
          onclick={() => onStart(stream)}
        >
          <RotateCcwIcon class="size-3" />
          Again
        </Button>
        <Button
          variant="outline"
          size="sm"
          class="h-[26px] flex-1 px-2.5 text-[11.5px]"
          onclick={onClear}
        >
          <XIcon class="size-3" />
          Clear
        </Button>
      {:else if working}
        <Button
          variant="outline"
          size="sm"
          class="h-[26px] w-full px-2.5 text-[11.5px]"
          disabled={!status.canCancel}
          onclick={onCancel}
        >
          {#if status.canCancel}
            <XIcon class="size-3" />
          {:else}
            <LoaderCircleIcon class="size-3 animate-spin motion-reduce:animate-none" />
          {/if}
          Cancel
        </Button>
      {:else if failed}
        <Button
          variant="outline"
          size="sm"
          class="h-[26px] flex-1 px-2.5 text-[11.5px]"
          disabled={!status.canStart || voiceActive}
          onclick={() => onStart(stream)}
        >
          <RotateCcwIcon class="size-3" />
          Retry
        </Button>
        {#if status.streamingUnavailable && !status.streamingProfileUnavailable}
          <Button
            variant="outline"
            size="sm"
            class="h-[26px] flex-1 px-2.5 text-[11.5px]"
            onclick={onTryStreamingAgain}
          >
            Stream
          </Button>
        {:else}
          <Button
            variant="outline"
            size="sm"
            class="h-[26px] flex-1 px-2.5 text-[11.5px]"
            onclick={onClear}
          >
            Clear
          </Button>
        {/if}
      {:else if hasFile}
        <Button
          size="sm"
          class="h-[26px] flex-1 px-2.5 text-[11.5px]"
          disabled={!status.canStart || voiceActive}
          onclick={() => onStart(stream)}
        >
          Transcribe
        </Button>
        <Button
          variant="outline"
          size="sm"
          class="h-[26px] shrink-0 px-2 text-[11.5px]"
          aria-label="Clear selected audio file"
          onclick={onClear}
        >
          <XIcon class="size-3" />
        </Button>
      {:else}
        <Button
          size="sm"
          class="h-[26px] w-full px-2.5 text-[11.5px]"
          disabled={choosing || voiceActive}
          onclick={onChoose}
        >
          {#if choosing}
            <LoaderCircleIcon class="size-3 animate-spin motion-reduce:animate-none" />
          {:else}
            <FolderOpenIcon class="size-3" />
          {/if}
          Choose audio
        </Button>
      {/if}
    </div>

    <span class="sr-only" role="status" aria-live="polite" aria-atomic="true">
      {phaseAnnouncement}
    </span>
  {/snippet}
</TransportShell>
