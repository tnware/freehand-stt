<script lang="ts">
  import { onMount } from "svelte";
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
  import ConnectionManagerWindow from "./ConnectionManagerWindow.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";
  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import type { SettingsSectionID } from "$lib/navigation";

  let {
    session,
    navigation,
    onReturn,
  }: { session: Session; navigation: ShellNavigation; onReturn: () => void } =
    $props();
  let manager = $state<ConnectionManagerWindow>();
  let pending = $state<(() => void) | null>(null);
  let overlayPreviewing = $state(false);
  let revision = 0;
  let alive = true;
  let afterConnection = $state<(() => void) | null>(null);
  $effect(() => {
    const settings = session.editor.draft;
    if (!overlayPreviewing || !settings) return;
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
    void shortcutCapture
      .cancel()
      .catch((cause) => session.messages.fail(cause))
      .finally(() => shortcutCapture.reset());
  }
  export function acceptRequest(request: SettingsRequest) {
    const blocked =
      (manager?.blocksReentry() ?? false) ||
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
  export function openConnection(request: ConnectionManagerRequest) {
    if (manager) return;
    guard(() => {
      teardown();
      navigation.openConnection(request);
    });
  }
  export function selectSection(section: SettingsSectionID) {
    if (manager)
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
    if (save && !(await session.editor.save())) return;
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
    next?.();
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

<div data-window="settings" class="flex min-h-0 flex-1 overflow-hidden">
  {#if navigation.active === "connections"}
    <SettingsNav active="connections" onSelect={selectSection} />
    {#key navigation.connection}
      <ConnectionManagerWindow
        bind:this={manager}
        {session}
        onCancelClose={() => (afterConnection = null)}
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
      {session}
      bind:active={navigation.active}
      onClose={() => requestClose()}
      decisionOpen={pending !== null}
      onNavigate={selectSection}
      onOpenConnection={openConnection}
      onSaved={() => {
        if (alive && navigation.saveReturnsToTask) onReturn();
      }}
      saveReturnsToTask={navigation.saveReturnsToTask}
      {overlayPreviewing}
      onStartOverlayPreview={() => (overlayPreviewing = true)}
      onStopOverlayPreview={stopOverlayPreview}
    />
  {/if}
</div>
<Dialog.Root
  open={pending !== null}
  onOpenChange={(open) => {
    if (!open && !session.editor.saving) pending = null;
  }}
>
  <Dialog.Content
    showCloseButton={!session.editor.saving}
    onEscapeKeydown={(event) => {
      if (session.editor.saving) event.preventDefault();
    }}
    onInteractOutside={(event) => {
      if (session.editor.saving) event.preventDefault();
    }}
  >
    <Dialog.Header
      ><Dialog.Title>Save changes?</Dialog.Title><Dialog.Description
        >Your edits remain here until you save or discard them.</Dialog.Description
      ></Dialog.Header
    >
    {#if session.messages.error}<p role="alert" class="text-destructive">
        {session.messages.error}
      </p>{/if}
    <Dialog.Footer>
      <Button
        variant="outline"
        disabled={session.editor.saving}
        onclick={() => (pending = null)}>Keep editing</Button
      >
      <Button
        variant="ghost"
        disabled={session.editor.saving}
        onclick={() => resolve(false)}>Discard</Button
      >
      <Button disabled={session.editor.saving} onclick={() => resolve(true)}
        >Save</Button
      >
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
