<script lang="ts">
  import { tick } from "svelte";
  import { MediaQuery } from "svelte/reactivity";
  import PaneHeader from "$lib/components/home/PaneHeader.svelte";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import type { Message } from "$lib/utils/messages";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import HistoryList from "./HistoryList.svelte";
  import HistoryDetails from "./HistoryDetails.svelte";
  import * as Resizable from "$lib/components/ui/resizable";
  import HistorySidebar from "./HistorySidebar.svelte";
  import { Button } from "$lib/components/ui/button";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import PanelLeftIcon from "@lucide/svelte/icons/panel-left";
  import XIcon from "@lucide/svelte/icons/x";
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

  const uid = $props.id();
  const compact = new MediaQuery("(max-width: 699px)");
  const sideBySide = new MediaQuery("(min-width: 1100px)");
  let query = $state("");
  let source = $state<HistorySourceFilter>("all");
  let selectedID = $state<number | null>(null);
  let sidebarOpen = $state(false);
  let sidebarElement: HTMLDivElement;
  let sidebarToggle = $state<HTMLButtonElement | null>(null);
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

  // Preserve keyboard focus when the sidebar crosses its layout breakpoint.
  $effect.pre(() => {
    if (compact.current) {
      if (sidebarElement?.contains(document.activeElement)) sidebarOpen = true;
    } else {
      sidebarOpen = false;
      if (
        (sidebarToggle && document.activeElement === sidebarToggle) ||
        document.activeElement?.matches("[data-compact-history-control]")
      ) {
        void tick().then(() => {
          const target =
            sidebarElement?.querySelector<HTMLButtonElement>(
              '[aria-current="true"]',
            ) ?? sidebarElement?.querySelector<HTMLInputElement>("input");
          target?.focus();
        });
      }
    }
  });

  async function openSidebar() {
    sidebarOpen = true;
    await tick();
    sidebarElement?.querySelector<HTMLInputElement>("input")?.focus();
  }
  async function closeSidebar() {
    sidebarOpen = false;
    await tick();
    sidebarToggle?.focus();
  }
  function choose(id: number) {
    selectedID = id;
    if (compact.current && sidebarOpen) void closeSidebar();
  }
  function sidebarKey(event: KeyboardEvent) {
    if (
      compact.current &&
      sidebarOpen &&
      event.key === "Escape" &&
      !event.defaultPrevented
    ) {
      event.preventDefault();
      void closeSidebar();
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

<svelte:window onkeydown={sidebarKey} />

<div
  class="history-workbench relative flex min-h-0 min-w-0 flex-1 overflow-hidden"
>
  {#if compact.current && sidebarOpen}
    <button
      type="button"
      class="history-sidebar-backdrop"
      aria-label="Dismiss history sidebar"
      data-compact-history-control
      onclick={() => void closeSidebar()}
    ></button>
  {/if}
  <div
    bind:this={sidebarElement}
    id={`${uid}-sidebar`}
    class="history-browser-container relative flex min-h-0 shrink-0"
    style:display={compact.current && !sidebarOpen ? "none" : undefined}
  >
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
    {#if compact.current}
      <div class="absolute right-1 top-0.5">
        <TooltipButton
          label="Close history sidebar"
          data-compact-history-control
          size="icon-xs"
          onclick={() => void closeSidebar()}
        >
          <XIcon />
        </TooltipButton>
      </div>
    {/if}
  </div>

  <div
    class="flex min-h-0 min-w-0 flex-1 flex-col"
    inert={compact.current && sidebarOpen}
  >
    <PaneHeader
      title="History"
      summary={enabled
        ? `${entries.length} of ${retained.length} transcripts · in memory`
        : "retention is turned off"}
    >
      {#snippet actions()}
        {#if compact.current}
          <Button
            bind:ref={sidebarToggle}
            variant="outline"
            size="xs"
            aria-label="History sidebar"
            aria-expanded={sidebarOpen}
            aria-controls={`${uid}-sidebar`}
            onclick={() => void openSidebar()}
          >
            <PanelLeftIcon /> Browse
          </Button>
        {/if}
      {/snippet}
    </PaneHeader>

    <section
      class="flex min-h-0 min-w-0 flex-1 flex-col"
      aria-label="Transcript history"
    >
      {#if selected}
        <Resizable.PaneGroup
          direction={sideBySide.current ? "horizontal" : "vertical"}
          autoSaveId="freehand-history-reader"
          keyboardResizeBy={2}
        >
          <Resizable.Pane
            id="history-reader-pane"
            defaultSize={55}
            minSize={sideBySide.current ? 35 : 25}
          >
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
          </Resizable.Pane>
          <Resizable.Handle
            aria-label="Resize transcript and details"
            aria-orientation={sideBySide.current ? "vertical" : "horizontal"}
            class="w-1 rounded-none border-r border-hairline data-[direction=vertical]:h-1 data-[direction=vertical]:border-r-0 data-[direction=vertical]:border-t"
          />
          <Resizable.Pane
            id="history-details-pane"
            defaultSize={45}
            minSize={sideBySide.current ? 35 : 25}
          >
            <section
              class="flex h-full min-h-0 min-w-0 flex-col"
              aria-label="Selected transcription details"
            >
              {#key selected.id}
                <HistoryDetails entry={selected} embedded />
              {/key}
            </section>
          </Resizable.Pane>
        </Resizable.PaneGroup>
      {:else}
        <div
          class="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-5 text-center"
        >
          <p class="text-[15px] font-medium text-secondary-foreground">
            {!enabled
              ? "History is turned off."
              : filtered && retained.length
                ? "No matching transcripts."
                : "No transcripts yet."}
          </p>
          <p class="max-w-sm text-xs leading-relaxed text-muted-foreground">
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
      {#if showPlayback}
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

<style>
  .history-browser-container {
    width: 252px;
  }
  .history-sidebar-backdrop {
    position: absolute;
    inset: 0;
    z-index: 45;
    background: #0005;
  }
  @media (max-width: 699px) {
    .history-browser-container {
      position: absolute;
      inset: 0 auto 0 0;
      z-index: 46;
      width: min(280px, 85%);
      box-shadow: 8px 0 24px #0003;
    }
    .history-browser-container :global(.history-sidebar) {
      width: 100%;
    }
  }
</style>
