<script lang="ts">
  import PaneHeader from "$lib/components/home/PaneHeader.svelte";
  import HistoryPanel from "$lib/components/home/HistoryPanel.svelte";
  import { Button } from "$lib/components/ui/button";
  import type { Session } from "$lib/stores/session.svelte";
  import { FileTranscriptionPhase, State, TTSPhase } from "$lib/state";

  let {
    session,
    onOpenHistorySettings,
  }: { session: Session; onOpenHistorySettings: () => void } = $props();

  const settings = $derived(session.editor.applied);
  const enabled = $derived(settings?.historyEnabled ?? false);
  const count = $derived(session.history.entries.length);

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
</script>

<!-- Recent transcripts sit under the workspace as a panel; this is the place
     that shows all of them at full width, without a chain above it. -->
<div class="flex min-h-0 flex-1 flex-col">
  <PaneHeader
    title="History"
    summary={enabled
      ? "kept in memory until you quit"
      : "retention is turned off"}
  >
    {#snippet actions()}
      <Button variant="outline" size="xs" onclick={onOpenHistorySettings}>
        History settings
      </Button>
    {/snippet}
  </PaneHeader>

  {#if enabled && count > 0}
    <div class="flex min-h-0 flex-1 flex-col px-5 pt-3 pb-4">
      <HistoryPanel
        {enabled}
        entries={session.history.entries}
        fileStatus={session.files.status}
        fileHistoryGeneration={session.files.historyGeneration}
        onOpenSettings={onOpenHistorySettings}
        onCopy={(id) => session.history.copyHistoryEntry(id)}
        onCopyVersion={(id, version) =>
          session.history.copyHistoryEntryVersion(id, version)}
        onDelete={(id) => session.history.deleteHistoryEntry(id)}
        onCopyFile={() => session.files.copyFileTranscript()}
        ttsEnabled={settings?.textToSpeech.enabled ?? false}
        ttsAvailable={!voiceActive && !fileWorking && session.speech.canListen}
        ttsPending={session.speech.listening ?? undefined}
        ttsStatus={session.speech.status}
        onListen={(id, version) =>
          session.speech.listenHistoryEntry(id, version)}
        onListenFile={() => session.speech.listenFileTranscript()}
        onPauseTTS={() => session.speech.pauseTTS()}
        onResumeTTS={() => session.speech.resumeTTS()}
        onRestartTTS={() => session.speech.restartTTS()}
        onSeekTTS={(request) => session.speech.seekTTS(request)}
        seeking={session.speech.seeking}
        saving={session.speech.saving}
        onStopTTS={() => session.speech.stopTTS()}
        onSaveTTS={() => session.speech.saveTTSAudio()}
        onClearTTS={() => session.speech.clearTTSAudio()}
        collapsible={false}
      />
    </div>
  {:else}
    <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 px-5">
      <p class="text-sm text-secondary-foreground">
        {enabled ? "No transcripts yet." : "History is turned off."}
      </p>
      <p class="max-w-sm text-center text-xs text-muted-foreground">
        {enabled
          ? "Completed transcripts appear here and are kept in memory until you quit."
          : "Turn retention on in History settings to keep recent transcripts for this session."}
      </p>
    </div>
  {/if}
</div>

<style>
  /* The panel fills a pane of its own here, so its playback bar is the only
     one on screen and should not defer to a workspace copy. */
  div :global([data-slot="history-panel"]) {
    flex: 1;
    min-height: 0;
  }
</style>
