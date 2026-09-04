<script lang="ts">
  import HistoryIcon from "@lucide/svelte/icons/history";
  import DisclosureHeader from "$lib/components/shell/DisclosureHeader.svelte";
  import HistoryList from "$lib/components/history/HistoryList.svelte";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import { Button } from "$lib/components/ui/button";
  import {
    FileTranscriptionPhase,
    type FileTranscriptionStatus,
    type HistoryEntry,
    type HistoryTextVersion,
    TTSPhase,
    type TTSStatus,
  } from "$lib/state";
  import {
    readDisclosurePreference,
    writeDisclosurePreference,
  } from "$lib/utils/viewPreferences";

  let {
    enabled = false,
    entries,
    fileStatus,
    fileHistoryGeneration,
    onOpenSettings,
    onCopy,
    onCopyVersion,
    onDelete,
    onCopyFile,
    ttsEnabled = false,
    ttsAvailable = true,
    ttsStatus,
    onListen,
    onListenFile,
    onPauseTTS,
    onResumeTTS,
    onRestartTTS,
    onStopTTS,
    onSaveTTS,
    onClearTTS,
    ttsWorkspaceVisible = false,
    collapsible = true,
  }: {
    enabled?: boolean;
    entries: HistoryEntry[];
    fileStatus: FileTranscriptionStatus;
    fileHistoryGeneration: number;
    onOpenSettings: () => void;
    onCopy: (id: number) => Promise<boolean>;
    onCopyVersion: (id: number, version: HistoryTextVersion) => Promise<boolean>;
    onDelete: (id: number) => Promise<boolean>;
    onCopyFile: () => Promise<boolean>;
    ttsEnabled?: boolean;
    ttsAvailable?: boolean;
    ttsStatus: TTSStatus;
    onListen: (id: number, version: HistoryTextVersion) => void;
    onListenFile: () => void;
    onPauseTTS: () => void;
    onResumeTTS: () => void;
    onRestartTTS: () => void;
    onStopTTS: () => void;
    onSaveTTS: () => void;
    onClearTTS: () => void;
    ttsWorkspaceVisible?: boolean;
    /**
     * False when the panel already fills a pane of its own. Its header would
     * then only repeat the pane's name and offer a collapse that hides the
     * whole view, so the panel drops the header and stays open.
     */
    collapsible?: boolean;
  } = $props();

  const fileWorking = $derived(
    fileStatus.phase === FileTranscriptionPhase.FileTranscriptionUploading ||
      fileStatus.phase === FileTranscriptionPhase.FileTranscriptionProcessing ||
      fileStatus.phase === FileTranscriptionPhase.FileTranscriptionStreaming ||
      fileStatus.phase === FileTranscriptionPhase.FileTranscriptionCancelling,
  );
  const fileFinished = $derived(
    fileStatus.phase === FileTranscriptionPhase.FileTranscriptionCompleted ||
      fileStatus.phase === FileTranscriptionPhase.FileTranscriptionFailed,
  );
  const fileResultRetained = $derived(
    enabled &&
      fileStatus.phase === FileTranscriptionPhase.FileTranscriptionCompleted &&
      fileHistoryGeneration === fileStatus.generation,
  );
  const live = $derived.by(() => {
    if ((!fileWorking && !fileFinished) || fileResultRetained) return undefined;
    if (
      fileStatus.phase === FileTranscriptionPhase.FileTranscriptionFailed &&
      !fileStatus.transcript
    ) {
      return undefined;
    }
    let status = "preparing";
    if (fileStatus.phase === FileTranscriptionPhase.FileTranscriptionUploading) {
      status = "uploading";
    } else if (fileStatus.phase === FileTranscriptionPhase.FileTranscriptionStreaming) {
      status = "streaming";
    } else if (fileStatus.phase === FileTranscriptionPhase.FileTranscriptionProcessing) {
      status = "processing";
    } else if (fileStatus.phase === FileTranscriptionPhase.FileTranscriptionCancelling) {
      status = "cancelling";
    } else if (fileStatus.phase === FileTranscriptionPhase.FileTranscriptionFailed) {
      status = "partial result";
    } else if (fileStatus.phase === FileTranscriptionPhase.FileTranscriptionCompleted) {
      status = "audio file";
    }
    const text = fileStatus.transcript ?? "";
    return {
      text,
      fileName: fileStatus.fileName ?? "Audio file",
      status,
      characterCount: Array.from(text).length,
      working: fileWorking,
      canCopy: !fileWorking && fileStatus.canCopy,
      failed: fileStatus.phase === FileTranscriptionPhase.FileTranscriptionFailed,
    };
  });

  // The count stays on the header in both states: unlike Quick Settings,
  // the body does not repeat it, and it is the one fact worth having while
  // the list is folded away.
  const summary = $derived.by(() => {
    if (live?.working) return enabled ? `live · ${entries.length} kept` : "live · not retained";
    if (live) return enabled ? `result · ${entries.length} kept` : "result · not retained";
    if (!enabled) return "off";
    return entries.length === 1 ? "1 kept · in memory" : `${entries.length} kept · in memory`;
  });
  const showPlayback = $derived(
    ttsStatus.phase !== TTSPhase.Idle &&
      ttsStatus.phase !== TTSPhase.Cancelled &&
      !(ttsWorkspaceVisible && ttsStatus.source === "compose"),
  );

  let stored = $state(readDisclosurePreference("history", true));
  const open = $derived(collapsible ? stored : true);

  function toggleOpen() {
    stored = !stored;
    writeDisclosurePreference("history", stored);
  }
</script>

<!--
  The list is the tallest thing on this screen, so it collapses the way the
  Quick Settings does. It fills its column rather than sizing to its content,
  which is why the height comes from flex-grow — that interpolates, where
  swapping a flex child in and out of the layout would jump.
-->
<section
  class="history-card rounded-lg border border-card-stroke bg-card shadow-lift"
  class:open
  class:bare={!collapsible}
  aria-label="Transcript history"
>
  {#if collapsible}
    <DisclosureHeader
      label="Transcripts"
      {summary}
      {open}
      controls="history-detail"
      onToggle={toggleOpen}
    />
  {/if}

  <div class="drawer" id="history-detail" inert={!open}>
    {#if enabled || entries.length > 0 || live}
      <HistoryList
        {entries}
        {live}
        {onCopy}
        {onCopyVersion}
        {onDelete}
        onCopyLive={onCopyFile}
        {ttsEnabled}
        {ttsAvailable}
        {ttsStatus}
        {onListen}
        onListenLive={onListenFile}
      />
    {:else}
      <div class="flex h-full min-h-48 flex-col items-center justify-center gap-3 p-8 text-center">
        <div class="grid size-9 place-items-center rounded-full bg-muted text-ink-quiet">
          <HistoryIcon class="size-[17px]" />
        </div>
        <div class="max-w-sm">
          <p class="text-[13px] font-medium">Nothing is being kept</p>
          <p class="mt-1.5 text-[11.5px] leading-relaxed text-muted-foreground">
            Turn history on to recover transcripts that did not land. Entries stay in memory only
            and are cleared when Freehand quits.
          </p>
        </div>
        <Button variant="outline" size="sm" onclick={onOpenSettings}>Turn history on</Button>
      </div>
    {/if}
    {#if showPlayback}
      <PlaybackBar
        status={ttsStatus}
        onPause={onPauseTTS}
        onResume={onResumeTTS}
        onRestart={onRestartTTS}
        onStop={onStopTTS}
        onSave={onSaveTTS}
        onClear={onClearTTS}
      />
    {/if}
  </div>
</section>

<style>
  .history-card {
    display: flex;
    flex-direction: column;
    /* Header plus the same two-pixel frame used by the other primary cards,
       so flex-grow alone drives the collapse without clipping the shared row. */
    flex: 0 1 calc(2.125rem + 2px);
    min-height: 0;
    overflow: hidden;
    transition: flex-grow 260ms cubic-bezier(0.4, 0, 0.2, 1);
  }
  .history-card.open {
    flex-grow: 1;
  }
  /* Without a header there is no collapsed height to hold, so the card simply
     fills the pane it was given. */
  .history-card.bare {
    flex: 1 1 0;
  }

  /* Inset rather than a border, so the hairline costs no height while closed. */
  .drawer {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    overflow: hidden;
    box-shadow: inset 0 1px 0 var(--hairline);
  }

  @media (prefers-reduced-motion: reduce) {
    .history-card {
      transition: none;
    }
  }
</style>
