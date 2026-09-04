<script lang="ts">
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
  import { connectionSucceeded } from "$lib/utils/connection";
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
    session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionUploading ||
      session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionProcessing ||
      session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionStreaming ||
      session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionCancelling,
  );
  const voiceActive = $derived(
    session.status.state !== State.Idle && session.status.state !== State.Failed,
  );
  const ttsWorking = $derived(
    session.ttsStatus.phase === TTSPhase.Generating ||
      session.ttsStatus.phase === TTSPhase.Playing ||
      session.ttsStatus.phase === TTSPhase.Paused,
  );
  const runtimeSettings = $derived(session.appliedSettings ?? session.settings);
  const readiness = $derived(
    runtimeSettings
      ? appReadiness(runtimeSettings, session.connection, session.devices, session.devicesBusy)
      : null,
  );
  let dismissedRecoveryKey = $state("");
  const showReadiness = $derived(
    Boolean(
      readiness && readinessVisible(readiness, dismissedRecoveryKey) && !voiceActive && !fileWorking,
    ),
  );
  /*
   * Below the stacking width the rack and the transcript list stop being
   * columns and become two panes. Setup is something you touch when something
   * is wrong; the transcript is what you opened the window to read, so it is
   * the pane that opens first, and Setup carries a lamp when it needs you.
   */
  let homeWidth = $state(0);
  const stacked = $derived(homeWidth > 0 && homeWidth <= 959);
  let pane = $state<"transcripts" | "setup">("transcripts");
  const setupNeedsAttention = $derived(
    session.sttConnectionStale ||
      Boolean(session.connection && !connectionSucceeded(session.connection)),
  );

  const microphoneLabel = $derived.by(() => {
    const selectedID = runtimeSettings?.microphoneID ?? "";
    if (!selectedID) return "system default";
    return session.devices.find((device) => device.id === selectedID)?.name ?? "selected";
  });

  // Keep the active job visible. A hotkey can start voice capture while the
  // file tab is selected, so the UI must follow the work rather than hide it.
  $effect(() => {
    if (voiceActive) inputMode = "voice";
    else if (fileWorking) inputMode = "file";
    else if (ttsWorking && session.ttsStatus.source === TTSSource.SourceCompose) inputMode = "tts";
  });

  const messages = $derived.by(() => {
    const out: Message[] = [];
    // The status explanation comes first: it is about the screen you are
    // looking at, where the other two are about an action you took.
    const explanation = statusMessage(session.status);
    if (explanation) {
      out.push({
        id: "status",
        tone: isFailure(session.status) ? "error" : "info",
        source: "system",
        text: explanation,
      });
    }
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
    if (session.info) {
      out.push({
        id: "system-info",
        tone: "info",
        source: "system",
        text: session.info,
        onDismiss: () => session.dismissInfo(),
      });
    }
    if (session.error) {
      out.push({
        id: "error",
        tone: "error",
        source: "action",
        text: session.error,
        onDismiss: () => session.dismissError(),
      });
    }
    if (session.notice) {
      out.push({
        id: "notice",
        tone: "success",
        source: "action",
        text: session.notice,
        onDismiss: () => session.dismissNotice(),
      });
    }
    return out;
  });
</script>

<!--
  The transport spans the window and every other surface hangs below it: a rack
  of settings modules on the left, the transcript feed filling the rest. First
  run replaces both columns, because there is nothing to dictate into yet.
-->
<main class="home" aria-label="Freehand workspace" bind:clientWidth={homeWidth}>
  {#if session.settings}
    {#if !showReadiness}
      {#if inputMode === "voice"}
        <TransportBar
          status={session.status}
          busy={fileWorking}
          toggleShortcut={session.settings.toggleShortcut}
          model={runtimeSettings?.model ?? ""}
          processingModel={runtimeSettings?.postProcessing.model ?? ""}
          microphone={microphoneLabel}
          onToggle={() => session.toggleRecording()}
          onCancel={() => session.cancel()}
          onCopy={() => session.copyPending()}
          onOpenSettings={onOpenServerSettings}
        />
      {:else if inputMode === "file"}
        <AudioFileTranscription
          status={session.fileStatus}
          choosing={session.fileChoosing}
          voiceActive={voiceActive || ttsWorking}
          onChoose={() => session.chooseAudioFile()}
          onStart={(stream) => session.startFileTranscription(stream)}
          onTryStreamingAgain={() => session.tryFileStreamingAgain()}
          onCancel={() => session.cancelFileTranscription()}
          onClear={() => session.clearAudioFile()}
        />
      {:else}
        <TextToSpeech
          settings={runtimeSettings?.textToSpeech ?? session.settings.textToSpeech}
          status={session.ttsStatus}
          unavailable={voiceActive || fileWorking}
          onSpeak={(text) => session.speakText(text)}
          onPause={() => session.pauseTTS()}
          onResume={() => session.resumeTTS()}
          onRestart={() => session.restartTTS()}
          onStop={() => session.stopTTS()}
          onSave={() => session.saveTTSAudio()}
          onClear={() => session.clearTTSAudio()}
          onOpenSettings={onOpenSpeechSettings}
        />
      {/if}
    {/if}
  {:else}
    <Skeleton class="h-[132px] w-full rounded-none" />
  {/if}

  <div class="body">
    <Notifications {messages} />

    {#if session.settings}
      {#if showReadiness && readiness}
        <ReadinessPanel
          {readiness}
          testing={session.sttConnectionTesting}
          completing={session.setupCompleting}
          onTestConnection={() => session.testConnection(session.appliedSettings, "")}
          onComplete={() => session.completeSetup()}
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
        {#if stacked}
          <div class="switcher" role="tablist" aria-label="Workspace pane">
            <button
              type="button"
              role="tab"
              id="pane-tab-transcripts"
              class="pane-tab"
              aria-selected={pane === "transcripts"}
              aria-controls="pane-transcripts"
              onclick={() => (pane = "transcripts")}
            >
              Transcripts
              <span class="figure count">{session.history.length}</span>
            </button>
            <button
              type="button"
              role="tab"
              id="pane-tab-setup"
              class="pane-tab"
              aria-selected={pane === "setup"}
              aria-controls="pane-setup"
              onclick={() => (pane = "setup")}
            >
              Setup
              {#if setupNeedsAttention}
                <span
                  class="size-1.5 shrink-0 rounded-full bg-warning"
                  aria-label="Needs attention"
                ></span>
              {/if}
            </button>
          </div>
        {/if}

        <div class="columns">
          <div
            class="rack"
            id="pane-setup"
            role={stacked ? "tabpanel" : undefined}
            aria-labelledby={stacked ? "pane-tab-setup" : undefined}
            hidden={stacked && pane !== "setup"}
          >
            <QuickSettings
              settings={session.appliedSettings ?? session.settings}
              devices={session.devices}
              processingProfiles={session.processingProfiles}
              connection={session.connection}
              processingConnection={session.processingConnection}
              sttStale={session.sttConnectionStale}
              processingStale={session.processingConnectionStale}
              pending={session.quickSettingsPending}
              savedField={session.quickSettingsSaved}
              sttTesting={session.sttConnectionTesting}
              processingTesting={session.processingConnectionTesting}
              onUpdate={(patch, field) => session.updateQuickSettings(patch, field)}
              onTestConnection={() => session.testConnection(session.appliedSettings, "")}
              onTestProcessingConnection={() =>
                session.testPostProcessingConnection(session.appliedSettings, "")}
              disabled={quickSettingsDisabled}
              {onOpenServerSettings}
              {onOpenProcessingSettings}
              {onOpenAudioSettings}
              onOpenDeliverySettings={onOpenGeneralSettings}
            />
          </div>

          <div
            class="feed"
            id="pane-transcripts"
            role={stacked ? "tabpanel" : undefined}
            aria-labelledby={stacked ? "pane-tab-transcripts" : undefined}
            hidden={stacked && pane !== "transcripts"}
          >
            <HistoryPanel
              enabled={session.appliedSettings?.historyEnabled ?? false}
              entries={session.history}
              fileStatus={session.fileStatus}
              fileHistoryGeneration={session.fileHistoryGeneration}
              onOpenSettings={onOpenHistorySettings}
              onCopy={(id) => session.copyHistoryEntry(id)}
              onCopyVersion={(id, version) => session.copyHistoryEntryVersion(id, version)}
              onDelete={(id) => session.deleteHistoryEntry(id)}
              onCopyFile={() => session.copyFileTranscript()}
              ttsEnabled={runtimeSettings?.textToSpeech.enabled ?? false}
              ttsAvailable={!voiceActive && !fileWorking}
              ttsStatus={session.ttsStatus}
              onListen={(id, version) => session.listenHistoryEntry(id, version)}
              onListenFile={() => session.listenFileTranscript()}
              onPauseTTS={() => session.pauseTTS()}
              onResumeTTS={() => session.resumeTTS()}
              onRestartTTS={() => session.restartTTS()}
              onStopTTS={() => session.stopTTS()}
              onSaveTTS={() => session.saveTTSAudio()}
              onClearTTS={() => session.clearTTSAudio()}
              ttsWorkspaceVisible={inputMode === "tts"}
              collapsible={!stacked}
            />
          </div>
        </div>
      {/if}
    {:else}
      <div class="columns">
        <Skeleton class="h-full w-[372px] shrink-0 rounded-lg" />
        <Skeleton class="h-full flex-1 rounded-lg" />
      </div>
    {/if}
  </div>
</main>

<style>
  /* The container is the padding-free wrapper, so the query measures the
     window rather than the window minus gutters. */
  .home {
    container-type: inline-size;
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    overflow: hidden;
  }
  .body {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    padding: 0.875rem 1rem 1rem;
  }
  .columns {
    display: flex;
    min-height: 0;
    flex: 1;
    gap: 0.875rem;
  }
  /*
   * The rack is the fixed column so the feed absorbs every pixel the window
   * gains, and it scrolls on its own: at the 560px minimum height the four
   * modules are taller than the space below the transport.
   */
  .rack {
    display: flex;
    width: 23.25rem;
    flex-shrink: 0;
    min-height: 0;
    min-width: 0;
    flex-direction: column;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
  .feed {
    display: flex;
    min-height: 0;
    min-width: 0;
    flex: 1;
    flex-direction: column;
  }

  /*
   * Narrow, the two columns become two panes behind a switcher instead of a
   * stack. Stacked, the transcript list sits under four rack modules: the
   * thing you opened the window to read is the thing you have to scroll past
   * everything else to reach. As panes they are peers, and each one still owns
   * the full height and scrolls on its own.
   */
  @container (max-width: 959px) {
    /* The column direction is the fallback for the frame before the width
       binding reports: without it the two panes would briefly sit side by side
       in a 560px window. */
    .columns {
      flex-direction: column;
    }
    .rack {
      width: auto;
      flex: 1;
    }
  }

  .switcher {
    display: flex;
    flex: 0 0 auto;
    gap: 0.25rem;
    padding-bottom: 0.625rem;
  }
  .pane-tab {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    height: 1.875rem;
    padding: 0 0.75rem;
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    font-size: 0.781rem;
    color: var(--muted-foreground);
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .pane-tab:hover {
    background-color: var(--subtle-fill-hover);
    color: var(--foreground);
  }
  .pane-tab[aria-selected="true"] {
    border-color: var(--card-stroke);
    background-color: var(--card);
    box-shadow: var(--lift);
    color: var(--foreground);
    font-weight: 500;
  }
  .pane-tab:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 1px;
  }
  .count {
    font-size: 0.594rem;
    color: var(--ink-quiet);
  }
  /* Both panes stay mounted so switching keeps scroll position and in-flight
     drafts; the attribute has to beat the column's own display rule. */
  .rack[hidden],
  .feed[hidden] {
    display: none;
  }
  @container (max-width: 559px) {
    .body {
      padding: 0.75rem 0.75rem 0.875rem;
    }
  }
</style>
