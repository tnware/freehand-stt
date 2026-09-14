<script lang="ts">
  import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import XIcon from "@lucide/svelte/icons/x";
  import { Button } from "$lib/components/ui/button";
  import Stage, { type StageTone } from "./Stage.svelte";
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
    onChoose,
    onStart,
    onCancel,
    onClear,
  }: {
    status: FileTranscriptionStatus;
    choosing?: boolean;
    starting?: boolean;
    cancelling?: boolean;
    clearing?: boolean;
    /** Why a file cannot be chosen right now, empty when it can. */
    blocked?: string;
    onChoose: () => void;
    onStart: () => void;
    onCancel: () => void;
    onClear: () => void;
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
  const failed = $derived(
    status.phase === FileTranscriptionPhase.FileTranscriptionFailed,
  );
  const busy = $derived(choosing || starting || clearing);
  const tone = $derived<StageTone>(
    failed ? "bad" : working ? "busy" : hasFile ? "ok" : "idle",
  );

  const uploaded = $derived(status.bytesUploaded ?? 0);
  const size = $derived(status.fileSize ?? 0);
  const percent = $derived(
    size > 0 ? Math.min(100, Math.round((uploaded / size) * 100)) : 0,
  );

  function formatBytes(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
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

<!-- The same position as the microphone in the voice chain: this is where the
     audio comes from. Only the source differs, which is the point. -->
<Stage ordinal="01" label="Source" {tone}>
  {#if hasFile}
    <div class="flex items-start gap-2">
      <span class="min-w-0 flex-1">
        <span class="block truncate text-[13px] font-medium" title={status.fileName}
          >{status.fileName || "Audio file"}</span
        >
        <span class="block truncate font-mono text-[10px] text-ink-quiet">
          {size > 0 ? formatBytes(size) : "size unknown"}{uploading
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
          aria-label="Remove this file"
          title="Remove this file"
        >
          <XIcon class="size-3" aria-hidden="true" />
        </button>
      {/if}
    </div>

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
  {:else}
    <Button
      variant="outline"
      size="sm"
      class="w-full justify-start"
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
    <p class="px-1 font-mono text-[10px] text-ink-quiet">
      {blocked || "no microphone required"}
    </p>
  {/if}

  {#snippet footer()}
    {#if working}
      <Button size="xs" variant="outline" onclick={onCancel} disabled={cancelling}>
        {cancelling ? "Cancelling" : "Cancel"}
      </Button>
    {:else if hasFile && status.canStart}
      <Button size="xs" onclick={onStart} disabled={starting}>
        {starting ? "Starting" : "Transcribe"}
      </Button>
    {:else}
      <span></span>
    {/if}
  {/snippet}
</Stage>
