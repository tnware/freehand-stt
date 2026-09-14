<script lang="ts">
  import { windowMaterial } from "$lib/platform";
  import { onMount } from "svelte";
  import { Events, Window } from "@wailsio/runtime";
  import type { ConnectionManagerRequest } from "$bindings/windowing";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as BuildInfoService from "$bindings/buildinfo/service";
  import * as WindowingService from "$bindings/windowing/service";
  import ActivityRail from "$lib/components/shell/ActivityRail.svelte";
  import TitleBar from "$lib/components/shell/TitleBar.svelte";
  import HomeScreen from "$lib/components/home/HomeScreen.svelte";
  import StatusBar from "$lib/components/shell/StatusBar.svelte";
  import SettingsPane from "$lib/components/settings/SettingsPane.svelte";
  import HistoryPane from "$lib/components/history/HistoryPane.svelte";
  import RuntimesPane from "$lib/components/runtimes/RuntimesPane.svelte";
  import CommandPalette, {
    type Command,
  } from "$lib/components/shell/CommandPalette.svelte";
  import { PANES, paneByID, type PaneID } from "$lib/panes";
  import { SETTINGS_SECTIONS } from "$lib/navigation";
  import { canToggleRecording, isRecording } from "$lib/utils/status";
  import { ShellNavigation } from "$lib/shell-navigation.svelte";
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
  const navigation = new ShellNavigation();
  /** Non-workflow places. Null means a workflow chain is on screen. */
  let auxPane = $state<"runtimes" | "history" | "settings" | null>(null);
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
  // Endpoint freshness only needs a coarse tick, but the status bar now shows
  // a running clock, so pay for the fast timer only while capture is live.
  $effect(() => {
    const interval = voiceActive ? 1_000 : 30_000;
    const timer = setInterval(() => (now = Date.now()), interval);
    return () => clearInterval(timer);
  });

  // Runtimes is its own place on the rail but resolves to the settings pane's
  // runtime section, so there is one implementation of that screen, not two.
  const activePane = $derived<PaneID>(
    auxPane === null ? (inputMode as PaneID) : auxPane,
  );
  const settingsOpen = $derived(auxPane === "settings");

  function selectPane(id: PaneID) {
    if (id === "settings") return openSettings("general");
    if (id === "runtimes" || id === "history") {
      auxPane = id;
      return;
    }
    auxPane = null;
    inputMode = id;
  }

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

  let commandsOpen = $state(false);
  const macOS = $derived(session.editor.applied?.platform === "darwin");

  // Everything the rail and the chain already expose, addressed by name. No
  // command here can do something the interface cannot.
  const commands = $derived<Command[]>([
    ...PANES.map((pane) => ({
      id: `go:${pane.id}`,
      group: "Go to",
      label: pane.label,
      icon: pane.icon,
      keywords: "open show switch pane",
      run: () => selectPane(pane.id),
    })),
    {
      id: "dictation:toggle",
      group: "Dictation",
      label: isRecording(session.dictation.status)
        ? "Stop recording"
        : "Start recording",
      keywords: "record capture microphone talk",
      detail: session.editor.applied?.toggleShortcut ?? "",
      disabled: !canToggleRecording(session.dictation.status, fileWorking),
      run: () => void session.dictation.toggleRecording(),
    },
    {
      id: "dictation:copy",
      group: "Dictation",
      label: "Copy the current transcript",
      keywords: "clipboard paste",
      disabled: !session.dictation.status.canCopy,
      run: () => void session.dictation.copyPending(),
    },
    {
      id: "cleanup:toggle",
      group: "Cleanup",
      label: session.editor.applied?.postProcessing.enabled
        ? "Turn cleanup off"
        : "Turn cleanup on",
      keywords: "post processing tidy rewrite s1-mini",
      detail: session.editor.applied?.postProcessing.model ?? "",
      disabled: !session.editor.applied,
      run: () =>
        void session.editor.updateQuickSettings(
          {
            postProcessing: {
              enabled: !session.editor.applied?.postProcessing.enabled,
            },
          },
          "processing-enabled",
        ),
    },
    ...SETTINGS_SECTIONS.map((section) => ({
      id: `settings:${section.id}`,
      group: "Settings",
      label: section.label,
      icon: section.icon,
      keywords: section.blurb,
      run: () => openSettings(section.id),
    })),
  ]);


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
    let alive = true;
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
    // The tray's Settings entry is now main-window navigation, carrying the
    // section it asked for.
    const offSettings = Events.On(
      "settings:open",
      (event: { data: string }) => {
        openSettings((event.data || "general") as SettingsSectionID);
      },
    );
    const offTask = Events.On(
      "workspace:select-task",
      (event: { data: string }) => {
        if (["voice", "file", "tts"].includes(event.data)) {
          auxPane = null;
          inputMode = event.data;
        }
      },
    );
    // One accelerator, owned by the renderer: the palette is a window surface,
    // not a global shortcut competing with dictation.
    function commandKey(event: KeyboardEvent) {
      if (event.key !== "k" && event.key !== "K") return;
      if (!(event.ctrlKey || event.metaKey) || event.altKey) return;
      event.preventDefault();
      commandsOpen = !commandsOpen;
    }
    window.addEventListener("keydown", commandKey);

    const offClose = Events.On("shell:close-requested", () => {
      void Window.Hide().catch((cause) => {
        if (alive) session.messages.fail(cause);
      });
    });
    const offAboutVisibility = Events.On(
      "about:visibility",
      (event: { data: boolean }) => {
        aboutOpen = event.data;
      },
    );
    void BuildInfoService.Current()
      .then((info) => {
        if (alive) version = info.version;
      })
      .catch(() => {
        if (alive) version = "";
      });

    void WindowingService.AboutVisible()
      .then((visible) => {
        if (alive) aboutOpen = visible;
      })
      .catch((cause) => {
        if (alive) session.messages.reportFailure(String(cause));
      });
    void session.load().finally(() => {
      if (!alive) return;
      setMode(activeAppearanceMode(session.editor.applied));
      void WindowingService.ShellReady().catch((cause) => {
        if (alive) session.messages.fail(cause);
      });
    });
    return () => {
      alive = false;
      offSession();
      session.dispose();
      offLevel();
      offSecondInstance();
      offSettings();
      offTask();
      window.removeEventListener('keydown', commandKey);
      offClose();
      offAboutVisibility();
      session.editor.clearCredentialDraft();
    };
  });

  function openSettings(sectionID: SettingsSectionID = "general") {
    if (sectionID === "local-runtime") {
      auxPane = "runtimes";
      return;
    }
    navigation.openSettings(sectionID, inputMode);
    auxPane = "settings";
  }
  function openConnection(request: ConnectionManagerRequest) {
    navigation.openConnection(request, inputMode);
    auxPane = "settings";
  }
  /** Leaving configuration returns to the workflow that opened it. */
  function closeSettings() {
    const origin = navigation.origin;
    navigation.done();
    auxPane = null;
    if (origin && ["voice", "file", "tts"].includes(origin)) inputMode = origin;
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
  <TitleBar
    paneLabel={paneByID(activePane).label}
    commandHint={macOS ? "⌘" : "Ctrl"}
    onOpenCommands={() => (commandsOpen = true)}
  />

  <CommandPalette bind:open={commandsOpen} {commands} />

  <div class="flex min-h-0 flex-1">
    <ActivityRail
      pane={activePane}
      {voiceActive}
      {fileWorking}
      onSelect={selectPane}
    />

    <div
      class="flex min-w-0 flex-1 flex-col"
      style:display={auxPane === null ? "flex" : "none"}
    >
      <HomeScreen
        {session}
        {now}
        bind:inputMode
        onOpenHistorySettings={() => openSettings("history")}
        onOpenServerSettings={() => openSettings("server")}
        onOpenProcessingSettings={() => openSettings("processing")}
        onOpenAudioSettings={() => openSettings("audio")}
        onOpenShortcutSettings={() => openSettings("shortcuts")}
        onOpenSpeechSettings={() => openSettings("speech")}
        onOpenGeneralSettings={() => openSettings("general")}
        onOpenSettingsSection={openSettings}
        onOpenConnection={openConnection}
        quickSettingsDisabled={settingsOpen}
      />
    </div>

    {#if auxPane === "runtimes"}
      <RuntimesPane
        {session}
        workBusy={voiceActive || fileWorking}
        onOpenConnections={() => openSettings("connections")}
      />
    {:else if auxPane === "history"}
      <HistoryPane
        {session}
        onOpenHistorySettings={() => openSettings("history")}
      />
    {:else if auxPane !== null}
      <SettingsPane {session} {navigation} onReturn={closeSettings} />
    {/if}
  </div>

  <StatusBar
    dictation={session.dictation.status}
    {now}
    toggleShortcut={(session.editor.applied ?? session.editor.draft)
      ?.toggleShortcut ?? ""}
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
    cleanup={session.editor.applied?.postProcessing.enabled
      ? (session.editor.applied.postProcessing.model ?? "on")
      : ""}
    delivery={inputMode === "tts"
      ? "Play or save"
      : inputMode === "file"
        ? "Copy or save"
        : session.editor.applied?.autoInsert
          ? "Insert → focused app"
          : "Copy only"}
    commandHint={macOS ? "⌘K" : "Ctrl+K"}
    {version}
    {aboutOpen}
    onAbout={openAbout}
  />
</div>
