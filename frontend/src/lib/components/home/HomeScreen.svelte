<script lang="ts">
  import { Purpose } from "$bindings/savedconnection";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import CurrentResult from "$lib/components/home/CurrentResult.svelte";
  import * as WindowingService from "$bindings/windowing/service";
  import SpeechQuickSettings from "./SpeechQuickSettings.svelte";
  import ResultQuickSettings from "./ResultQuickSettings.svelte";
  import WorkspaceSplit from "./WorkspaceSplit.svelte";
  import { Button } from "$lib/components/ui/button";
  import VoiceTranscriptionSettings from "./VoiceTranscriptionSettings.svelte";
  import QuickSettings from "$lib/components/home/QuickSettings.svelte";
  import ReadinessPanel from "$lib/components/home/ReadinessPanel.svelte";
  import AudioFileTranscription from "$lib/components/home/AudioFileTranscription.svelte";
  import HistoryPanel from "$lib/components/home/HistoryPanel.svelte";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import TransportBar from "$lib/components/home/TransportBar.svelte";
  import TextToSpeech from "$lib/components/home/TextToSpeech.svelte";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import type { Session } from "$lib/stores/session.svelte";
  import type { Message } from "$lib/utils/messages";
  import { isFailure, statusMessage } from "$lib/utils/status";
  import { appReadiness, readinessVisible } from "$lib/utils/readiness";
  import { FileTranscriptionPhase, State, TTSPhase, TTSSource } from "$lib/state";

  let {
    session,
    inputMode = $bindable("voice"),
    onOpenHistorySettings,
    onOpenServerSettings,
    onOpenProcessingSettings,
    onOpenAudioSettings,
    onOpenShortcutSettings,
    onOpenSpeechSettings,
    onOpenGeneralSettings,
    quickSettingsDisabled = false,
  }: {
    session: Session;
    inputMode: string;
    onOpenHistorySettings: () => void;
    onOpenServerSettings: () => void;
    onOpenProcessingSettings: () => void;
    onOpenAudioSettings: () => void;
    onOpenShortcutSettings: () => void;
    onOpenSpeechSettings: () => void;
    onOpenGeneralSettings: () => void;
    quickSettingsDisabled?: boolean;
  } = $props();

  const fileWorking = $derived(
    session.files.status.phase === FileTranscriptionPhase.FileTranscriptionUploading ||
      session.files.status.phase === FileTranscriptionPhase.FileTranscriptionProcessing ||
      session.files.status.phase === FileTranscriptionPhase.FileTranscriptionStreaming ||
      session.files.status.phase === FileTranscriptionPhase.FileTranscriptionCancelling,
  );
  const voiceActive = $derived(
    session.dictation.status.state !== State.Idle &&
      session.dictation.status.state !== State.Failed,
  );
  const ttsWorking = $derived(
    session.speech.status.phase === TTSPhase.Generating ||
      session.speech.status.phase === TTSPhase.Playing ||
      session.speech.status.phase === TTSPhase.Paused,
  );
  const runtimeSettings = $derived(session.editor.applied ?? session.editor.draft);
  const readiness = $derived(
    runtimeSettings
      ? appReadiness(
          runtimeSettings,
          inputMode === "file" ? session.editor.connection : session.editor.currentVoiceConnection,
          session.editor.devices,
          session.editor.devicesBusy,
          inputMode === "file" ? "file" : "voice",
        )
      : null,
  );
  let dismissedRecoveryKey = $state("");
  function addConnection(purpose: Purpose) {
    if (quickSettingsDisabled) return;
    void WindowingService.OpenConnectionManager({ id: "", purpose, create: true }).catch((cause) =>
      session.messages.reportFailure(String(cause)),
    );
  }
  const showReadiness = $derived(
    Boolean(
      inputMode !== "tts" &&
        readiness &&
        readinessVisible(readiness, dismissedRecoveryKey) &&
        !voiceActive &&
        !fileWorking,
    ),
  );
  const hasHistory = $derived(
    Boolean(runtimeSettings?.historyEnabled && inputMode !== "tts" && !showReadiness),
  );
  const microphoneLabel = $derived.by(() => {
    const selectedID = runtimeSettings?.microphoneID ?? "";
    if (!selectedID) return "system default";
    return session.editor.devices.find((device) => device.id === selectedID)?.name ?? "selected";
  });

  // Keep the active job visible. A hotkey can start voice capture while the
  // file tab is selected, so the UI must follow the work rather than hide it.
  $effect(() => {
    if (voiceActive) inputMode = "voice";
    else if (fileWorking) inputMode = "file";
    else if (ttsWorking && session.speech.status.source === TTSSource.SourceCompose)
      inputMode = "tts";
  });

  const messages = $derived.by(() => {
    const out: Message[] = [];
    const configuration = runtimeSettings?.configuration;
    const preservedFields = configuration?.preservedFields ?? [];
    if (preservedFields.length > 0) {
      const remaining = Math.max(
        0,
        (configuration?.preservedFieldCount ?? preservedFields.length) - preservedFields.length,
      );
      out.push({
        id: "configuration-compatibility",
        tone: "info",
        source: "system",
        text: `Settings from a newer Freehand version are preserved but cannot be edited here: ${preservedFields.join(", ")}${remaining > 0 ? `, and ${remaining} more` : ""}.`,
      });
    }
    if (session.messages.info) {
      out.push({
        id: "system-info",
        tone: "info",
        source: "system",
        text: session.messages.info,
        onDismiss: () => session.messages.dismissInfo(),
      });
    }
    const speechFailureVisible =
      session.speech.status.phase === TTSPhase.Failed &&
      (session.speech.status.source !== TTSSource.SourceCompose || inputMode === "tts") &&
      session.messages.isSpeechFailure(session.speech.status.generation);
    if (session.messages.error && !speechFailureVisible) {
      out.push({
        id: "error",
        tone: "error",
        source: "action",
        text: session.messages.error,
        onDismiss: () => session.messages.dismissError(),
      });
    }
    if (session.messages.notice) {
      out.push({
        id: "notice",
        tone: "success",
        source: "action",
        text: session.messages.notice,
        onDismiss: () => session.messages.dismissNotice(),
      });
    }
    return out;
  });
</script>

<main
  class="home"
  class:with-history={hasHistory}
  class:speech-workspace={inputMode === "tts"}
  class:onboarding={showReadiness}
  aria-label="Freehand workspace"
>
  {#if inputMode !== "tts"}
    <div class="transport-frame">
      {#if session.editor.draft}
        {#if !showReadiness}
          {#if inputMode === "voice"}
            <TransportBar
              status={session.dictation.status}
              busy={fileWorking}
              toggleShortcut={session.editor.draft.toggleShortcut}
              model={runtimeSettings?.voiceTranscription.model ?? ""}
              processingModel={runtimeSettings?.postProcessing.model ?? ""}
              microphone={microphoneLabel}
              onToggle={() => session.dictation.toggleRecording()}
              onCancel={() => session.dictation.cancel()}
              onCopy={() => session.dictation.copyPending()}
              onOpenSettings={() => void WindowingService.OpenSettings("voice-transcription")}
            />
          {:else if inputMode === "file"}
            <AudioFileTranscription
              status={session.files.status}
              choosing={session.files.choosing}
              streamingEnabled={session.files.streamingEnabled}
              resettingStreaming={session.files.resettingStreaming}
              onStreamingChange={(enabled) => (session.files.streamingPreferred = enabled)}
              voiceActive={voiceActive || ttsWorking}
              onOpenSettings={onOpenServerSettings}
              onChoose={() => session.files.chooseAudioFile()}
              onStart={() => session.files.startFileTranscription()}
              onTryStreamingAgain={() => session.files.tryFileStreamingAgain()}
              onCancel={() => session.files.cancelFileTranscription()}
              onClear={() => session.files.clearAudioFile()}
            />
          {/if}
        {/if}
      {:else}
        <Skeleton class="h-[132px] w-full rounded-none" />
      {/if}
    </div>
  {/if}
  <div class="body">
    {#if messages.length}<Notifications {messages} abovePlayback={inputMode === "tts"} />{/if}
    {#if session.speech.status.source !== TTSSource.SourceCompose && session.speech.status.phase !== TTSPhase.Idle && session.speech.status.phase !== TTSPhase.Cancelled}
      <PlaybackBar
        status={session.speech.status}
        onPause={() => session.speech.pauseTTS()}
        onResume={() => session.speech.resumeTTS()}
        onRestart={() => session.speech.restartTTS()}
        onSeek={(request) => session.speech.seekTTS(request)}
        seeking={session.speech.seeking}
        onStop={() => session.speech.stopTTS()}
        onSave={() => session.speech.saveTTSAudio()}
        onClear={() => session.speech.clearTTSAudio()}
        onOpenSettings={onOpenSpeechSettings}
      />
    {/if}
    {#if inputMode === "tts" && session.editor.draft}
      <TextToSpeech
        bind:text={session.speech.draft}
        settings={runtimeSettings?.textToSpeech ?? session.editor.draft.textToSpeech}
        status={session.speech.status}
        unavailable={voiceActive || fileWorking}
        submitting={session.speech.submitting}
        onSpeak={(text) => session.speech.speakText(text)}
        onPause={() => session.speech.pauseTTS()}
        onResume={() => session.speech.resumeTTS()}
        onRestart={() => session.speech.restartTTS()}
        onSeek={(request) => session.speech.seekTTS(request)}
        seeking={session.speech.seeking}
        onStop={() => session.speech.stopTTS()}
        onSave={() => session.speech.saveTTSAudio()}
        onClear={() => session.speech.clearTTSAudio()}
        onOpenSettings={onOpenSpeechSettings}
      >
        {#snippet quickSettings()}
          <SpeechQuickSettings
            settings={runtimeSettings!}
            editor={session.editor}
            disabled={quickSettingsDisabled || session.editor.saving}
            onAddConnection={addConnection}
            onOpenSettings={onOpenSpeechSettings}
          />
        {/snippet}
      </TextToSpeech>
    {:else}
      <WorkspaceSplit
        {hasHistory}
        historyCount={session.history.entries.length}
        working={voiceActive || fileWorking}
      >
        {#snippet result()}
          <div class="task-main">
            {#if session.editor.draft && inputMode !== "tts" && (!showReadiness || (inputMode === "file" ? session.files.status.transcript : session.dictation.status.transcript))}
              <CurrentResult
                live={inputMode === "voice" &&
                  session.dictation.status.live &&
                  (session.dictation.status.state === "recording" ||
                    session.dictation.status.state === "transcribing")}
                liveFinal={session.dictation.status.liveFinal ?? ""}
                livePartial={session.dictation.status.livePartial ?? ""}
                message={inputMode === "voice"
                  ? isFailure(session.dictation.status)
                    ? ""
                    : (statusMessage(session.dictation.status) ?? "")
                  : session.files.status.phase === FileTranscriptionPhase.FileTranscriptionFailed
                    ? session.files.status.transcript
                      ? "Transcription did not finish. Any text received is kept below."
                      : ""
                    : session.files.status.phase ===
                          FileTranscriptionPhase.FileTranscriptionCompleted &&
                        !session.files.status.transcript
                      ? "No speech was detected. Choose another file or transcribe again."
                      : ""}
                failed={inputMode === "voice"
                  ? isFailure(session.dictation.status)
                  : session.files.status.phase === FileTranscriptionPhase.FileTranscriptionFailed}
                resultKey={`${inputMode}:${inputMode === "file" ? session.files.status.generation : session.dictation.status.generation}`}
                mode={inputMode === "file" ? "file" : "voice"}
                text={inputMode === "file"
                  ? (session.files.status.transcript ?? "")
                  : (session.dictation.status.transcript ?? "")}
                working={inputMode === "file" ? fileWorking : voiceActive}
                canCopy={inputMode === "file"
                  ? session.files.status.canCopy
                  : Boolean(session.dictation.status.transcript)}
                recovery={inputMode === "voice" && session.dictation.status.canCopy}
                onCopy={() =>
                  inputMode === "file"
                    ? session.files.copyFileTranscript()
                    : session.dictation.copyCurrent()}
                onClear={() => {
                  if (inputMode === "file") void session.files.clearAudioFile();
                  else void session.dictation.clearCurrent();
                }}
                onListen={runtimeSettings?.textToSpeech.enabled && !voiceActive && !fileWorking
                  ? () =>
                      inputMode === "file"
                        ? session.speech.listenFileTranscript()
                        : session.speech.listenVoiceTranscript(session.dictation.status.generation)
                  : undefined}
              >
                {#snippet quickSettings()}
                  <ResultQuickSettings
                    settings={runtimeSettings!}
                    editor={session.editor}
                    showCapture={inputMode === "voice"}
                    disabled={quickSettingsDisabled || session.editor.saving}
                    onAddConnection={addConnection}
                    {onOpenServerSettings}
                    {onOpenProcessingSettings}
                    {onOpenAudioSettings}
                    {onOpenGeneralSettings}
                  />
                {/snippet}
              </CurrentResult>
            {/if}

            {#if session.editor.draft}
              {#if showReadiness && readiness}
                <ReadinessPanel
                  {readiness}
                  task={inputMode === "file" ? "file" : "voice"}
                  saving={session.editor.saving || session.editor.quickSettingsPending.length > 0}
                  testing={inputMode === "file"
                    ? session.editor.sttConnectionTesting
                    : session.editor.voiceConnectionTesting}
                  completing={session.editor.setupCompleting}
                  onTestConnection={() =>
                    inputMode === "file"
                      ? session.editor.testConnection(session.editor.applied, "")
                      : session.editor.testVoiceConnection()}
                  onComplete={() => session.editor.completeSetup()}
                  onDismiss={() => {
                    dismissedRecoveryKey = readiness.recoveryKey;
                  }}
                  onOpenSettings={(section) => {
                    if (section === "audio") onOpenAudioSettings();
                    else if (section === "shortcuts") onOpenShortcutSettings();
                    else if (section === "voice-transcription")
                      void WindowingService.OpenSettings("voice-transcription");
                    else onOpenServerSettings();
                  }}
                >
                  {#snippet serverControls()}
                    {#if inputMode === "voice"}
                      <VoiceTranscriptionSettings
                        setup
                        editor={session.editor}
                        settings={runtimeSettings!}
                        disabled={quickSettingsDisabled || session.editor.saving}
                        onAddConnection={addConnection}
                      />
                    {:else}
                      <QuickSettings
                        showCapture={false}
                        showCleanup={false}
                        settings={runtimeSettings!}
                        devices={session.editor.devices}
                        processingProfiles={session.editor.processingProfiles}
                        connection={session.editor.connection}
                        processingConnection={session.editor.processingConnection}
                        sttStale={session.editor.sttConnectionStale ||
                          session.editor.connectionResultStale(
                            Purpose.Transcription,
                            runtimeSettings,
                          )}
                        processingStale={session.editor.processingConnectionStale ||
                          session.editor.connectionResultStale(Purpose.Cleanup, runtimeSettings)}
                        pending={session.editor.quickSettingsPending}
                        savedField={session.editor.quickSettingsSaved}
                        failedField={session.editor.quickSettingsFailed}
                        sttTesting={session.editor.sttConnectionTesting}
                        processingTesting={session.editor.processingConnectionTesting}
                        onAddConnection={addConnection}
                        onChangeConnection={(change) => session.editor.changeConnection(change)}
                        onUpdate={(patch, field) =>
                          session.editor.updateQuickSettings(patch, field)}
                        onTestConnection={() =>
                          session.editor.testConnection(session.editor.applied, "")}
                        onTestProcessingConnection={() =>
                          session.editor.testPostProcessingConnection(session.editor.applied, "")}
                        disabled={quickSettingsDisabled || session.editor.saving}
                        {onOpenServerSettings}
                        {onOpenProcessingSettings}
                        {onOpenAudioSettings}
                        onOpenDeliverySettings={onOpenGeneralSettings}
                      />
                    {/if}
                  {/snippet}
                </ReadinessPanel>
              {:else if !runtimeSettings?.historyEnabled}
                <div
                  class="flex items-center justify-between gap-3 px-1 text-xs text-muted-foreground"
                >
                  <span
                    >History is off. Current results remain available until you clear them or start
                    again.</span
                  >
                  <Button variant="ghost" size="sm" onclick={onOpenHistorySettings}
                    >History settings</Button
                  >
                </div>
              {/if}
            {:else}
              <div class="columns">
                <Skeleton class="h-full w-[372px] shrink-0 rounded-lg" />
                <Skeleton class="h-full flex-1 rounded-lg" />
              </div>
            {/if}
          </div>
        {/snippet}
        {#snippet history()}
          <aside class="history-sidebar" aria-label="Recent history">
            <div
              class="flex min-h-14 shrink-0 flex-wrap items-baseline gap-x-2 gap-y-1 border-b border-hairline px-4 py-3"
            >
              <h2 class="text-sm font-semibold">
                Recent history <span class="ml-1 text-muted-foreground"
                  >{session.history.entries.length}</span
                >
              </h2>
              <span class="text-xs text-muted-foreground">In memory until you quit</span>
            </div>
            <div class="history-area">
              <HistoryPanel
                enabled={session.editor.applied?.historyEnabled ?? false}
                entries={session.history.entries}
                fileStatus={session.files.status}
                fileHistoryGeneration={session.files.historyGeneration}
                onOpenSettings={onOpenHistorySettings}
                onCopy={(id) => session.history.copyHistoryEntry(id)}
                onCopyVersion={(id, version) =>
                  session.history.copyHistoryEntryVersion(id, version)}
                onDelete={(id) => session.history.deleteHistoryEntry(id)}
                onCopyFile={() => session.files.copyFileTranscript()}
                ttsEnabled={runtimeSettings?.textToSpeech.enabled ?? false}
                ttsAvailable={!voiceActive && !fileWorking}
                ttsStatus={session.speech.status}
                onListen={(id, version) => session.speech.listenHistoryEntry(id, version)}
                onListenFile={() => session.speech.listenFileTranscript()}
                onPauseTTS={() => session.speech.pauseTTS()}
                onResumeTTS={() => session.speech.resumeTTS()}
                onRestartTTS={() => session.speech.restartTTS()}
                onSeekTTS={(request) => session.speech.seekTTS(request)}
                seeking={session.speech.seeking}
                onStopTTS={() => session.speech.stopTTS()}
                onSaveTTS={() => session.speech.saveTTSAudio()}
                onClearTTS={() => session.speech.clearTTSAudio()}
                ttsWorkspaceVisible={inputMode === "tts"}
                collapsible={false}
                playbackVisibleElsewhere
              />
            </div>
          </aside>
        {/snippet}
      </WorkspaceSplit>
    {/if}
  </div>
</main>

<style>
  .home {
    container-type: inline-size;
    width: 100%;
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    overflow: hidden;
  }
  .transport-frame {
    flex-shrink: 0;
    padding: 1rem 1rem 0;
  }
  .transport-frame :global(.transport) {
    border: 1px solid var(--hairline);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }
  .body {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    gap: 0.75rem;
    padding: 0.75rem 1rem 1rem;
  }
  .task-main {
    height: 100%;
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    gap: 0.75rem;
  }
  .history-sidebar {
    height: 100%;
    --history-max-height: none;
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-lg);
    background: var(--layer-fill);
  }
  .history-area {
    min-height: 0;
    flex: 1;
    overflow-y: auto;
  }
  .onboarding .task-main {
    overflow-y: auto;
  }
  .speech-workspace .body {
    padding-top: 1rem;
  }
  .columns {
    display: flex;
    gap: 0.875rem;
  }
</style>
