<script lang="ts">
  import { onMount } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as BuildInfoService from "$bindings/buildinfo/service";
  import * as WindowingService from "$bindings/windowing/service";
  import AppHeader from "$lib/components/shell/AppHeader.svelte";
  import HomeScreen from "$lib/components/home/HomeScreen.svelte";
  import StatusStrip from "$lib/components/shell/StatusStrip.svelte";
  import ConfigurationRecoveryDialog from "$lib/components/settings/ConfigurationRecoveryDialog.svelte";
  import type { SettingsSectionID } from "$lib/navigation";
  import {
    FileTranscriptionPhase,
    State,
    type FileTranscriptionStatus,
    type Settings,
    type Status,
    type TTSStatus,
  } from "$lib/state";
  import { levels } from "$lib/stores/levels.svelte";
  import { session } from "$lib/stores/session.svelte";
  import { activeAppearanceMode } from "$lib/appearance";
  import { shouldAutomaticallyTestConnection } from "$lib/utils/connection";

  let settingsOpen = $state(false);
  let aboutOpen = $state(false);
  let inputMode = $state("voice");
  // The status strip carries the same release identity About shows, read from
  // the one build-info source rather than restated here.
  let version = $state("");

  const fileWorking = $derived(
    session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionUploading ||
      session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionProcessing ||
      session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionStreaming ||
      session.fileStatus.phase === FileTranscriptionPhase.FileTranscriptionCancelling,
  );
  const voiceActive = $derived(
    session.status.state !== State.Idle && session.status.state !== State.Failed,
  );

  $effect(() => {
    document.documentElement.dataset.material = session.appliedSettings?.micaActive
      ? "mica"
      : "solid";
  });

  // A bounded metadata-only probe makes readiness real rather than requiring
  // the user to manufacture connection state manually. Failed checks remain
  // stable until an explicit retry or a confirmed STT profile change.
  $effect(() => {
    const settings = session.appliedSettings;
    if (
      !settings ||
      settings.configuration.recoveryRequired ||
      !shouldAutomaticallyTestConnection(
        settings,
        session.sttConnectionChecked,
        session.sttConnectionTesting,
      )
    ) return;
    void session.testConnection(settings, "", false);
  });

  onMount(() => {
    setMode("system");

    const off = Events.On("dictation:status", (event: { data: Status }) => {
      const wasRecording = session.status.state === "recording";
      session.applyStatus(event.data);
      if (event.data.state === State.Idle || event.data.state === State.Failed) {
        void session.refreshHistory();
      }
      // Clear on both edges: a new recording starts from an empty meter, and
      // stopping does not leave the tail of the last one frozen on screen.
      if ((event.data.state === "recording") !== wasRecording) levels.reset();
    });
    // Go only sends these while recording and while this window is on screen,
    // so there is no stream to pay for the rest of the time.
    const offLevel = Events.On("dictation:level", (event: { data: number }) => {
      levels.push(event.data);
    });
    const offFile = Events.On(
      "file-transcription:status",
      (event: { data: FileTranscriptionStatus }) => {
        if (event.data.generation < session.fileStatus.generation) return;
        const wasActive = [
          FileTranscriptionPhase.FileTranscriptionUploading,
          FileTranscriptionPhase.FileTranscriptionProcessing,
          FileTranscriptionPhase.FileTranscriptionStreaming,
          FileTranscriptionPhase.FileTranscriptionCancelling,
        ].includes(session.fileStatus.phase);
        session.applyFileStatus(event.data);
        const isActive = [
          FileTranscriptionPhase.FileTranscriptionUploading,
          FileTranscriptionPhase.FileTranscriptionProcessing,
          FileTranscriptionPhase.FileTranscriptionStreaming,
          FileTranscriptionPhase.FileTranscriptionCancelling,
        ].includes(event.data.phase);
        if (wasActive && !isActive) {
          void session.refreshHistory();
        }
      },
    );
    const offFileDelta = Events.On("file-transcription:delta", (event) => {
      if (session.applyFileDelta(event.data) === "gap") void session.refreshFileStatus();
    });
    const offTTS = Events.On("tts:status", (event: { data: TTSStatus }) => {
      session.applyTTSStatus(event.data);
    });
    const offHide = Events.On("common:WindowHide", () => {
      session.clearCredentialDraft();
    });
    const offSecondInstance = Events.On("app:second-instance-revealed", () => {
      session.reportInfo(
        "Freehand is already running — that launch revealed this window instead of starting a second recorder.",
      );
    });
    const offSettingsVisibility = Events.On("settings:visibility", (event: { data: boolean }) => {
      settingsOpen = event.data;
    });
    const offSettingsChanged = Events.On("settings:changed", (event: { data: Settings }) => {
      session.applySettingsSnapshot(event.data);
    });
    const offAboutVisibility = Events.On("about:visibility", (event: { data: boolean }) => {
      aboutOpen = event.data;
    });
    void BuildInfoService.Current()
      .then((info) => (version = info.version))
      .catch(() => (version = ""));
    void WindowingService.SettingsVisible()
      .then((visible) => (settingsOpen = visible))
      .catch((cause) => session.reportFailure(String(cause)));
    void WindowingService.AboutVisible()
      .then((visible) => (aboutOpen = visible))
      .catch((cause) => session.reportFailure(String(cause)));
    void session.load().finally(() => setMode(activeAppearanceMode(session.appliedSettings)));
    return () => {
      off();
      offLevel();
      offFile();
      offFileDelta();
      offTTS();
      offHide();
      offSecondInstance();
      offSettingsVisibility();
      offSettingsChanged();
      offAboutVisibility();
      session.clearCredentialDraft();
    };
  });

  function openSettings(sectionID: SettingsSectionID = "general") {
    session.clearMessages();
    void WindowingService.OpenSettings(sectionID).catch((cause) => {
      session.reportFailure(String(cause));
    });
  }

  function openAbout() {
    session.clearMessages();
    void WindowingService.OpenAbout().catch((cause) => {
      session.reportFailure(String(cause));
    });
  }
</script>

<ModeWatcher defaultMode="system" disableTransitions />

<ConfigurationRecoveryDialog {session} />

<div class="flex h-screen flex-col overflow-hidden bg-transparent text-foreground">
  <AppHeader
    bind:inputMode
    settings={session.appliedSettings ?? session.settings}
    {voiceActive}
    {fileWorking}
    onSettings={() => openSettings()}
    {settingsOpen}
  />

  <HomeScreen
    {session}
    bind:inputMode
    onOpenHistorySettings={() => openSettings("history")}
    onOpenServerSettings={() => openSettings("server")}
    onOpenProcessingSettings={() => openSettings("processing")}
    onOpenAudioSettings={() => openSettings("audio")}
    onOpenShortcutSettings={() => openSettings("shortcuts")}
    onOpenSpeechSettings={() => openSettings("speech")}
    onOpenGeneralSettings={() => openSettings("general")}
    quickSettingsDisabled={settingsOpen}
  />

  <StatusStrip
    settings={session.appliedSettings}
    connection={session.connection}
    {version}
    {aboutOpen}
    onAbout={openAbout}
  />
</div>
