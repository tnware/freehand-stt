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
  import { Purpose } from "$bindings/savedconnection";
  import { State, type Settings } from "$lib/state";
  import { session } from "$lib/stores/session.svelte";
  import { subscribeSessionEvents } from "$lib/stores/session-events";
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
    if (!overlayPreviewing || !session.editor.draft) return;
    const version = ++overlayPreviewRequestVersion;
    void OverlayService.StartPreview(overlayPreviewRequest(session.editor.draft)).catch((cause) => {
      if (version !== overlayPreviewRequestVersion) return;
      overlayPreviewing = false;
      session.messages.reportFailure(String(cause));
    });
  });

  $effect(() => {
    document.documentElement.dataset.material = session.editor.applied?.micaActive
      ? "mica"
      : "solid";
  });

  function settingsSection(value: string): SettingsSectionID {
    return SETTINGS_SECTIONS.some((section) => section.id === value)
      ? (value as SettingsSectionID)
      : "general";
  }

  function focusActiveSection() {
    queueMicrotask(() => {
      navigationRef?.querySelector<HTMLElement>(`[data-settings-section="${active}"]`)?.focus();
    });
  }

  async function prepareSettings(section: string) {
    if (section === "connections") {
      await WindowingService.OpenConnectionManager({
        id: "",
        purpose: Purpose.$zero,
        create: false,
      });
      return;
    }
    if (windowVisible && session.editor.dirty) {
      active = settingsSection(section);
      session.messages.reportInfo(
        "Your unsaved settings are still here. Save or discard them before switching connections.",
      );
      focusActiveSection();
      return;
    }
    windowVisible = true;
    active = settingsSection(section);
    discardSettingsOpen = false;
    session.editor.discardSettingsDraft();
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
    void OverlayService.StopPreview().catch((cause) =>
      session.messages.reportFailure(String(cause)),
    );
  }

  function cleanUpSettings() {
    windowVisible = false;
    stopOverlayPreview();
    session.editor.discardSettingsDraft();
    session.editor.clearCredentialDraft();
    session.messages.clear();
    void shortcutCapture.cancel().finally(() => shortcutCapture.reset());
  }

  async function closeSettings() {
    discardSettingsOpen = false;
    cleanUpSettings();
    try {
      await WindowingService.HideSettings();
    } catch (cause) {
      session.messages.reportFailure(String(cause));
    }
  }

  function requestSettingsClose() {
    if (session.editor.saving) return;
    if (session.editor.dirty) {
      discardSettingsOpen = true;
      return;
    }
    void closeSettings();
  }

  function discardAndCloseSettings() {
    session.editor.discardSettingsDraft();
    void closeSettings();
  }

  onMount(() => {
    setMode("system");

    const offOpen = Events.On("settings:open", (event: { data: string }) => {
      void prepareSettings(event.data);
    });
    const offClose = Events.On("settings:close-requested", requestSettingsClose);
    const offVisibility = Events.On("settings:visibility", (event: { data: boolean }) => {
      windowVisible = event.data;
    });
    const offSession = subscribeSessionEvents(session, Events.On, (status) => {
      if (status.state !== State.Idle && status.state !== State.Failed) {
        overlayPreviewing = false;
        overlayPreviewRequestVersion++;
      }
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
        else await session.editor.load();
        setMode(activeAppearanceMode(session.editor.applied));
      })
      .catch((cause) => session.messages.reportFailure(String(cause)));
    return () => {
      offSession();
      session.dispose();
      offOpen();
      offClose();
      offVisibility();
      offShortcutCapture();
      offHide();
      cleanUpSettings();
    };
  });
</script>

<ModeWatcher defaultMode="system" disableTransitions />

<ConfigurationRecoveryDialog {session} />

<div
  data-window="settings"
  class="fixed inset-0 flex min-h-0 flex-col overflow-hidden bg-transparent text-foreground"
>
  <SettingsScreen
    {session}
    visible={windowVisible}
    bind:active
    bind:navigationRef
    onClose={requestSettingsClose}
    {overlayPreviewing}
    onStartOverlayPreview={startOverlayPreview}
    onStopOverlayPreview={stopOverlayPreview}
  />
</div>

<Dialog.Root open={discardSettingsOpen} onOpenChange={(open) => (discardSettingsOpen = open)}>
  <Dialog.Content class="gap-0 bg-dialog-surface p-0 shadow-xl ring-dialog-stroke sm:max-w-[420px]">
    <Dialog.Header class="border-b border-hairline px-5 py-4 pr-14">
      <Dialog.Title class="text-base font-semibold">Discard unsaved changes?</Dialog.Title>
      <Dialog.Description class="mt-1 text-[13px] leading-relaxed">
        Settings you changed in this window have not been applied. Closing now will restore the last
        saved configuration.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="border-t-0 px-5 py-4">
      <Button variant="outline" onclick={() => (discardSettingsOpen = false)}>Keep editing</Button>
      <Button variant="destructive" onclick={discardAndCloseSettings}>Discard changes</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
