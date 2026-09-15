<script lang="ts">
  import PaneHeader from "$lib/components/home/PaneHeader.svelte";
  import HistoryIcon from "@lucide/svelte/icons/history";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import type { Message } from "$lib/utils/messages";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import HistoryList from "./HistoryList.svelte";
  import HistoryDetails from "./HistoryDetails.svelte";
  import SidebarContribution from "$lib/components/shell/SidebarContribution.svelte";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
  import HistorySidebar from "./HistorySidebar.svelte";
  import { Button } from "$lib/components/ui/button";
  import type { Session } from "$lib/stores/session.svelte";
  import { FileTranscriptionPhase, State, TTSPhase } from "$lib/state";
  import {
    filterHistoryEntries,
    selectedHistoryEntry,
    type HistorySourceFilter,
  } from "$lib/utils/historyBrowser";

  let {
    session,
    onOpenHistorySettings,
  }: { session: Session; onOpenHistorySettings: () => void } = $props();

  const layout = getWorkbenchLayout();
  let query = $state("");
  let source = $state<HistorySourceFilter>("all");
  let selectedID = $state<number | null>(null);
  let clearing = $state(false);
  const settings = $derived(session.editor.applied);
  const enabled = $derived(settings?.historyEnabled ?? false);
  const retained = $derived(enabled ? session.history.entries : []);
  const entries = $derived(filterHistoryEntries(retained, query, source));
  const selected = $derived(selectedHistoryEntry(entries, selectedID));
  const filtered = $derived(Boolean(query.trim()) || source !== "all");

  const voiceActive = $derived(
    session.dictation.status.state !== State.Idle &&
      session.dictation.status.state !== State.Failed,
  );
  const fileWorking = $derived(
    session.files.starting ||
      [
        FileTranscriptionPhase.FileTranscriptionUploading,
        FileTranscriptionPhase.FileTranscriptionProcessing,
        FileTranscriptionPhase.FileTranscriptionStreaming,
        FileTranscriptionPhase.FileTranscriptionCancelling,
      ].includes(session.files.status.phase),
  );
  const showPlayback = $derived(
    session.speech.status.phase !== TTSPhase.Idle &&
      session.speech.status.phase !== TTSPhase.Cancelled,
  );

  const messages = $derived.by(() => {
    const out: Message[] = [];
    if (session.messages.info)
      out.push({
        id: "system-info",
        tone: "info",
        source: "system",
        text: session.messages.info,
        onDismiss: () => session.messages.dismissInfo(),
      });
    const speechFailureVisible =
      showPlayback &&
      session.speech.status.phase === TTSPhase.Failed &&
      session.messages.isSpeechFailure(session.speech.status.generation);
    if (session.messages.error && !speechFailureVisible)
      out.push({
        id: "error",
        tone: "error",
        source: "action",
        text: session.messages.error,
        onDismiss: () => session.messages.dismissError(),
      });
    if (session.messages.notice)
      out.push({
        id: "notice",
        tone: "success",
        source: "action",
        text: session.messages.notice,
        onDismiss: () => session.messages.dismissNotice(),
      });
    return out;
  });

  $effect(() => {
    if (!layout) return;
    layout.details = details;
    return () => {
      if (layout.details === details) layout.details = undefined;
    };
  });

  function choose(id: number) {
    selectedID = id;
    if (layout?.compact.current) {
      layout.compactPrimaryOpen = false;
      document
        .querySelector<HTMLButtonElement>(
          'button[aria-label="Toggle primary sidebar"]',
        )
        ?.focus();
    }
  }

  async function clearHistory() {
    if (clearing || !retained.length) return;
    clearing = true;
    try {
      await session.history.clearHistory();
    } finally {
      clearing = false;
    }
  }
</script>

{#snippet details()}
  <section
    class="flex h-full min-h-0 min-w-0 flex-col"
    aria-label="Selected transcription details"
  >
    {#if selected}
      {#key selected.id}<HistoryDetails entry={selected} embedded />{/key}
    {:else}
      <PaneHeader title="Transcription details" />

      <p class="content-meta px-5 py-4">
        Select a transcript to inspect its details.
      </p>
    {/if}
  </section>
{/snippet}

<div
  class="history-workbench relative flex min-h-0 min-w-0 flex-1 overflow-hidden"
>
  <SidebarContribution id="history">
    <HistorySidebar
      {entries}
      totalEntries={retained}
      {enabled}
      {clearing}
      selectedID={selected?.id}
      bind:query
      bind:source
      onSelect={choose}
      onOpenSettings={onOpenHistorySettings}
      onClear={() => void clearHistory()}
    />
  </SidebarContribution>

  <div class="flex min-h-0 min-w-0 flex-1 flex-col">
    <PaneHeader
      title="History"
      icon={HistoryIcon}
      summary={enabled
        ? `${entries.length} of ${retained.length} transcripts · in memory`
        : "retention is turned off"}
    />

    <section
      class="flex min-h-0 min-w-0 flex-1 flex-col"
      aria-label="Transcript history"
    >
      {#if selected}
        <section
          class="flex h-full min-h-0 min-w-0 flex-col"
          aria-label="Selected transcript"
        >
          {#key selected.id}
            <HistoryList
              entries={[selected]}
              reader
              onCopy={(id) => session.history.copyHistoryEntry(id)}
              onCopyVersion={(id, version) =>
                session.history.copyHistoryEntryVersion(id, version)}
              onDelete={(id) => session.history.deleteHistoryEntry(id)}
              ttsEnabled={settings?.textToSpeech.enabled ?? false}
              ttsAvailable={!voiceActive &&
                !fileWorking &&
                session.speech.canListen}
              ttsPending={session.speech.listening ?? undefined}
              ttsStatus={session.speech.status}
              onListen={(id, version) =>
                session.speech.listenHistoryEntry(id, version)}
            />
          {/key}
        </section>
      {:else}
        <div
          class="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-5 text-center"
        >
          <p class="content-title">
            {!enabled
              ? "History is turned off."
              : filtered && retained.length
                ? "No matching transcripts."
                : "No transcripts yet."}
          </p>

          <p class="content-meta max-w-sm">
            {!enabled
              ? "Turn retention on in History settings to keep recent transcripts for this session."
              : filtered && retained.length
                ? "Try another search or show all sources."
                : "Completed transcripts appear here and are kept in memory until you quit."}
          </p>

          {#if !enabled}
            <Button variant="outline" size="sm" onclick={onOpenHistorySettings}
              >History settings</Button
            >
          {:else if filtered}
            <Button
              variant="outline"
              size="sm"
              onclick={() => {
                query = "";
                source = "all";
              }}>Reset filters</Button
            >
          {/if}
        </div>
      {/if}

      {#if showPlayback && !layout}
        <div class="shrink-0 border-t border-hairline">
          <PlaybackBar
            embedded
            status={session.speech.status}
            onPause={() => session.speech.pauseTTS()}
            onResume={() => session.speech.resumeTTS()}
            onRestart={() => session.speech.restartTTS()}
            onSeek={(request) => session.speech.seekTTS(request)}
            seeking={session.speech.seeking}
            saving={session.speech.saving}
            onStop={() => session.speech.stopTTS()}
            onSave={() => session.speech.saveTTSAudio()}
            onClear={() => session.speech.clearTTSAudio()}
          />
        </div>
      {/if}
    </section>
  </div>

  {#if messages.length}
    <div class="relative z-50">
      <Notifications {messages} abovePlayback={showPlayback} />
    </div>
  {/if}
</div>
