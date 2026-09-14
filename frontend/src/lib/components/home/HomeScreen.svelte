<script lang="ts">
  import { setContext } from "svelte";
  import { SETTINGS_NAVIGATION, type SettingsSectionID } from "$lib/navigation";
  import { TASK_CONNECTION_NAVIGATION } from "$lib/shell-navigation.svelte";
  import type { ConnectionManagerRequest } from "$bindings/windowing";
  import { Purpose } from "$bindings/savedconnection";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import CurrentResult from "$lib/components/home/CurrentResult.svelte";
  import * as WindowingService from "$bindings/windowing/service";
  import SpeechQuickSettings from "./SpeechQuickSettings.svelte";
  import WorkspaceSplit from "./WorkspaceSplit.svelte";
  import PaneHeader from "./PaneHeader.svelte";
  import RuntimeOutputDrawer from "$lib/components/runtimes/RuntimeOutputDrawer.svelte";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  import { Button } from "$lib/components/ui/button";
  import VoiceTranscriptionSettings from "./VoiceTranscriptionSettings.svelte";
  import QuickSettings from "$lib/components/home/QuickSettings.svelte";
  import ReadinessPanel from "$lib/components/home/ReadinessPanel.svelte";
  import FileChain from "$lib/components/chain/FileChain.svelte";
  import SpeechChain from "$lib/components/chain/SpeechChain.svelte";
  import HistoryPanel from "$lib/components/home/HistoryPanel.svelte";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import VoiceChain from "$lib/components/chain/VoiceChain.svelte";
  import TextToSpeech from "$lib/components/home/TextToSpeech.svelte";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import type { Session } from "$lib/stores/session.svelte";
  import type { Message } from "$lib/utils/messages";
  import { isFailure, statusMessage } from "$lib/utils/status";
  import { appReadiness, readinessVisible } from "$lib/utils/readiness";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { endpointHost } from "$lib/utils/endpoint";
  import {
    FileTranscriptionPhase,
    State,
    TTSPhase,
    TTSSource,
  } from "$lib/state";

  let {
    session,
    now,
    inputMode = $bindable("voice"),
    onOpenHistorySettings,
    onOpenServerSettings,
    onOpenProcessingSettings,
    onOpenAudioSettings,
    onOpenShortcutSettings,
    onOpenSpeechSettings,
    onOpenGeneralSettings,
    onOpenSettingsSection,
    onOpenConnection,
    quickSettingsDisabled = false,
  }: {
    session: Session;
    /** Shared shell clock; the input stage renders a per-second capture time. */
    now: number;
    inputMode: string;
    onOpenHistorySettings: () => void;
    onOpenServerSettings: () => void;
    onOpenProcessingSettings: () => void;
    onOpenAudioSettings: () => void;
    onOpenShortcutSettings: () => void;
    onOpenSpeechSettings: () => void;
    onOpenGeneralSettings: () => void;
    /** Navigates the shell to a settings section; there is no second window. */
    onOpenSettingsSection: (section: SettingsSectionID) => void;
    onOpenConnection: (request: ConnectionManagerRequest) => void;
    quickSettingsDisabled?: boolean;
  } = $props();

  setContext(SETTINGS_NAVIGATION, (section: SettingsSectionID) =>
    onOpenSettingsSection(section),
  );
  setContext(TASK_CONNECTION_NAVIGATION, (request: ConnectionManagerRequest) => {
    if (!quickSettingsDisabled) onOpenConnection(request);
  });

  const fileWorking = $derived(
    session.files.starting ||
      session.files.status.phase ===
        FileTranscriptionPhase.FileTranscriptionUploading ||
      session.files.status.phase ===
        FileTranscriptionPhase.FileTranscriptionProcessing ||
      session.files.status.phase ===
        FileTranscriptionPhase.FileTranscriptionStreaming ||
      session.files.status.phase ===
        FileTranscriptionPhase.FileTranscriptionCancelling,
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
  const runtimeSettings = $derived(
    session.editor.applied ?? session.editor.draft,
  );
  const instanceID = $derived(
    (inputMode === "file"
      ? runtimeSettings?.managedInstanceID
      : runtimeSettings?.voiceTranscription.managedInstanceID) ?? "",
  );
  const managed = $derived(!!instanceID);
  const runtime = $derived(session.runtime.statusFor(instanceID));
  const local = $derived(runtimePresentation(runtime?.status));
  const localModel = $derived(
    local.selected?.name || runtime?.instance.model || "Local speech",
  );
  const recordingAvailability = $derived(
    managed && (!local.ready || session.runtime.isBusy(instanceID))
      ? `Local speech: ${session.runtime.isBusy(instanceID) ? "Updating" : local.label}`
      : "",
  );
  function openLocalRuntime() {
    onOpenSettingsSection("local-runtime");
  }
  const readiness = $derived(
    runtimeSettings
      ? appReadiness(
          runtimeSettings,
          inputMode === "file"
            ? session.editor.connection
            : session.editor.currentVoiceConnection,
          session.editor.devices,
          session.editor.devicesBusy,
          inputMode === "file" ? "file" : "voice",
          runtime,
        )
      : null,
  );
  let dismissedRecoveryKey = $state("");
  const onAddConnection = addConnection;
  function addConnection(purpose: Purpose) {
    if (quickSettingsDisabled) return;
    onOpenConnection({ id: "", purpose, create: true });
  }
  const showReadiness = $derived(
    Boolean(
      inputMode !== "tts" &&
      readiness &&
      readinessVisible(readiness, dismissedRecoveryKey) &&
      // Runtime lifecycle is recoverable in quick settings. Keep that surface
      // mounted when stopping/switching models instead of navigating away.
      !(
        managed &&
        !readiness.initialSetup &&
        readiness.steps.every(
          (step) => !step.blocking || step.settingsSection === "local-runtime",
        )
      ) &&
      !voiceActive &&
      !fileWorking,
    ),
  );
  const hasHistory = $derived(
    Boolean(
      runtimeSettings?.historyEnabled && inputMode !== "tts" && !showReadiness,
    ),
  );
  const paneTitle = $derived(
    inputMode === "file"
      ? "Audio file"
      : inputMode === "tts"
        ? "Text to speech"
        : "Voice transcription",
  );
  const paneSummary = $derived(
    inputMode === "file"
      ? "file → transcribe → clean up → copy"
      : inputMode === "tts"
        ? "text → synthesize → play or save"
        : "mic → transcribe → clean up → deliver",
  );
  const liveDictation = $derived(
    inputMode === "voice" && !!runtimeSettings?.voiceTranscription.realtime,
  );

  const microphoneLabel = $derived.by(() => {
    const selectedID = runtimeSettings?.microphoneID ?? "";
    if (!selectedID) return "system default";
    return (
      session.editor.devices.find((device) => device.id === selectedID)?.name ??
      "selected"
    );
  });

  // Keep the active job visible. A hotkey can start voice capture while the
  // file tab is selected, so the UI must follow the work rather than hide it.
  $effect(() => {
    if (voiceActive) inputMode = "voice";
    else if (fileWorking) inputMode = "file";
    else if (
      ttsWorking &&
      session.speech.status.source === TTSSource.SourceCompose
    )
      inputMode = "tts";
  });

  const messages = $derived.by(() => {
    const out: Message[] = [];
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
      (session.speech.status.source !== TTSSource.SourceCompose ||
        inputMode === "tts") &&
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
  <PaneHeader title={paneTitle} summary={paneSummary}>
    {#snippet actions()}
      {#if liveDictation}
        <span
          class="inline-flex h-5 items-center rounded-sm border border-accent-edge bg-accent-wash px-1.5 text-[11px] text-accent-text"
          >Live dictation</span
        >
      {/if}
    {/snippet}
  </PaneHeader>

  <div class="transport-frame">
    {#if session.editor.draft}
      {#if inputMode === "tts" || !readiness?.initialSetup}
          {#if inputMode === "voice"}
            <VoiceChain
              {session}
              {now}
              busy={fileWorking}
              microphone={microphoneLabel}
              availability={recordingAvailability}
              transcribeModel={managed
                ? localModel
                : (runtimeSettings?.voiceTranscription.model ?? "")}
              transcribeSource={managed
                ? local.label
                : endpointHost(
                    runtimeSettings?.voiceTranscription.baseURL ?? "",
                  )}
              onOpenAudio={onOpenAudioSettings}
              onOpenTranscribe={() =>
                managed
                  ? openLocalRuntime()
                  : onOpenSettingsSection("voice-transcription")}
              onOpenCleanup={onOpenProcessingSettings}
              onOpenDelivery={onOpenGeneralSettings}
              onOpenRuntimes={openLocalRuntime}
              {onAddConnection}
            />
          {:else if inputMode === "file"}
            <FileChain
              {session}
              {now}
              blocked={voiceActive || ttsWorking
                ? "Finish the current job first"
                : ""}
              transcribeModel={managed
                ? localModel
                : (runtimeSettings?.model ?? "")}
              transcribeSource={managed
                ? local.label
                : endpointHost(runtimeSettings?.baseURL ?? "")}
              onOpenTranscribe={managed
                ? openLocalRuntime
                : onOpenServerSettings}
              onOpenCleanup={onOpenProcessingSettings}
              onOpenDelivery={onOpenHistorySettings}
              onOpenRuntimes={openLocalRuntime}
              {onAddConnection}
            />
          {:else}
            <SpeechChain
              {session}
              {now}
              onOpenSpeech={onOpenSpeechSettings}
              onOpenRuntimes={openLocalRuntime}
              {onAddConnection}
            />
          {/if}
      {/if}
    {:else}
      <Skeleton class="h-[132px] w-full rounded-none" />
    {/if}
  </div>
  <div class="body">
    {#if messages.length}<Notifications
        {messages}
        abovePlayback={inputMode === "tts"}
      />{/if}
    {#if session.speech.status.source !== TTSSource.SourceCompose && session.speech.status.phase !== TTSPhase.Idle && session.speech.status.phase !== TTSPhase.Cancelled}
      <PlaybackBar
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
        onOpenSettings={onOpenSpeechSettings}
      />
    {/if}
    {#if inputMode === "tts" && session.editor.draft}
      <TextToSpeech
        bind:text={session.speech.draft}
        settings={runtimeSettings?.textToSpeech ??
          session.editor.draft.textToSpeech}
        status={session.speech.status}
        unavailable={voiceActive || fileWorking}
        submitting={session.speech.submitting ||
          session.speech.listening !== null}
        onSpeak={(text) => session.speech.speakText(text)}
        onPause={() => session.speech.pauseTTS()}
        onResume={() => session.speech.resumeTTS()}
        onRestart={() => session.speech.restartTTS()}
        onSeek={(request) => session.speech.seekTTS(request)}
        seeking={session.speech.seeking}
        saving={session.speech.saving}
        onStop={() => session.speech.stopTTS()}
        onSave={() => session.speech.saveTTSAudio()}
        onClear={() => session.speech.clearTTSAudio()}
        onOpenSettings={onOpenSpeechSettings}
      >
        {#snippet quickSettings()}
          <SpeechQuickSettings
            settings={runtimeSettings!}
            runtime={session.runtime}
            onManageRuntime={openLocalRuntime}
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
        outputAvailable={managed && runtime?.status.state === "running"}
      >
        {#snippet output()}
          <RuntimeOutputDrawer
            {instanceID}
            running={runtime?.status.state === "running"}
          />
        {/snippet}
        {#snippet diagnostics()}
          <div class="min-h-0 flex-1 overflow-y-auto px-4 py-3">
            {#if inputMode === "file" ? session.editor.connection : session.editor.currentVoiceConnection}
              <ConnectionDiagnostics
                result={(inputMode === "file"
                  ? session.editor.connection
                  : session.editor.currentVoiceConnection)!}
                busy={inputMode === "file"
                  ? session.editor.sttConnectionTesting
                  : session.editor.voiceConnectionTesting}
                onCheck={() =>
                  session.editor.testAppliedConnection(
                    inputMode === "file" ? Purpose.Transcription : Purpose.Voice,
                  )}
              />
            {:else}
              <p class="text-[13px] text-muted-foreground">
                No endpoint check has run for this workflow yet.
              </p>
            {/if}
          </div>
        {/snippet}
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
                  : session.files.status.phase ===
                      FileTranscriptionPhase.FileTranscriptionFailed
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
                  : session.files.status.phase ===
                    FileTranscriptionPhase.FileTranscriptionFailed}
                resultKey={`${inputMode}:${inputMode === "file" ? session.files.status.generation : session.dictation.status.generation}`}
                mode={inputMode === "file" ? "file" : "voice"}
                text={inputMode === "file"
                  ? (session.files.status.transcript ?? "")
                  : (session.dictation.status.transcript ?? "")}
                working={inputMode === "file" ? fileWorking : voiceActive}
                canCopy={inputMode === "file"
                  ? session.files.status.canCopy
                  : Boolean(session.dictation.status.transcript)}
                recovery={inputMode === "voice" &&
                  session.dictation.status.canCopy}
                onCopy={() =>
                  inputMode === "file"
                    ? session.files.copyFileTranscript()
                    : session.dictation.copyCurrent()}
                onClear={() => {
                  if (inputMode === "file") void session.files.clearAudioFile();
                  else void session.dictation.clearCurrent();
                }}
                listenDisabled={!session.speech.canListen}
                listenBusy={session.speech.listening?.source ===
                  (inputMode === "file"
                    ? TTSSource.SourceFile
                    : TTSSource.SourceVoice) ||
                  (session.speech.status.phase === TTSPhase.Generating &&
                    session.speech.status.source ===
                      (inputMode === "file"
                        ? TTSSource.SourceFile
                        : TTSSource.SourceVoice))}
                onListen={runtimeSettings?.textToSpeech.enabled &&
                !voiceActive &&
                !fileWorking
                  ? () =>
                      inputMode === "file"
                        ? session.speech.listenFileTranscript()
                        : session.speech.listenVoiceTranscript(
                            session.dictation.status.generation,
                          )
                  : undefined}
              >
              </CurrentResult>
            {/if}

            {#if session.editor.draft}
              {#if showReadiness && readiness}
                <ReadinessPanel
                  {readiness}
                  task={inputMode === "file" ? "file" : "voice"}
                  saving={session.editor.saving ||
                    session.editor.quickSettingsPending.length > 0}
                  testing={inputMode === "file"
                    ? session.editor.sttConnectionTesting
                    : session.editor.voiceConnectionTesting}
                  completing={session.editor.setupCompleting}
                  onTestConnection={() =>
                    inputMode === "file"
                      ? session.editor.testConnection(
                          session.editor.applied,
                          "",
                        )
                      : session.editor.testVoiceConnection()}
                  onComplete={() => session.editor.completeSetup()}
                  onDismiss={() => {
                    dismissedRecoveryKey = readiness.recoveryKey;
                  }}
                  onOpenSettings={(section) => {
                    if (section === "local-runtime") openLocalRuntime();
                    else if (section === "audio") onOpenAudioSettings();
                    else if (section === "shortcuts") onOpenShortcutSettings();
                    else if (section === "voice-transcription")
                      onOpenSettingsSection("voice-transcription");
                    else onOpenServerSettings();
                  }}
                >
                  {#snippet serverControls()}
                    {#if inputMode === "voice"}
                      <VoiceTranscriptionSettings
                        setup
                        runtime={session.runtime}
                        onManageRuntime={openLocalRuntime}
                        editor={session.editor}
                        settings={runtimeSettings!}
                        disabled={quickSettingsDisabled ||
                          session.editor.saving}
                        onAddConnection={addConnection}
                      />
                    {:else}
                      <QuickSettings
                        runtime={session.runtime}
                        onManageRuntime={openLocalRuntime}
                        onEnterTranscription={() =>
                          void session.editor.ensureConnectionMetadata(
                            Purpose.Transcription,
                            true,
                          )}
                        sttMetadataStatus={session.editor.connectionMetadataStatus(
                          Purpose.Transcription,
                        )}
                        showCapture={false}
                        showCleanup={false}
                        settings={runtimeSettings!}
                        devices={session.editor.devices}
                        processingProfiles={session.editor.processingProfiles}
                        connection={session.editor.connection}
                        processingConnection={session.editor
                          .processingConnection}
                        sttStale={session.editor.sttConnectionStale ||
                          session.editor.connectionResultStale(
                            Purpose.Transcription,
                            runtimeSettings,
                          )}
                        processingStale={session.editor
                          .processingConnectionStale ||
                          session.editor.connectionResultStale(
                            Purpose.Cleanup,
                            runtimeSettings,
                          )}
                        pending={session.editor.quickSettingsPending}
                        savedField={session.editor.quickSettingsSaved}
                        failedField={session.editor.quickSettingsFailed}
                        sttTesting={session.editor.sttConnectionTesting}
                        processingTesting={session.editor
                          .processingConnectionTesting}
                        onAddConnection={addConnection}
                        onChangeConnection={(change) =>
                          session.editor.changeConnection(change)}
                        onUpdate={(patch, field) =>
                          session.editor.updateQuickSettings(patch, field)}
                        onTestConnection={() =>
                          session.editor.testConnection(
                            session.editor.applied,
                            "",
                          )}
                        onTestProcessingConnection={() =>
                          session.editor.testPostProcessingConnection(
                            session.editor.applied,
                            "",
                          )}
                        disabled={quickSettingsDisabled ||
                          session.editor.saving}
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
                  class="flex shrink-0 items-center justify-between gap-3 border-t border-hairline px-4 py-2 text-xs text-muted-foreground"
                >
                  <span
                    >History is off. Current results remain available until you
                    clear them or start again.</span
                  >
                  <Button
                    variant="ghost"
                    size="sm"
                    onclick={onOpenHistorySettings}>History settings</Button
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
                ttsAvailable={!voiceActive &&
                  !fileWorking &&
                  session.speech.canListen}
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
    padding: 0.875rem 1.25rem 1rem;
  }
  .transport-frame :global(.transport) {
    overflow: hidden;
    border: 1px solid var(--hairline);
    border-radius: 0.5rem;
    background: linear-gradient(115deg, var(--card), var(--layer-fill));
  }
  .body {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    gap: 0;
    padding: 0;
  }
  .task-main {
    height: 100%;
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    gap: 0;
    overflow: hidden;
    border-top: 1px solid var(--hairline);
    background: transparent;
  }
  .history-sidebar {
    height: 100%;
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    overflow: hidden;
    background: transparent;
  }
  .history-area {
    display: flex;
    flex-direction: column;
    min-height: 0;
    flex: 1;
    overflow: hidden;
  }
  .onboarding .task-main {
    padding: 1.25rem;
    gap: 1.25rem;
    overflow-y: auto;
  }
  .speech-workspace .body {
    padding-top: 0;
  }
  .columns {
    display: flex;
    gap: 0.875rem;
  }
  @container (max-width: 699px) {
    .transport-frame {
      padding: 0 0.75rem 0.75rem;
    }
    .body {
      padding: 0 0.75rem 0.75rem;
    }
  }
</style>
