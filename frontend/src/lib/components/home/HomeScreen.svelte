<script lang="ts">
  import { setContext } from "svelte";
  import MicIcon from "@lucide/svelte/icons/mic";
  import FileAudioIcon from "@lucide/svelte/icons/file-audio";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import PaneHeader from "./PaneHeader.svelte";
  import { MediaQuery } from "svelte/reactivity";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
  import SidebarContribution from "$lib/components/shell/SidebarContribution.svelte";
  import { SETTINGS_NAVIGATION, type SettingsSectionID } from "$lib/navigation";
  import { TASK_CONNECTION_NAVIGATION } from "$lib/shell-navigation.svelte";
  import type { ConnectionManagerRequest } from "$bindings/windowing";
  import { Purpose } from "$bindings/savedconnection";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import CurrentResult from "$lib/components/home/CurrentResult.svelte";
  import WorkspaceSplit from "./WorkspaceSplit.svelte";
  import VoiceBar from "./VoiceBar.svelte";
  import WorkflowSettingsButton from "./WorkflowSettingsButton.svelte";
  import RuntimeOutputDrawer from "$lib/components/runtimes/RuntimeOutputDrawer.svelte";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  import { Button } from "$lib/components/ui/button";
  import VoiceTranscriptionSettings from "./VoiceTranscriptionSettings.svelte";
  import QuickSettings from "$lib/components/home/QuickSettings.svelte";
  import ReadinessPanel from "$lib/components/home/ReadinessPanel.svelte";
  import FileBar from "./FileBar.svelte";
  import HistoryPanel from "$lib/components/home/HistoryPanel.svelte";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import WorkflowSidebar from "./WorkflowSidebar.svelte";
  import TextToSpeech from "$lib/components/home/TextToSpeech.svelte";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import type { Session } from "$lib/stores/session.svelte";
  import type { Message } from "$lib/utils/messages";
  import { isFailure, statusMessage } from "$lib/utils/status";
  import { appReadiness, readinessVisible } from "$lib/utils/readiness";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import { taskConnectionDetails } from "$lib/utils/connection";
  import {
    managedVoiceAvailability,
    manualVoiceAvailability,
  } from "$lib/utils/voiceCaptureAvailability";
  import {
    FileTranscriptionPhase,
    State,
    TTSPhase,
    TTSSource,
  } from "$lib/state";

  let {
    session,
    now,
    active = true,
    inputMode = $bindable("voice"),
    onOpenHistorySettings,
    onOpenServerSettings,
    onOpenProcessingSettings,
    onOpenAudioSettings,
    onOpenShortcutSettings,
    onOpenSpeechSettings,
    onOpenGeneralSettings,
    onOpenDeliverySettings = onOpenGeneralSettings,
    onOpenSettingsSection,
    onOpenConnection,
    quickSettingsDisabled = false,
  }: {
    session: Session;
    /** Shared shell clock; the input stage renders a per-second capture time. */
    now: number;
    /** The retained workspace releases sensitive readers while another pane is shown. */
    active?: boolean;
    inputMode: string;
    onOpenHistorySettings: () => void;
    onOpenServerSettings: () => void;
    onOpenProcessingSettings: () => void;
    onOpenAudioSettings: () => void;
    onOpenShortcutSettings: () => void;
    onOpenSpeechSettings: () => void;
    onOpenGeneralSettings: () => void;
    onOpenDeliverySettings?: () => void;
    /** Navigates the shell to a settings section; there is no second window. */
    onOpenSettingsSection: (section: SettingsSectionID) => void;
    onOpenConnection: (request: ConnectionManagerRequest) => void;
    quickSettingsDisabled?: boolean;
  } = $props();
  const workbench = getWorkbenchLayout();
  const compact = new MediaQuery("(max-width: 699px)");
  let taskSettingsOpen = $state(false);
  let taskSettingsTrigger = $state<HTMLButtonElement | null>(null);

  setContext(SETTINGS_NAVIGATION, (section: SettingsSectionID) =>
    onOpenSettingsSection(section),
  );
  setContext(
    TASK_CONNECTION_NAVIGATION,
    (request: ConnectionManagerRequest) => {
      if (!quickSettingsDisabled) onOpenConnection(request);
    },
  );

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
      : inputMode === "tts"
        ? runtimeSettings?.textToSpeech.managedInstanceID
        : runtimeSettings?.voiceTranscription.managedInstanceID) ?? "",
  );
  const managed = $derived(!!instanceID);
  const runtime = $derived(session.runtime.statusFor(instanceID));
  const local = $derived(runtimePresentation(runtime?.status));
  const localModel = $derived(
    local.selected?.name || runtime?.instance.model || "Local speech",
  );
  const voiceConnection = $derived(
    taskConnectionDetails("voice", session.editor),
  );
  const recordingAvailability = $derived(
    managed
      ? managedVoiceAvailability(runtime?.status, now, {
          pending: session.runtime.pendingFor(instanceID),
          busy: session.runtime.isBusy(instanceID),
          checking:
            session.runtime.loading || !session.runtime.providers.length,
          error:
            session.runtime.errorFor(instanceID) ||
            (!runtime ? session.runtime.error : ""),
        })
      : manualVoiceAvailability(
          voiceConnection,
          session.editor.voiceConnectionChecked,
        ),
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
  function addConnection(purpose: Purpose) {
    if (quickSettingsDisabled) return;
    onOpenConnection({ id: "", purpose, create: true });
  }
  const showReadiness = $derived(
    Boolean(
      inputMode !== "tts" &&
      readiness &&
      readinessVisible(readiness, dismissedRecoveryKey) &&
      // Lifecycle and metadata availability belong in the capture strip. Keep
      // the transcript mounted while these recoverable checks change state.
      !(
        !readiness.initialSetup &&
        readiness.steps.every(
          (step) =>
            !step.blocking ||
            (managed && step.settingsSection === "local-runtime") ||
            (inputMode === "voice" && !managed && step.id === "connection"),
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

<svelte:window
  onkeydown={(event) => {
    if (
      event.key === "Escape" &&
      !event.defaultPrevented &&
      !workbench &&
      compact.current &&
      taskSettingsOpen
    ) {
      taskSettingsOpen = false;
      taskSettingsTrigger?.focus();
    }
  }}
/>

<main
  class="home"
  class:onboarding={showReadiness}
  aria-label="Freehand workspace"
>
  {#if !workbench && compact.current && session.editor.draft && (inputMode === "tts" || !readiness?.initialSetup)}
    <div class="flex shrink-0 items-center border-b border-hairline px-3 py-1">
      <Button
        variant="ghost"
        size="sm"
        aria-expanded={taskSettingsOpen}
        bind:ref={taskSettingsTrigger}
        onclick={() => (taskSettingsOpen = !taskSettingsOpen)}
        >Task settings</Button
      >
      <span class="ml-auto text-xs text-muted-foreground">{paneTitle}</span>
    </div>
  {/if}
  <div class="workspace" class:fallback={!workbench}>
    {#if !workbench && compact.current && taskSettingsOpen}<button
        type="button"
        class="settings-backdrop"
        aria-label="Close task settings"
        onclick={() => (taskSettingsOpen = false)}
      ></button>{/if}
    {#if session.editor.draft && (workbench || !compact.current || taskSettingsOpen) && (inputMode === "tts" || !readiness?.initialSetup)}
      {@const workflowSection =
        inputMode === "file"
          ? "server"
          : inputMode === "tts"
            ? "speech"
            : "voice-transcription"}
      <SidebarContribution id="workflow">
        <div class={workbench ? "workflow-sidebar-shell" : "contents"}>
          <WorkflowSidebar
            {session}
            workflow={inputMode === "file"
              ? "file"
              : inputMode === "tts"
                ? "tts"
                : "voice"}
            title={paneTitle}
            disabled={session.editor.saving}
            editing={quickSettingsDisabled}
            onAddConnection={addConnection}
            onOpenOptions={() => onOpenSettingsSection(workflowSection)}
            onOpenCleanup={onOpenProcessingSettings}
            onOpenVocabulary={() => onOpenSettingsSection("vocabulary")}
            onOpenDelivery={inputMode === "voice"
              ? onOpenDeliverySettings
              : () => onOpenSettingsSection(workflowSection)}
            onOpenOverlay={() => onOpenSettingsSection("overlay")}
            onOpenRuntime={openLocalRuntime}
          />
        </div>
      </SidebarContribution>
    {/if}
    <div class="body">
      <PaneHeader
        title={paneTitle}
        icon={inputMode === "file"
          ? FileAudioIcon
          : inputMode === "tts"
            ? Volume2Icon
            : MicIcon}
      >
        {#snippet actions()}
          <WorkflowSettingsButton
            label={inputMode === "file"
              ? "Audio file settings"
              : inputMode === "tts"
                ? "Text to speech settings"
                : "Voice settings"}
            disabled={!session.editor.draft || session.editor.saving}
            onclick={() =>
              inputMode === "file"
                ? onOpenServerSettings()
                : inputMode === "tts"
                  ? onOpenSpeechSettings()
                  : onOpenSettingsSection("voice-transcription")}
          />
        {/snippet}
      </PaneHeader>
      {#if inputMode === "voice" && session.editor.draft && !readiness?.initialSetup}
        <div class="transport-frame">
          <VoiceBar
            showSettings={false}
            status={session.dictation.status}
            {now}
            busy={fileWorking}
            availability={recordingAvailability}
            onCheckConnection={() =>
              session.editor.testAppliedConnection(Purpose.Voice)}
            connectionCheckDisabled={session.editor.saving ||
              !!session.editor.quickSettingsPending.length ||
              !!runtimeSettings?.configuration.recoveryRequired}
            microphone={microphoneLabel}
            toggleShortcut={runtimeSettings?.toggleShortcut ?? ""}
            onToggle={() => session.dictation.toggleRecording()}
            onCancel={() => session.dictation.cancel()}
            onOpenOptions={() => onOpenSettingsSection("voice-transcription")}
            optionsDisabled={session.editor.saving}
            onOpenSettings={() =>
              managed
                ? openLocalRuntime()
                : onOpenSettingsSection("voice-transcription")}
          />
        </div>
      {/if}
      {#if inputMode === "file" && session.editor.draft && !readiness?.initialSetup}
        <div class="transport-frame">
          <FileBar
            showSettings={false}
            status={session.files.status}
            choosing={session.files.choosing}
            starting={session.files.starting}
            cancelling={session.files.cancelling}
            clearing={session.files.clearing}
            blocked={voiceActive || ttsWorking
              ? "Finish the current job first"
              : ""}
            resettingStreaming={session.files.resettingStreaming}
            onChoose={() => session.files.chooseAudioFile()}
            onStart={() => session.files.startFileTranscription()}
            onCancel={() => session.files.cancelFileTranscription()}
            onClear={() => session.files.clearAudioFile()}
            onTryStreamingAgain={() => session.files.tryFileStreamingAgain()}
            optionsDisabled={session.editor.saving}
            onOpenSettings={onOpenServerSettings}
          />
        </div>
      {/if}
      {#if messages.length}<Notifications
          {messages}
          abovePlayback={inputMode === "tts"}
        />{/if}
      {#if !workbench && session.speech.status.source !== TTSSource.SourceCompose && session.speech.status.phase !== TTSPhase.Idle && session.speech.status.phase !== TTSPhase.Cancelled}
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
          showSettings={false}
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
          optionsDisabled={session.editor.saving}
        ></TextToSpeech>
      {:else}
        <WorkspaceSplit
          {hasHistory}
          historyCount={session.history.entries.length}
          working={voiceActive || fileWorking}
        >
          {#snippet output()}
            {#if active}<RuntimeOutputDrawer {instanceID} />{/if}
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
                      inputMode === "file"
                        ? Purpose.Transcription
                        : Purpose.Voice,
                    )}
                />
              {:else}
                <p class="text-[13px] text-muted-foreground">
                  No endpoint check has run for this workflow yet.
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={quickSettingsDisabled || session.editor.saving}
                  onclick={() =>
                    session.editor.testAppliedConnection(
                      inputMode === "file"
                        ? Purpose.Transcription
                        : Purpose.Voice,
                    )}>Check connection</Button
                >
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
                    if (inputMode === "file")
                      void session.files.clearAudioFile();
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
                ></CurrentResult>
              {/if}

              {#if session.editor.draft}
                {#if showReadiness && readiness}
                  <ReadinessPanel
                    embedded
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
                      else if (section === "shortcuts")
                        onOpenShortcutSettings();
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
                          showCleanup={false}
                          settings={runtimeSettings!}
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
                        />
                      {/if}
                    {/snippet}
                  </ReadinessPanel>
                {:else if !runtimeSettings?.historyEnabled}
                  <div
                    class="flex shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-1 border-t border-hairline px-3 py-1 text-xs text-muted-foreground"
                  >
                    <span class="min-w-0 basis-64 flex-1"
                      >History is off. Current results remain available until
                      you clear them or start again.</span
                    >
                    <Button
                      variant="ghost"
                      size="sm"
                      onclick={onOpenHistorySettings}>History settings</Button
                    >
                  </div>
                {/if}
              {:else}
                <div
                  class="flex min-h-0 flex-1 flex-col"
                  role="status"
                  aria-label="Loading workspace"
                  aria-busy="true"
                >
                  <div
                    class="flex max-w-xl flex-col gap-3 px-3 py-4"
                    aria-hidden="true"
                  >
                    <Skeleton class="h-3 w-3/4 rounded-sm" />
                    <Skeleton class="h-3 w-full rounded-sm" />
                    <Skeleton class="h-3 w-1/2 rounded-sm" />
                  </div>
                  <span class="sr-only">Loading workspace…</span>
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
  }
  .workspace {
    position: relative;
    display: flex;
    min-height: 0;
    flex: 1;
    overflow: hidden;
  }
  .body {
    display: flex;
    min-width: 0;
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
    overflow-y: auto;
  }
  .workflow-sidebar-shell {
    display: flex;
    width: 100%;
    height: 100%;
    min-height: 0;
  }
  .workflow-sidebar-shell :global(.workflow-settings) {
    width: 100%;
    min-height: 0;
  }
  .workflow-sidebar-shell :global(.srow) {
    flex-direction: row;
    align-items: center;
    gap: 0.5rem;
    padding-block: 0;
  }
  @container (max-width: 699px) {
    .workspace.fallback :global(.workflow-settings) {
      position: absolute;
      inset: 0 auto 0 0;
      z-index: 46;
      width: min(280px, 85%);
      background: var(--background);
      box-shadow: 8px 0 24px #0003;
    }
  }
  .settings-backdrop {
    position: absolute;
    inset: 0 0 0 min(280px, 85%);
    z-index: 45;
    background: #0005;
  }
</style>
