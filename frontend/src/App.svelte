<script lang="ts">
  import { windowMaterial } from "$lib/platform";
  import { onMount, tick, untrack } from "svelte";
  import { Events, Window } from "@wailsio/runtime";
  import type { ConnectionManagerRequest } from "$bindings/windowing";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as BuildInfoService from "$bindings/buildinfo/service";
  import * as WindowingService from "$bindings/windowing/service";
  import ActivityRail from "$lib/components/shell/ActivityRail.svelte";
  import TitleBar from "$lib/components/shell/TitleBar.svelte";
  import XIcon from "@lucide/svelte/icons/x";
  import { Button } from "$lib/components/ui/button";
  import TitleBarMenu, {
    type TitleMenu,
  } from "$lib/components/shell/TitleBarMenu.svelte";
  import WorkbenchFrame from "$lib/components/shell/WorkbenchFrame.svelte";
  import WorkbenchPanel from "$lib/components/shell/WorkbenchPanel.svelte";
  import PlaybackBar from "$lib/components/home/PlaybackBar.svelte";
  import Notifications from "$lib/components/shell/Notifications.svelte";
  import type { Message } from "$lib/utils/messages";
  import {
    WorkbenchLayout,
    provideWorkbenchLayout,
  } from "$lib/workbench-layout.svelte";
  import HomeScreen from "$lib/components/home/HomeScreen.svelte";
  import StatusBar from "$lib/components/shell/StatusBar.svelte";
  import SettingsPane from "$lib/components/settings/SettingsPane.svelte";
  import HistoryPane from "$lib/components/history/HistoryPane.svelte";
  import RuntimesPane from "$lib/components/runtimes/RuntimesPane.svelte";
  import CommandPalette, {
    type Command,
  } from "$lib/components/shell/CommandPalette.svelte";
  import {
    PANES,
    isWorkflowPane,
    workflowBlockedReason,
    type PaneID,
    type WorkflowPane,
  } from "$lib/panes";
  import { SETTINGS_SECTIONS } from "$lib/navigation";
  import {
    GENERAL_SECTIONS,
    WORKFLOW_SECTIONS,
    configurationRealm,
  } from "$lib/configuration-realms";
  import { canToggleRecording, isRecording } from "$lib/utils/status";
  import { ShellNavigation } from "$lib/shell-navigation.svelte";
  import ConfigurationRecoveryDialog from "$lib/components/settings/ConfigurationRecoveryDialog.svelte";
  import type { SettingsSectionID } from "$lib/navigation";
  import {
    FileTranscriptionPhase,
    State,
    TTSPhase,
    TTSSource,
  } from "$lib/state";
  import { levels } from "$lib/stores/levels.svelte";
  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
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
  const layout = new WorkbenchLayout();
  provideWorkbenchLayout(layout);
  let layoutRestored = $state(false);
  onMount(() => {
    layout.restore();
    layoutRestored = true;
  });
  $effect(() => {
    if (layoutRestored) layout.save();
  });
  /** Non-workflow places. Null means a workflow chain is on screen. */
  let auxPane = $state<
    "connections" | "runtimes" | "history" | "settings" | null
  >(null);
  let inspectorOpen = $state(false);
  let inspectorRealm = $state<PaneID>("voice");
  let aboutOpen = $state(false);
  let configuration = $state<SettingsPane>();
  let alive = true;
  let navigationGeneration = 0;
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

  const activePane = $derived<PaneID>(
    auxPane === null ? (inputMode as PaneID) : auxPane,
  );
  const settingsOpen = $derived(
    auxPane === "settings" || auxPane === "connections",
  );
  const configurationOpen = $derived(settingsOpen || inspectorOpen);
  const configurationEditing = $derived(
    settingsOpen ||
      (inspectorOpen &&
        (session.editor.dirty ||
          session.editor.saving ||
          session.editor.quickSettingsPending.length > 0 ||
          session.editor.connectionDraft !== null ||
          shortcutCapture.capturing ||
          layout.notificationsPaused)),
  );
  const configurationSections = $derived(
    inspectorOpen
      ? inspectorRealm === "history"
        ? ["history" as const]
        : WORKFLOW_SECTIONS[
            isWorkflowPane(inspectorRealm) ? inspectorRealm : "voice"
          ]
      : auxPane === "connections"
        ? ["connections" as const]
        : GENERAL_SECTIONS,
  );
  const secondaryAvailable = $derived(
    auxPane === null || auxPane === "history",
  );
  const inspectorVisible = $derived(
    inspectorOpen &&
      inspectorRealm === activePane &&
      (layout.secondaryAvailable.current || layout.compactSecondaryOpen),
  );
  const secondaryShown = $derived(
    inspectorOpen
      ? inspectorVisible
      : auxPane === "history" && layout.secondaryVisible,
  );
  const bottomAvailable = $derived(
    auxPane !== "settings" && layout.bottomAvailable.current,
  );
  const composerVisible = $derived(auxPane === null && inputMode === "tts");
  const primaryAvailable = $derived(
    Boolean(layout.sidebars[auxPane ?? "workflow"]),
  );
  const messages = $derived.by(() => {
    const items: Message[] = [];
    if (layout.notificationsPaused) return items;
    if (session.messages.info)
      items.push({
        id: "info",
        tone: "info",
        source: "system",
        text: session.messages.info,
        onDismiss: () => session.messages.dismissInfo(),
      });
    const speechFailureVisible =
      session.speech.status.phase === TTSPhase.Failed &&
      session.messages.isSpeechFailure(session.speech.status.generation);
    if (session.messages.error && !speechFailureVisible && !layout.errorOwned)
      items.push({
        id: "error",
        tone: "error",
        source: "action",
        text: session.messages.error,
        onDismiss: () => session.messages.dismissError(),
      });
    if (session.messages.notice)
      items.push({
        id: "notice",
        tone: "success",
        source: "action",
        text: session.messages.notice,
        onDismiss: () => session.messages.dismissNotice(),
      });
    return items;
  });

  function selectPane(id: PaneID) {
    if (workflowBlockedReason(id, voiceActive, fileWorking)) return;
    navigationGeneration++;
    if (id === "settings") return openSettings("general");
    if (id === "connections") return openSettings("connections");
    leaveSettings(() => {
      if (isWorkflowPane(id)) {
        auxPane = null;
        inputMode = id;
      } else auxPane = id;
    });
  }

  function leaveSettings(action: () => void) {
    const finish = () => {
      inspectorOpen = false;
      layout.compactSecondaryOpen = false;
      navigation.done();
      action();
    };
    if (configurationOpen && configuration) configuration.requestClose(finish);
    else finish();
  }

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

  let commandsOpen = $state(false);
  const macOS = $derived(session.editor.applied?.platform === "darwin");

  // Everything the rail and the chain already expose, addressed by name. No
  // command here can do something the interface cannot.
  const commands = $derived<Command[]>([
    {
      id: "layout:primary",
      group: "Layout",
      label: "Toggle primary sidebar",
      keywords: "left sidebar show hide",
      disabled: !primaryAvailable,
      run: () => layout.togglePrimary(),
    },
    {
      id: "layout:bottom",
      group: "Layout",
      label: "Toggle bottom panel",
      keywords: "panel show hide recent output diagnostics",
      disabled: !bottomAvailable,
      run: () => layout.toggleBottom(),
    },
    {
      id: "layout:secondary",
      group: "Layout",
      label: "Toggle secondary sidebar",
      keywords: "right sidebar history details show hide",
      disabled: !secondaryAvailable,
      run: toggleSecondary,
    },
    ...PANES.map((pane) => ({
      id: `go:${pane.id}`,
      group: "Go to",
      label: pane.label,
      icon: pane.icon,
      keywords: "open show switch pane",
      disabled: !!workflowBlockedReason(pane.id, voiceActive, fileWorking),
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
      disabled:
        !session.editor.applied ||
        configurationEditing ||
        session.editor.saving ||
        session.editor.quickSettingsPending.length > 0 ||
        !!session.editor.applied.configuration.recoveryRequired,
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

  const fileSelectionBlocked = $derived(
    voiceActive ||
      fileWorking ||
      session.files.selectionBusy ||
      [TTSPhase.Generating, TTSPhase.Playing, TTSPhase.Paused].includes(
        session.speech.status.phase,
      ),
  );
  function chooseAudioFileFromMenu() {
    if (fileSelectionBlocked) return;
    navigationGeneration++;
    leaveSettings(() => {
      if (fileSelectionBlocked) return;
      auxPane = null;
      inputMode = "file";
      void session.files.chooseAudioFile();
    });
  }
  const titleMenus = $derived<TitleMenu[]>([
    {
      id: "file",
      label: "File",
      groups: [
        [
          {
            id: "file:open",
            label: "Open audio file…",
            disabled: fileSelectionBlocked,
            run: chooseAudioFileFromMenu,
          },
        ],
        [
          {
            id: "file:connections",
            label: "Connections",
            run: () => openSettings("connections"),
          },
          {
            id: "file:settings",
            label: "Settings",
            run: () => openSettings("general"),
          },
        ],
        [
          {
            id: "file:close",
            label: "Close window",
            shortcut: "Alt+F4",
            run: () =>
              void Window.Close().catch((cause) =>
                session.messages.fail(cause),
              ),
          },
        ],
      ],
    },
    {
      id: "view",
      label: "View",
      groups: [
        [
          {
            id: "view:commands",
            label: "Command palette…",
            shortcut: "Ctrl+K",
            run: () => {
              commandsOpen = true;
            },
          },
        ],
        commands.filter((command) => command.group === "Go to"),
        commands
          .filter((command) => command.group === "Layout")
          .map((command) => ({
            ...command,
            checked:
              command.id === "layout:primary"
                ? layout.primaryVisible && primaryAvailable
                : command.id === "layout:bottom"
                  ? bottomAvailable && layout.bottomVisible
                  : secondaryShown,
            shortcut:
              command.id === "layout:primary"
                ? "Ctrl+B"
                : command.id === "layout:bottom"
                  ? "Ctrl+J"
                  : "Ctrl+Alt+B",
          })),
      ],
    },
    {
      id: "help",
      label: "Help",
      groups: [[{ id: "help:about", label: "About Freehand", run: openAbout }]],
    },
  ]);

  $effect(() => {
    document.documentElement.dataset.material = windowMaterial(
      session.editor.applied,
    );
    const mode = activeAppearanceMode(session.editor.applied);
    untrack(() => setMode(mode));
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
    const offSettings = Events.On("settings:open", () => {
      void takeSettingsRequest();
    });
    const offTask = Events.On(
      "workspace:select-task",
      (event: { data: string }) => {
        if (isWorkflowPane(event.data)) selectPane(event.data);
      },
    );
    // Workspace shortcuts are local to this window; native dictation shortcuts
    // remain owned by Go.
    function commandKey(event: KeyboardEvent) {
      if (shortcutCapture.capturing || layout.notificationsPaused) return;
      const key = event.key.toLowerCase();
      if (!["k", "b", "j"].includes(key) || event.defaultPrevented) return;
      if (
        (macOS
          ? !event.metaKey || event.ctrlKey
          : !event.ctrlKey || event.metaKey) ||
        event.shiftKey ||
        event.repeat ||
        event.isComposing
      )
        return;
      if (event.altKey && key !== "b") return;
      if (key === "j" && !bottomAvailable) return;
      if (key === "b" && !event.altKey && !primaryAvailable) return;
      if (key === "b" && event.altKey && !secondaryAvailable) return;
      event.preventDefault();
      if (key === "k") commandsOpen = !commandsOpen;
      else if (key === "j") layout.toggleBottom();
      else if (event.altKey) toggleSecondary();
      else layout.togglePrimary();
    }
    window.addEventListener("keydown", commandKey);

    const offClose = Events.On("shell:close-requested", () => {
      navigationGeneration++;
      leaveSettings(() => {
        void Window.Hide()
          .then(hidden)
          .catch((cause) => {
            if (alive) session.messages.fail(cause);
          });
      });
    });
    const offHidden = Events.On("shell:hidden", hidden);
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
    void session
      .load()
      .catch((cause) => {
        if (alive) session.messages.fail(cause);
      })
      .then(() => {
        if (!alive) return;
        setMode(activeAppearanceMode(session.editor.applied));
        void WindowingService.ShellReady().catch((cause) => {
          if (alive) session.messages.fail(cause);
        });
      });
    return () => {
      alive = false;
      offSession();
      // The default session belongs to this WebView, not a component mount.
      // Svelte HMR remounts App while retaining its imported session module.
      if (session !== defaultSession) session.dispose();
      offLevel();
      offSecondInstance();
      offSettings();
      offTask();
      window.removeEventListener("keydown", commandKey);
      offClose();
      offHidden();
      offAboutVisibility();
      session.editor.clearCredentialDraft();
    };
  });

  function showSettings(
    sectionID: SettingsSectionID,
    preserveDraft = false,
    realmOverride?: WorkflowPane,
  ) {
    const realm = realmOverride ?? configurationRealm(sectionID, inputMode);
    if (workflowBlockedReason(realm, voiceActive, fileWorking)) return;
    navigationGeneration++;
    const origin = navigation.origin || inputMode;
    if (!preserveDraft) navigation.done();
    navigation.openSettings(sectionID, realm === "settings" ? "" : origin);
    inspectorOpen =
      realm !== "settings" && realm !== "connections" && realm !== "runtimes";
    inspectorRealm = realm;
    layout.compactPrimaryOpen = false;
    if (isWorkflowPane(realm)) {
      inputMode = realm;
      auxPane = null;
    } else auxPane = realm;
    if (inspectorOpen) {
      const generation = navigationGeneration;
      void tick().then(() => {
        if (generation === navigationGeneration && inspectorOpen)
          layout.compactSecondaryOpen = !layout.secondaryAvailable.current;
      });
    }
    if (
      preserveDraft &&
      session.editor.validationIssue?.section === sectionID
    ) {
      const generation = navigationGeneration;
      void tick().then(() => {
        if (generation === navigationGeneration)
          void configuration?.revealValidationIssue();
      });
    }
  }
  function openSettings(
    sectionID: SettingsSectionID = "general",
    realmOverride?: WorkflowPane,
  ) {
    navigationGeneration++;
    const realm = realmOverride ?? configurationRealm(sectionID, inputMode);
    if (workflowBlockedReason(realm, voiceActive, fileWorking)) return;
    if (
      configurationOpen &&
      configuration &&
      realm === activePane &&
      configurationSections.includes(sectionID)
    ) {
      configuration.selectSection(sectionID);
      if (inspectorOpen) {
        layout.compactPrimaryOpen = false;
        layout.compactSecondaryOpen = !layout.secondaryAvailable.current;
      }
      return;
    }
    leaveSettings(() => showSettings(sectionID, false, realmOverride));
  }
  function openWorkflowSettings(sectionID: SettingsSectionID) {
    const realm =
      isWorkflowPane(inputMode) &&
      WORKFLOW_SECTIONS[inputMode].includes(sectionID)
        ? inputMode
        : undefined;
    openSettings(sectionID, realm);
  }
  function revealConfigurationSection(section: SettingsSectionID) {
    const realm =
      inspectorOpen &&
      isWorkflowPane(inspectorRealm) &&
      WORKFLOW_SECTIONS[inspectorRealm].includes(section)
        ? inspectorRealm
        : undefined;
    showSettings(section, true, realm);
  }
  function showConnection(request: ConnectionManagerRequest) {
    navigationGeneration++;
    navigation.connection = null;
    navigation.openConnection(request, inputMode);
    inspectorOpen = false;
    layout.compactPrimaryOpen = false;
    layout.compactSecondaryOpen = false;
    auxPane = "connections";
  }
  function openConnection(request: ConnectionManagerRequest) {
    navigationGeneration++;
    if (configurationOpen && configuration) {
      configuration.openConnection(request);
      return;
    }
    showConnection(request);
  }
  function closeInspector(hideHistory = false) {
    navigationGeneration++;
    const finish = () => {
      inspectorOpen = false;
      layout.compactSecondaryOpen = false;
      if (hideHistory && auxPane === "history") layout.secondaryOpen = false;
      navigation.done();
      void tick().then(() =>
        document
          .querySelector<HTMLButtonElement>(
            'button[aria-label="Toggle secondary sidebar"]',
          )
          ?.focus(),
      );
    };
    if (inspectorOpen && configuration) configuration.requestClose(finish);
    else finish();
  }
  function toggleSecondary() {
    if (!secondaryAvailable) return;
    if (inspectorOpen) {
      if (inspectorVisible) closeInspector(true);
      else {
        layout.compactPrimaryOpen = false;
        layout.compactSecondaryOpen = true;
      }
    } else if (auxPane === "history") layout.toggleSecondary();
    else
      openSettings(
        WORKFLOW_SECTIONS[isWorkflowPane(inputMode) ? inputMode : "voice"][0],
      );
  }
  function dismissSecondary() {
    if (inspectorOpen) closeInspector(true);
    else {
      if (layout.secondaryAvailable.current) layout.secondaryOpen = false;
      layout.compactSecondaryOpen = false;
      void tick().then(() =>
        document
          .querySelector<HTMLButtonElement>(
            'button[aria-label="Toggle secondary sidebar"]',
          )
          ?.focus(),
      );
    }
  }
  function showHistoryDetails() {
    navigationGeneration++;
    const generation = navigationGeneration;
    const finish = () => {
      inspectorOpen = false;
      navigation.done();
      layout.compactPrimaryOpen = false;
      layout.secondaryOpen = true;
      layout.compactSecondaryOpen = !layout.secondaryAvailable.current;
      void tick().then(() => {
        if (generation === navigationGeneration && auxPane === "history")
          document.getElementById("workbench-secondary-sidebar")?.focus();
      });
    };
    if (inspectorOpen && configuration) configuration.requestClose(finish);
    else finish();
  }
  /** Explicit Done returns to the caller; inspectors close beside their workflow. */
  function closeSettings() {
    navigationGeneration++;
    if (inspectorOpen) {
      closeInspector();
      return;
    }
    const origin = navigation.origin;
    navigation.done();
    auxPane = null;
    if (isWorkflowPane(origin)) inputMode = origin;
  }

  function hidden() {
    if (!alive) return;
    navigationGeneration++;
    commandsOpen = false;
    layout.compactPrimaryOpen = false;
    layout.compactSecondaryOpen = false;
    if (configurationOpen) {
      const origin = navigation.origin;
      inspectorOpen = false;
      navigation.done();
      auxPane = null;
      if (isWorkflowPane(origin)) inputMode = origin;
    }
    session.editor.discardSettingsDraft();
    session.editor.cancelConnectionEdit();
    session.editor.clearCredentialDraft();
  }

  async function takeSettingsRequest() {
    const generation = navigationGeneration;
    try {
      const state = await WindowingService.TakeSettingsRequest();
      if (!alive || generation !== navigationGeneration || !state.pending)
        return;
      const accepted =
        configurationOpen && configuration
          ? configuration.acceptRequest(state.request)
          : navigation.acceptRequest(state.request, false);
      if (!accepted) return;
      navigationGeneration++;
      if (isWorkflowPane(state.request.origin))
        inputMode = state.request.origin;
      if (state.request.connection) {
        inspectorOpen = false;
        auxPane = "connections";
      } else showSettings(state.request.section as SettingsSectionID, true);
    } catch (cause) {
      if (alive) session.messages.fail(cause);
    }
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
    platform={session.editor.applied?.platform}
    onWindowError={(cause) => session.messages.fail(cause)}
    commandHint={macOS ? "⌘" : "Ctrl"}
    onOpenCommands={() => (commandsOpen = true)}
    primaryVisible={layout.primaryVisible && primaryAvailable}
    bottomVisible={bottomAvailable && layout.bottomVisible}
    secondaryVisible={secondaryShown}
    {secondaryAvailable}
    {bottomAvailable}
    bottomUnavailableReason={auxPane === "settings"
      ? "The bottom panel is hidden in Settings"
      : undefined}
    onTogglePrimary={primaryAvailable
      ? () => layout.togglePrimary()
      : undefined}
    onToggleBottom={() => layout.toggleBottom()}
    onToggleSecondary={toggleSecondary}
  >
    {#snippet menu()}<TitleBarMenu menus={titleMenus} />{/snippet}
  </TitleBar>

  <CommandPalette bind:open={commandsOpen} {commands} />
  {#if messages.length}<Notifications shell {messages} />{/if}

  <div class="flex min-h-0 flex-1">
    <ActivityRail
      pane={activePane}
      {voiceActive}
      {fileWorking}
      onSelect={selectPane}
    />

    <WorkbenchFrame
      {layout}
      area={auxPane ?? "workflow"}
      panelAvailable={auxPane !== "settings"}
      {secondaryShown}
      retainSecondary={inspectorOpen}
      onDismissSecondary={dismissSecondary}
    >
      {#snippet secondaryContent()}
        {#if auxPane === "history"}
          <div
            class="workbench-header gap-0 pl-0 pr-2"
            role="group"
            aria-label="History sidebar view"
          >
            <button
              type="button"
              class="workbench-tab"
              aria-pressed={!inspectorOpen}
              onclick={showHistoryDetails}>Details</button
            >
            <button
              type="button"
              class="workbench-tab"
              aria-pressed={inspectorOpen}
              onclick={() => openSettings("history")}>History settings</button
            >
            <span class="flex-1"></span>
            <Button
              variant="ghost"
              size="icon-xs"
              class="shrink-0"
              aria-label="Close secondary sidebar"
              title="Close sidebar"
              disabled={inspectorOpen && session.editor.saving}
              onclick={dismissSecondary}
              ><XIcon class="size-4" aria-hidden="true" /></Button
            >
          </div>
        {/if}
        {#if inspectorOpen}
          <SettingsPane
            bind:this={configuration}
            {session}
            {navigation}
            inspector
            onCloseSidebar={auxPane === "history"
              ? undefined
              : dismissSecondary}
            visible={inspectorVisible}
            sections={configurationSections}
            onReturn={() => closeInspector()}
            onOpenRuntimes={() => selectPane("runtimes")}
            onNavigateExternal={showSettings}
            onConnectionNavigate={showConnection}
            onRevealSection={revealConfigurationSection}
          />
        {:else if auxPane === "history" && layout.details}{@render layout.details()}{/if}
      {/snippet}
      {#snippet panel()}
        <WorkbenchPanel
          {session}
          {inputMode}
          settingsOpen={configurationEditing}
          onOpenHistorySettings={() => openSettings("history")}
        />
      {/snippet}
      {#snippet children()}
        <div
          class="flex min-w-0 flex-1 flex-col"
          style:display={auxPane === null ? "flex" : "none"}
        >
          <HomeScreen
            {session}
            {now}
            active={auxPane === null}
            bind:inputMode
            onOpenHistorySettings={() => openSettings("history")}
            onOpenServerSettings={() => openSettings("server")}
            onOpenProcessingSettings={() => openSettings("processing")}
            onOpenAudioSettings={() => openSettings("audio")}
            onOpenShortcutSettings={() => openSettings("shortcuts")}
            onOpenSpeechSettings={() => openSettings("speech")}
            onOpenGeneralSettings={() => openSettings("general")}
            onOpenDeliverySettings={() => openWorkflowSettings("general")}
            onOpenSettingsSection={openWorkflowSettings}
            onOpenConnection={openConnection}
            quickSettingsDisabled={configurationEditing}
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
            onOpenDetails={showHistoryDetails}
            detailsVisible={!inspectorOpen && secondaryShown}
          />
        {:else if auxPane !== null}
          <SettingsPane
            bind:this={configuration}
            {session}
            {navigation}
            sections={configurationSections}
            workbenchPage={auxPane === "connections"}
            onNavigateExternal={showSettings}
            onConnectionNavigate={showConnection}
            onRevealSection={revealConfigurationSection}
            onReturn={closeSettings}
            onOpenRuntimes={() => {
              navigation.done();
              auxPane = "runtimes";
            }}
          />
        {/if}
        {#if (session.speech.status.source !== TTSSource.SourceCompose || !composerVisible) && session.speech.status.phase !== TTSPhase.Idle && session.speech.status.phase !== TTSPhase.Cancelled}
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
            onOpenSettings={() => openSettings("speech")}
          />
        {/if}
      {/snippet}
    </WorkbenchFrame>
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
