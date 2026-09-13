<script lang="ts">
  import { windowMaterial } from "$lib/platform";
  import { onMount } from "svelte";
  import { Events, Window } from "@wailsio/runtime";
  import type { ConnectionManagerRequest } from "$bindings/windowing";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as BuildInfoService from "$bindings/buildinfo/service";
  import * as WindowingService from "$bindings/windowing/service";
  import AppHeader from "$lib/components/shell/AppHeader.svelte";
  import HomeScreen from "$lib/components/home/HomeScreen.svelte";
  import StatusStrip from "$lib/components/shell/StatusStrip.svelte";
  import ConfigurationRecoveryDialog from "$lib/components/settings/ConfigurationRecoveryDialog.svelte";
  import type { SettingsSectionID } from "$lib/navigation";
  import { FileTranscriptionPhase, State } from "$lib/state";
  import { levels } from "$lib/stores/levels.svelte";
  import {
    session as defaultSession,
    type Session,
  } from "$lib/stores/session.svelte";
  import { subscribeSessionEvents } from "$lib/stores/session-events";
  import { activeAppearanceMode } from "$lib/appearance";
  import {
    shouldAutomaticallyTestConnection,
    taskConnectionStatus,
    taskConnectionDetails,
  } from "$lib/utils/connection";

  let { session = defaultSession }: { session?: Session } = $props();
  let settingsOpen = $state(false);
  let aboutOpen = $state(false);
  let inputMode = $state("voice");
  // The status strip carries the same release identity About shows, read from
  // the one build-info source rather than restated here.
  let version = $state("");
  let now = $state(Date.now());
  const footerConnection = $derived(
    taskConnectionDetails(inputMode, session.editor),
  );
  const footerStatus = $derived(
    taskConnectionStatus(inputMode, session.editor, now),
  );
  $effect(() => {
    const timer = setInterval(() => (now = Date.now()), 30_000);
    return () => clearInterval(timer);
  });

  const fileWorking = $derived(
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

  $effect(() => {
    document.documentElement.dataset.material = windowMaterial(
      session.editor.applied,
    );
  });

  // A bounded metadata-only probe makes readiness real rather than requiring
  // the user to manufacture connection state manually. Failed checks remain
  // stable until an explicit retry or a confirmed STT profile change.
  $effect(() => {
    const settings = session.editor.applied;
    if (
      inputMode !== "file" ||
      !settings ||
      settings.configuration.recoveryRequired ||
      !shouldAutomaticallyTestConnection(
        settings,
        session.editor.sttConnectionChecked,
        session.editor.sttConnectionTesting,
      )
    )
      return;
    void session.editor.testConnection(settings, "", false);
  });

  $effect(() => {
    const settings = session.editor.applied;
    if (
      inputMode !== "voice" ||
      !settings ||
      settings.configuration.recoveryRequired ||
      !settings.savedConnections.selected?.voice ||
      session.editor.voiceConnectionChecked ||
      session.editor.voiceConnectionTesting
    )
      return;
    void session.editor.testVoiceConnection();
  });

  onMount(() => {
    setMode("system");

    const offSession = subscribeSessionEvents(
      session,
      Events.On,
      (status, previous) => {
        if (
          (status.state === State.Recording) !==
          (previous.state === State.Recording)
        )
          levels.reset();
      },
    );
    // Go only sends these while recording and while this window is on screen,
    // so there is no stream to pay for the rest of the time.
    const offLevel = Events.On("dictation:level", (event: { data: number }) => {
      levels.push(event.data);
    });

    const offSecondInstance = Events.On("app:second-instance-revealed", () => {
      session.messages.reportInfo(
        "Freehand is already running — that launch revealed this window instead of starting a second recorder.",
      );
    });
    const offSettingsVisibility = Events.On(
      "settings:visibility",
      (event: { data: boolean }) => {
        settingsOpen = event.data;
      },
    );
    const offTask = Events.On(
      "workspace:select-task",
      (event: { data: string }) => {
        if (["voice", "file", "tts"].includes(event.data))
          inputMode = event.data;
      },
    );
    const offClose = Events.On("shell:close-requested", () => {
      void Window.Hide().catch((cause) => session.messages.fail(cause));
    });
    void WindowingService.SettingsVisible()
      .then((visible) => (settingsOpen = visible))
      .catch((cause) => session.messages.fail(cause));
    const offAboutVisibility = Events.On(
      "about:visibility",
      (event: { data: boolean }) => {
        aboutOpen = event.data;
      },
    );
    void BuildInfoService.Current()
      .then((info) => (version = info.version))
      .catch(() => (version = ""));

    void WindowingService.AboutVisible()
      .then((visible) => (aboutOpen = visible))
      .catch((cause) => session.messages.reportFailure(String(cause)));
    void session.load().finally(() => {
      setMode(activeAppearanceMode(session.editor.applied));
      void WindowingService.ShellReady().catch((cause) =>
        session.messages.fail(cause),
      );
    });
    return () => {
      offSession();
      session.dispose();
      offLevel();
      offSecondInstance();
      offSettingsVisibility();
      offTask();
      offClose();
      offAboutVisibility();
      session.editor.clearCredentialDraft();
    };
  });

  function openSettings(sectionID: SettingsSectionID = "general") {
    void WindowingService.OpenTaskSettings(sectionID, inputMode).catch(
      (cause) => session.messages.fail(cause),
    );
  }
  function openConnection(request: ConnectionManagerRequest) {
    void WindowingService.OpenTaskConnection(request, inputMode).catch(
      (cause) => session.messages.fail(cause),
    );
  }

  function openAbout() {
    session.messages.clear();
    void WindowingService.OpenAbout().catch((cause) => {
      session.messages.reportFailure(String(cause));
    });
  }
</script>

<ModeWatcher defaultMode="system" disableTransitions />

<ConfigurationRecoveryDialog {session} />

<div
  class="fixed inset-0 flex flex-col overflow-hidden bg-transparent text-foreground"
>
  <AppHeader
    bind:inputMode
    settings={session.editor.applied ?? session.editor.draft}
    {voiceActive}
    {fileWorking}
    onSettings={() => {
      void WindowingService.OpenSettings("general").catch((cause) =>
        session.messages.fail(cause),
      );
    }}
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
    connectionState={footerStatus}
    connectionDetails={footerConnection}
    disabled={session.editor.saving ||
      session.editor.quickSettingsPending.length > 0 ||
      !!session.editor.applied?.configuration.recoveryRequired}
    onCheck={() =>
      session.editor.testAppliedConnection(footerConnection.purpose)}
    onEdit={() => {
      openConnection({
        id: footerConnection.selected?.id ?? "",
        purpose: footerConnection.purpose,
        create: false,
      });
    }}
    onSettings={() =>
      openSettings(
        inputMode === "tts"
          ? "speech"
          : inputMode === "file"
            ? "server"
            : "voice-transcription",
      )}
    {version}
    {aboutOpen}
    onAbout={openAbout}
  />
</div>
