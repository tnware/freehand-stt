<script lang="ts">
  import { Purpose } from "$bindings/savedconnection";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import CurrentResult from "$lib/components/home/CurrentResult.svelte";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Button } from "$lib/components/ui/button";
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
          session.editor.connection,
          session.editor.devices,
          session.editor.devicesBusy,
          inputMode === "file" ? "file" : "voice",
        )
      : null,
  );
  let dismissedRecoveryKey = $state("");
  let taskHeight = $state(0);
  const showReadiness = $derived(
    Boolean(
      inputMode !== "tts" &&
      readiness &&
      readinessVisible(readiness, dismissedRecoveryKey) &&
      !voiceActive &&
      !fileWorking,
    ),
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
    if (session.messages.error) {
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
  class:with-history={runtimeSettings?.historyEnabled && inputMode !== "tts" && !showReadiness}
  aria-label="Freehand workspace"
>
  <div class="transport-frame">
    {#if session.editor.draft}
      {#if !showReadiness}
        {#if inputMode === "voice"}
          <TransportBar
            status={session.dictation.status}
            busy={fileWorking}
            toggleShortcut={session.editor.draft.toggleShortcut}
            model={runtimeSettings?.model ?? ""}
            processingModel={runtimeSettings?.postProcessing.model ?? ""}
            microphone={microphoneLabel}
            onToggle={() => session.dictation.toggleRecording()}
            onCancel={() => session.dictation.cancel()}
            onCopy={() => session.dictation.copyPending()}
            onOpenSettings={onOpenServerSettings}
          />
        {:else if inputMode === "file"}
          <AudioFileTranscription
            status={session.files.status}
            choosing={session.files.choosing}
            voiceActive={voiceActive || ttsWorking}
            onChoose={() => session.files.chooseAudioFile()}
            onStart={(stream) => session.files.startFileTranscription(stream)}
            onTryStreamingAgain={() => session.files.tryFileStreamingAgain()}
            onCancel={() => session.files.cancelFileTranscription()}
            onClear={() => session.files.clearAudioFile()}
          />
        {:else}
          <TextToSpeech
            bind:text={session.speech.draft}
            settings={runtimeSettings?.textToSpeech ?? session.editor.draft.textToSpeech}
            status={session.speech.status}
            unavailable={voiceActive || fileWorking}
            onSpeak={(text) => session.speech.speakText(text)}
            onPause={() => session.speech.pauseTTS()}
            onResume={() => session.speech.resumeTTS()}
            onRestart={() => session.speech.restartTTS()}
            onStop={() => session.speech.stopTTS()}
            onSave={() => session.speech.saveTTSAudio()}
            onClear={() => session.speech.clearTTSAudio()}
            onOpenSettings={onOpenSpeechSettings}
          />
        {/if}
      {/if}
    {:else}
      <Skeleton class="h-[132px] w-full rounded-none" />
    {/if}
  </div>
  <div class="body">
    <Notifications {messages} />
    {#if session.speech.status.source !== TTSSource.SourceCompose && session.speech.status.phase !== TTSPhase.Idle && session.speech.status.phase !== TTSPhase.Cancelled}
      <PlaybackBar
        status={session.speech.status}
        onPause={() => session.speech.pauseTTS()}
        onResume={() => session.speech.resumeTTS()}
        onRestart={() => session.speech.restartTTS()}
        onStop={() => session.speech.stopTTS()}
        onSave={() => session.speech.saveTTSAudio()}
        onClear={() => session.speech.clearTTSAudio()}
      />
    {/if}
    <div class="task-grid" style:--task-height={`${taskHeight}px`}>
      <div class="task-main" bind:clientHeight={taskHeight}>
        {#if session.editor.draft && inputMode !== "tts" && (!showReadiness || (inputMode === "file" ? session.files.status.transcript : session.dictation.status.transcript))}
          <CurrentResult
            message={inputMode === "voice" ? (statusMessage(session.dictation.status) ?? "") : ""}
            failed={inputMode === "voice" && isFailure(session.dictation.status)}
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
            onListen={inputMode === "file" &&
            runtimeSettings?.textToSpeech.enabled &&
            !voiceActive &&
            !fileWorking
              ? () => session.speech.listenFileTranscript()
              : undefined}
          />
        {/if}

        {#if session.editor.draft}
          {#if showReadiness && readiness}
            <ReadinessPanel
              {readiness}
              testing={session.editor.sttConnectionTesting}
              completing={session.editor.setupCompleting}
              onTestConnection={() => session.editor.testConnection(session.editor.applied, "")}
              onComplete={() => session.editor.completeSetup()}
              onDismiss={() => {
                dismissedRecoveryKey = readiness.recoveryKey;
              }}
              onOpenSettings={(section) => {
                if (section === "audio") onOpenAudioSettings();
                else if (section === "shortcuts") onOpenShortcutSettings();
                else onOpenServerSettings();
              }}
            />
          {:else}
            {#if inputMode === "tts"}
              <section
                class="rounded-lg border border-hairline bg-layer-fill p-4"
                aria-label="Text to speech connection"
              >
                <div class="flex flex-wrap items-center gap-3">
                  <label for="home-speech-connection" class="text-sm font-medium"
                    >Text to speech connection</label
                  >
                  <div class="min-w-48 flex-1">
                    <ConnectionSelect
                      id="home-speech-connection"
                      catalog={runtimeSettings!.savedConnections}
                      purpose={Purpose.Speech}
                      disabled={quickSettingsDisabled || session.editor.saving}
                      onChange={(change) => session.editor.changeConnection(change)}
                    />
                  </div>
                  <Button variant="outline" size="sm" onclick={onOpenSpeechSettings}
                    >Model and voice settings</Button
                  >
                </div>
                <p class="mt-2 text-xs text-muted-foreground">
                  Connection selection applies immediately. Your unsent text stays here while you
                  change settings.
                </p>
              </section>
            {:else}
              <details class="overflow-hidden rounded-lg border border-hairline bg-layer-fill">
                <summary class="cursor-pointer px-4 py-3 text-sm font-medium"
                  >{inputMode === "voice" ? "Dictation" : "Transcription"} settings
                  <span class="ml-2 text-xs font-normal text-muted-foreground"
                    >{runtimeSettings?.model || "Choose a connection and model"} · quick changes apply
                    immediately</span
                  >
                </summary>
                <div class="p-3">
                  <QuickSettings
                    showCapture={inputMode === "voice"}
                    settings={session.editor.applied ?? session.editor.draft}
                    devices={session.editor.devices}
                    processingProfiles={session.editor.processingProfiles}
                    connection={session.editor.connection}
                    processingConnection={session.editor.processingConnection}
                    sttStale={session.editor.sttConnectionStale ||
                      session.editor.connectionResultStale(Purpose.Transcription, runtimeSettings)}
                    processingStale={session.editor.processingConnectionStale ||
                      session.editor.connectionResultStale(Purpose.Cleanup, runtimeSettings)}
                    pending={session.editor.quickSettingsPending}
                    savedField={session.editor.quickSettingsSaved}
                    sttTesting={session.editor.sttConnectionTesting}
                    processingTesting={session.editor.processingConnectionTesting}
                    onChangeConnection={(change) => session.editor.changeConnection(change)}
                    onUpdate={(patch, field) => session.editor.updateQuickSettings(patch, field)}
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
                </div>
              </details>
              {#if !runtimeSettings?.historyEnabled}
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
            {/if}
          {/if}
        {:else}
          <div class="columns">
            <Skeleton class="h-full w-[372px] shrink-0 rounded-lg" />
            <Skeleton class="h-full flex-1 rounded-lg" />
          </div>
        {/if}
      </div>
      {#if runtimeSettings?.historyEnabled && inputMode !== "tts" && !showReadiness}
        <aside class="history-sidebar" aria-label="Recent history">
          <details class="overflow-hidden rounded-lg border border-hairline bg-layer-fill">
            <summary class="cursor-pointer px-4 py-3 text-sm font-medium"
              >Recent history · {session.history.entries.length}
              <span class="ml-2 text-xs font-normal text-muted-foreground"
                >In memory until you quit</span
              >
            </summary>
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
                onStopTTS={() => session.speech.stopTTS()}
                onSaveTTS={() => session.speech.saveTTSAudio()}
                onClearTTS={() => session.speech.clearTTSAudio()}
                ttsWorkspaceVisible={inputMode === "tts"}
                collapsible={false}
                playbackVisibleElsewhere
              />
            </div>
          </details>
        </aside>
      {/if}
    </div>
  </div>
</main>

<style>
  .home {
    container-type: inline-size;
    width: 100%;
    max-width: 80rem;
    margin-inline: auto;
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    overflow-y: auto;
  }
  .transport-frame {
    padding: 1rem 1rem 0;
  }
  .transport-frame :global(.transport) {
    border: 1px solid var(--hairline);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }
  .task-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 0.875rem;
    align-items: start;
  }
  .task-main {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 0.875rem;
  }
  .history-sidebar {
    --history-max-height: 24rem;
    min-width: 0;
  }
  .home.with-history {
    max-width: 96rem;
  }
  @container (min-width: 1120px) {
    .history-sidebar {
      --history-max-height: max(24rem, calc(var(--task-height) - 2.875rem));
    }
    .with-history .task-grid {
      grid-template-columns: minmax(0, 1.45fr) minmax(24rem, 1fr);
    }
  }
  .body {
    display: flex;
    flex: 0 0 auto;
    flex-direction: column;
    gap: 0.875rem;
    padding: 0.875rem 1rem 1rem;
  }
  .history-area {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .columns {
    display: flex;
    gap: 0.875rem;
  }
</style>
