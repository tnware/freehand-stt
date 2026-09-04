<script lang="ts">
  import { onMount } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as WindowingService from "$bindings/windowing/service";
  import * as OverlayService from "$bindings/overlay/service";
  import type { PreviewRequest } from "$bindings/overlay";
  import type { ShortcutCaptureProgress } from "$bindings/input";
  import SettingsScreen from "$lib/components/settings/SettingsScreen.svelte";
  import ConfigurationRecoveryDialog from "$lib/components/settings/ConfigurationRecoveryDialog.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";
  import { SETTINGS_SECTIONS, type SettingsSectionID } from "$lib/navigation";
  import {
    FileTranscriptionPhase,
    State,
    type FileTranscriptionStatus,
    type Settings,
    type Status,
    type TTSStatus,
  } from "$lib/state";
  import { session } from "$lib/stores/session.svelte";
  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import { activeAppearanceMode } from "$lib/appearance";

  let active = $state<SettingsSectionID>("general");
  let discardSettingsOpen = $state(false);
  let navigationRef = $state<HTMLElement | null>(null);
  let windowVisible = $state(false);
  let overlayPreviewing = $state(false);
  let overlayPreviewRequestVersion = 0;

  function overlayPreviewRequest(settings: Settings): PreviewRequest {
    return {
      preferences: {
        layout: settings.overlayLayout,
        anchor: settings.overlayAnchor,
        visibility: settings.overlayVisibility,
        motion: settings.overlayMotion,
        surface: settings.overlaySurface,
        visualizer: settings.overlayVisualizer,
        sizePercent: settings.overlaySizePercent,
        opacityPercent: settings.overlayOpacityPercent,
        edgeOffset: settings.overlayTopOffset,
        glowPercent: settings.overlayGlowPercent,
      },
      toggleShortcut: settings.toggleShortcut,
      holdShortcut: settings.holdShortcut,
    };
  }

  $effect(() => {
    if (!overlayPreviewing || !session.settings) return;
    const version = ++overlayPreviewRequestVersion;
    void OverlayService.StartPreview(overlayPreviewRequest(session.settings)).catch((cause) => {
      if (version !== overlayPreviewRequestVersion) return;
      overlayPreviewing = false;
      session.reportFailure(String(cause));
    });
  });

  $effect(() => {
    document.documentElement.dataset.material = session.appliedSettings?.micaActive
      ? "mica"
      : "solid";
  });

  $effect(() => {
    const settings = session.appliedSettings;
    if (
      !windowVisible ||
      !settings ||
      settings.configuration.recoveryRequired ||
      session.sttConnectionChecked ||
      session.sttConnectionTesting
    ) return;
    void session.testConnection(settings, "", false);
  });

  function settingsSection(value: string): SettingsSectionID {
    return SETTINGS_SECTIONS.some((section) => section.id === value)
      ? (value as SettingsSectionID)
      : "general";
  }

  function focusActiveSection() {
    queueMicrotask(() => {
      navigationRef
        ?.querySelector<HTMLElement>(`[data-settings-section="${active}"]`)
        ?.focus();
    });
  }

  async function prepareSettings(section: string) {
    windowVisible = true;
    active = settingsSection(section);
    discardSettingsOpen = false;
    session.discardSettingsDraft();
    await session.load();
    focusActiveSection();
  }

  function startOverlayPreview() {
    overlayPreviewing = true;
  }

  function stopOverlayPreview() {
    const wasPreviewing = overlayPreviewing;
    overlayPreviewing = false;
    overlayPreviewRequestVersion++;
    if (!wasPreviewing) return;
    void OverlayService.StopPreview().catch((cause) => session.reportFailure(String(cause)));
  }

  function cleanUpSettings() {
    windowVisible = false;
    stopOverlayPreview();
    session.discardSettingsDraft();
    session.clearCredentialDraft();
    session.clearMessages();
    void shortcutCapture.cancel().finally(() => shortcutCapture.reset());
  }

  async function closeSettings() {
    discardSettingsOpen = false;
    cleanUpSettings();
    try {
      await WindowingService.HideSettings();
    } catch (cause) {
      session.reportFailure(String(cause));
    }
  }

  function requestSettingsClose() {
    if (session.settingsSaving) return;
    if (session.settingsDirty) {
      discardSettingsOpen = true;
      return;
    }
    void closeSettings();
  }

  function discardAndCloseSettings() {
    session.discardSettingsDraft();
    void closeSettings();
  }

  onMount(() => {
    setMode("system");

    const offOpen = Events.On("settings:open", (event: { data: string }) => {
      void prepareSettings(event.data);
    });
    const offClose = Events.On("settings:close-requested", requestSettingsClose);
    const offChanged = Events.On("settings:changed", (event: { data: Settings }) => {
      session.applySettingsSnapshot(event.data);
    });
    const offVisibility = Events.On("settings:visibility", (event: { data: boolean }) => {
      windowVisible = event.data;
    });
    const offStatus = Events.On("dictation:status", (event: { data: Status }) => {
      session.applyStatus(event.data);
      if (event.data.state !== State.Idle && event.data.state !== State.Failed) {
        overlayPreviewing = false;
        overlayPreviewRequestVersion++;
      }
      if (event.data.state === State.Idle || event.data.state === State.Failed) {
        void session.refreshHistory();
      }
    });
    const offFile = Events.On(
      "file-transcription:status",
      (event: { data: FileTranscriptionStatus }) => {
        const wasActive = [
          FileTranscriptionPhase.FileTranscriptionUploading,
          FileTranscriptionPhase.FileTranscriptionProcessing,
          FileTranscriptionPhase.FileTranscriptionStreaming,
          FileTranscriptionPhase.FileTranscriptionCancelling,
        ].includes(session.fileStatus.phase);
        session.applyFileStatus(event.data);
        const active = [
          FileTranscriptionPhase.FileTranscriptionUploading,
          FileTranscriptionPhase.FileTranscriptionProcessing,
          FileTranscriptionPhase.FileTranscriptionStreaming,
          FileTranscriptionPhase.FileTranscriptionCancelling,
        ].includes(event.data.phase);
        if (wasActive && !active) void session.refreshHistory();
      },
    );
    const offFileDelta = Events.On("file-transcription:delta", (event) => {
      if (session.applyFileDelta(event.data) === "gap") void session.refreshFileStatus();
    });
    const offTTS = Events.On("tts:status", (event: { data: TTSStatus }) => {
      session.applyTTSStatus(event.data);
    });
    const offShortcutCapture = Events.On(
      "shortcut:capture-progress",
      (event: { data: ShortcutCaptureProgress }) => shortcutCapture.applyProgress(event.data),
    );
    const offHide = Events.On("common:WindowHide", cleanUpSettings);
    void WindowingService.SettingsVisible()
      .then(async (visible) => {
        windowVisible = visible;
        if (visible) await prepareSettings("general");
        else await session.loadSettings();
        setMode(activeAppearanceMode(session.appliedSettings));
      })
      .catch((cause) => session.reportFailure(String(cause)));
    return () => {
      offOpen();
      offClose();
      offChanged();
      offVisibility();
      offStatus();
      offFile();
      offFileDelta();
      offTTS();
      offShortcutCapture();
      offHide();
      cleanUpSettings();
    };
  });
</script>

<ModeWatcher defaultMode="system" disableTransitions />

<ConfigurationRecoveryDialog {session} />

<div class="flex h-screen flex-col overflow-hidden bg-transparent text-foreground">
  <SettingsScreen
    {session}
    bind:active
    bind:navigationRef
    onClose={requestSettingsClose}
    {overlayPreviewing}
    onStartOverlayPreview={startOverlayPreview}
    onStopOverlayPreview={stopOverlayPreview}
  />
</div>

<Dialog.Root
  open={discardSettingsOpen}
  onOpenChange={(open) => (discardSettingsOpen = open)}
>
  <Dialog.Content class="gap-0 bg-dialog-surface p-0 shadow-xl ring-dialog-stroke sm:max-w-[420px]">
    <Dialog.Header class="border-b border-hairline px-5 py-4 pr-14">
      <Dialog.Title class="text-base font-semibold">Discard unsaved changes?</Dialog.Title>
      <Dialog.Description class="mt-1 text-[13px] leading-relaxed">
        Settings you changed in this window have not been applied. Closing now will restore the
        last saved configuration.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="border-t-0 px-5 py-4">
      <Button variant="outline" onclick={() => (discardSettingsOpen = false)}>Keep editing</Button>
      <Button variant="destructive" onclick={discardAndCloseSettings}>Discard changes</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
