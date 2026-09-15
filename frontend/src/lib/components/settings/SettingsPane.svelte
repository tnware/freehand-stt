<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Events } from "@wailsio/runtime";
  import * as OverlayService from "$bindings/overlay/service";
  import type { ShortcutCaptureProgress } from "$bindings/input";
  import type { Session } from "$lib/stores/session.svelte";
  import {
    acceptConnectionClose,
    type ShellNavigation,
  } from "$lib/shell-navigation.svelte";
  import { Purpose } from "$bindings/savedconnection";
  import type {
    ConnectionManagerRequest,
    SettingsRequest,
  } from "$bindings/windowing";
  import SettingsScreen from "$lib/components/settings/SettingsScreen.svelte";
  import SettingsNav from "$lib/components/settings/SettingsNav.svelte";
  import ConnectionManagerWindow from "../../../ConnectionManagerWindow.svelte";
  import PendingChangesDialog from "$lib/components/settings/PendingChangesDialog.svelte";
  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import type { SettingsSectionID } from "$lib/navigation";

  let {
    session,
    navigation,
    onReturn,
    onOpenRuntimes = onReturn,
    inspector = false,
    visible = true,
    sections,
    workbenchPage = false,
    onNavigateExternal,
    onConnectionNavigate,
    onRevealSection,
  }: {
    session: Session;
    navigation: ShellNavigation;
    onReturn: () => void;
    onOpenRuntimes?: () => void;
    inspector?: boolean;
    visible?: boolean;
    sections?: SettingsSectionID[];
    workbenchPage?: boolean;
    onNavigateExternal?: (section: SettingsSectionID) => void;
    onConnectionNavigate?: (request: ConnectionManagerRequest) => void;
    onRevealSection?: (section: SettingsSectionID) => void;
  } = $props();
  let manager = $state<ConnectionManagerWindow>();
  let screen = $state<SettingsScreen>();
  let pending = $state<(() => void) | null>(null);
  let overlayPreviewing = $state(false);
  let revision = 0;
  let alive = true;
  let afterConnection = $state<(() => void) | null>(null);
  $effect(() => {
    if (!visible && overlayPreviewing) stopOverlayPreview();
  });
  $effect(() => {
    const settings = session.editor.draft;
    if (!visible || !overlayPreviewing || !settings) return;
    const current = ++revision;
    void OverlayService.StartPreview({
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
    }).catch((cause) => {
      if (current !== revision) return;
      overlayPreviewing = false;
      session.messages.fail(cause);
    });
  });
  function stopOverlayPreview() {
    revision++;
    if (overlayPreviewing)
      void OverlayService.StopPreview().catch((cause) =>
        session.messages.fail(cause),
      );
    overlayPreviewing = false;
  }
  function teardown() {
    stopOverlayPreview();
    session.editor.clearCredentialDraft();
    // Invalidate the old capture immediately. A late cancellation reply must
    // not reset a new capture started after navigating back to Shortcuts.
    const cancelling = shortcutCapture.cancel();
    shortcutCapture.reset();
    void cancelling.catch((cause) => {
      if (alive) session.messages.fail(cause);
    });
  }
  export function acceptRequest(request: SettingsRequest) {
    const blocked =
      (manager?.blocksReentry() ?? false) ||
      (screen?.blocksNavigation() ?? false) ||
      pending !== null ||
      afterConnection !== null ||
      session.editor.dirty ||
      session.editor.saving ||
      session.editor.managedConnectionTesting ||
      shortcutCapture.capturing;
    if (blocked) return false;
    stopOverlayPreview();
    return navigation.acceptRequest(request, false);
  }
  function guard(action: () => void) {
    if (screen?.blocksNavigation()) return;
    if (!alive || pending || afterConnection || session.editor.saving) return;
    if (manager)
      acceptConnectionClose(session.editor, () => {
        afterConnection = action;
        manager?.requestClose();
      });
    else if (session.editor.runtimeDirty) pending = action;
    else action();
  }
  export function requestClose(action = onReturn) {
    guard(action);
  }
  export async function revealValidationIssue() {
    if (alive) await screen?.revealValidationIssue();
  }
  export function openConnection(request: ConnectionManagerRequest) {
    guard(() => {
      teardown();
      if (onConnectionNavigate) {
        onConnectionNavigate(request);
        return;
      }
      navigation.connection = null;
      navigation.openConnection(request);
    });
  }
  export function selectSection(section: SettingsSectionID) {
    if (sections && !sections.includes(section) && onNavigateExternal)
      guard(() => onNavigateExternal(section));
    else if (section === "local-runtime") requestClose(onOpenRuntimes);
    else if (pending || afterConnection || session.editor.saving) return;
    else if (manager)
      guard(() => {
        navigation.connection = null;
        navigation.openSettings(section);
      });
    else if (section === "connections")
      guard(() => {
        teardown();
        navigation.openSettings(section);
      });
    else {
      stopOverlayPreview();
      navigation.active = section;
    }
  }
  async function resolve(save: boolean) {
    if (save && !(await session.editor.save())) {
      if (alive && session.editor.validationIssue) {
        pending = null;
        await tick();
        await revealValidationIssue();
      }
      return;
    }
    if (!alive) return;
    if (save) shortcutCapture.markSaved();
    else session.editor.discardSettingsDraft();
    const action = pending;
    pending = null;
    action?.();
  }
  function returned(purpose?: Purpose) {
    if (!alive) return;
    const next = afterConnection;
    afterConnection = null;
    navigation.returnFromConnection(purpose);
    if (next) next();
    else if (onNavigateExternal) onNavigateExternal(navigation.active);
  }
  onMount(() => {
    const off = Events.On(
      "shortcut:capture-progress",
      (event: { data: ShortcutCaptureProgress }) =>
        shortcutCapture.applyProgress(event.data),
    );
    return () => {
      alive = false;
      pending = null;
      afterConnection = null;
      off();
      teardown();
    };
  });
</script>

<div
  data-pane={workbenchPage
    ? "connections"
    : inspector
      ? "configuration"
      : "settings"}
  class="flex min-h-0 min-w-0 flex-1 overflow-hidden"
>
  {#if navigation.active === "connections"}
    {#if !workbenchPage}<SettingsNav
        active="connections"
        onSelect={selectSection}
        {sections}
      />{/if}
    {#key navigation.connection}
      <ConnectionManagerWindow
        bind:this={manager}
        {session}
        {workbenchPage}
        onCancelClose={() => (afterConnection = null)}
        onManageRuntime={() => selectSection("local-runtime")}
        initialRequest={navigation.connection ?? {
          id: "",
          purpose: Purpose.$zero,
          create: false,
        }}
        onReturn={(purpose) => {
          if (!alive) return;
          if (!navigation.connection && !afterConnection && !purpose)
            onReturn();
          else returned(purpose);
        }}
      />
    {/key}
  {:else}
    <SettingsScreen
      bind:this={screen}
      bind:session
      bind:active={navigation.active}
      {inspector}
      {visible}
      {sections}
      {onRevealSection}
      onClose={() => requestClose()}
      onOpenRuntimes={() => requestClose(onOpenRuntimes)}
      decisionOpen={pending !== null}
      onNavigate={selectSection}
      onOpenConnection={openConnection}
      onSaved={() => {
        if (alive && !inspector && navigation.saveReturnsToTask) onReturn();
      }}
      saveReturnsToTask={navigation.saveReturnsToTask}
      {overlayPreviewing}
      onStartOverlayPreview={() => (overlayPreviewing = true)}
      onStopOverlayPreview={stopOverlayPreview}
    />
  {/if}
</div>
<PendingChangesDialog
  open={pending !== null}
  busy={session.editor.saving}
  title="Save changes?"
  description="Your edits remain here until you save or discard them."
  error={session.messages.error}
  onKeepEditing={() => (pending = null)}
  onDiscard={() => resolve(false)}
  onSave={() => resolve(true)}
/>
