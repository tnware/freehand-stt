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
  import { FileTranscriptionPhase, State } from "$lib/state";
  import { levels } from "$lib/stores/levels.svelte";
  import { session } from "$lib/stores/session.svelte";
  import { subscribeSessionEvents } from "$lib/stores/session-events";
  import { activeAppearanceMode } from "$lib/appearance";
  import { shouldAutomaticallyTestConnection } from "$lib/utils/connection";

  let settingsOpen = $state(false);
  let aboutOpen = $state(false);
  let inputMode = $state("voice");
  // The status strip carries the same release identity About shows, read from
  // the one build-info source rather than restated here.
  let version = $state("");

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
    document.documentElement.dataset.material = session.editor.applied
      ?.micaActive
      ? "mica"
      : "solid";
  });

  // A bounded metadata-only probe makes readiness real rather than requiring
  // the user to manufacture connection state manually. Failed checks remain
  // stable until an explicit retry or a confirmed STT profile change.
  $effect(() => {
    const settings = session.editor.applied;
    if (
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
    const offHide = Events.On("common:WindowHide", () => {
      session.editor.clearCredentialDraft();
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
    const offAboutVisibility = Events.On(
      "about:visibility",
      (event: { data: boolean }) => {
        aboutOpen = event.data;
      },
    );
    void BuildInfoService.Current()
      .then((info) => (version = info.version))
      .catch(() => (version = ""));
    void WindowingService.SettingsVisible()
      .then((visible) => (settingsOpen = visible))
      .catch((cause) => session.messages.reportFailure(String(cause)));
    void WindowingService.AboutVisible()
      .then((visible) => (aboutOpen = visible))
      .catch((cause) => session.messages.reportFailure(String(cause)));
    void session
      .load()
      .finally(() => setMode(activeAppearanceMode(session.editor.applied)));
    return () => {
      offSession();
      session.dispose();
      offLevel();
      offHide();
      offSecondInstance();
      offSettingsVisibility();
      offAboutVisibility();
      session.editor.clearCredentialDraft();
    };
  });

  function openSettings(sectionID: SettingsSectionID = "general") {
    session.messages.clear();
    void WindowingService.OpenSettings(sectionID).catch((cause) => {
      session.messages.reportFailure(String(cause));
    });
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
  class="flex h-screen flex-col overflow-hidden bg-transparent text-foreground"
>
  <AppHeader
    bind:inputMode
    settings={session.editor.applied ?? session.editor.draft}
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
    settings={session.editor.applied}
    connection={session.editor.connection}
    {version}
    {aboutOpen}
    onAbout={openAbout}
  />
</div>
