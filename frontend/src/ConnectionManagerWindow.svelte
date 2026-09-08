<script lang="ts">
  import { onMount } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as WindowingService from "$bindings/windowing/service";
  import * as SettingsService from "$bindings/settings/service";
  import type { ConnectionManagerRequest } from "$bindings/windowing";
  import type { Settings } from "$lib/state";
  import { session } from "$lib/stores/session.svelte";
  import { activeAppearanceMode } from "$lib/appearance";
  import ConnectionsSection from "$lib/components/settings/sections/ConnectionsSection.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Dialog from "$lib/components/ui/dialog";

  let request = $state<ConnectionManagerRequest | null>(null);
  let visible = $state(false);
  let loading = $state(false);
  let discardOpen = $state(false);
  let closeAfterDiscard = $state(false);
  let revision = 0;

  function applyAppearance(settings: Settings) {
    setMode(activeAppearanceMode(settings));
    document.documentElement.dataset.material = settings.micaActive ? "mica" : "solid";
  }

  async function prepare() {
    if (visible || loading) return;
    const current = ++revision;
    loading = true;
    const timeout = setTimeout(() => {
      if (current !== revision) return;
      revision++;
      loading = false;
      session.messages.reportFailure(
        "Connections did not load. Try again after Freehand finishes starting.",
      );
    }, 10000);
    try {
      const state = await WindowingService.CurrentConnectionManager();
      if (current !== revision || !state.visible) return;
      const settings = await SettingsService.GetSettings();
      if (current !== revision) return;
      session.editor.applySettingsSnapshot(settings);
      applyAppearance(settings);
      request = state.request;
      visible = true;
      if (request.create) session.editor.beginConnection(undefined, request.purpose || undefined);
      else if (request.id) {
        const connection = session.editor.applied?.savedConnections.entries?.find(
          (c) => c.id === request?.id,
        );
        if (connection) session.editor.beginConnection(connection);
        else session.messages.reportFailure("This connection is no longer available.");
      }
    } catch (cause) {
      session.messages.reportFailure(String(cause));
    } finally {
      clearTimeout(timeout);
      if (current === revision) loading = false;
    }
  }

  function clear() {
    revision++;
    loading = false;
    visible = false;
    request = null;
    discardOpen = false;
    session.editor.cancelConnectionEdit();
    session.editor.clearCredentialDraft();
  }
  async function close() {
    clear();
    await WindowingService.HideConnectionManager();
  }
  function leave(closeWindow: boolean) {
    if (session.editor.saving || session.editor.managedConnectionTesting) return;
    if (session.editor.connectionDirty) {
      closeAfterDiscard = closeWindow;
      discardOpen = true;
    } else if (closeWindow) void close();
    else session.editor.cancelConnectionEdit();
  }
  function discard() {
    session.editor.cancelConnectionEdit();
    discardOpen = false;
    if (closeAfterDiscard) void close();
  }
  function saved() {
    session.editor.cancelConnectionEdit();
    if (request?.create && request.purpose) void close();
  }
  onMount(() => {
    setMode("system");
    const offs = [
      Events.On("connections:open", () => void prepare()),
      Events.On("connections:close-requested", () => leave(true)),
      Events.On("common:WindowHide", clear),
      Events.On("settings:changed", (event: { data: Settings }) => {
        session.editor.applySettingsSnapshot(event.data);
        applyAppearance(event.data);
      }),
    ];
    void prepare();
    return () => {
      for (const off of offs) off();
      clear();
      session.dispose();
    };
  });
</script>

<svelte:window
  onkeydown={(event) => {
    if (event.key === "Escape" && !discardOpen) {
      event.preventDefault();
      leave(true);
    }
  }}
/>
<ModeWatcher defaultMode="system" disableTransitions />
<div class="flex h-screen flex-col overflow-hidden bg-transparent text-foreground">
  <header
    class="flex shrink-0 items-center justify-between gap-4 border-b border-hairline px-5 py-4"
  >
    <div>
      <h1 class="text-base font-semibold">Connection Manager</h1>
      <p class="mt-1 text-xs text-muted-foreground">
        Servers and credentials shared by your speech workflows.
      </p>
    </div>
    <Button variant="ghost" disabled={session.editor.saving} onclick={() => leave(true)}
      >Close</Button
    >
  </header>
  <main class="min-h-0 flex-1 overflow-y-auto p-5" aria-busy={loading}>
    {#if loading}<p class="text-sm text-muted-foreground">Loading connections…</p>
    {:else if visible && session.editor.applied}
      <ConnectionsSection
        editor={session.editor}
        activateFor={request?.create && request.purpose ? request.purpose : undefined}
        error={session.messages.error}
        onBack={() => leave(false)}
        onSaved={saved}
        onOpenFeature={(purpose) => {
          void WindowingService.OpenSettings(
            purpose === "voice"
              ? "voice-transcription"
              : purpose === "stt"
                ? "server"
                : purpose === "cleanup"
                  ? "processing"
                  : "speech",
          );
        }}
      />
    {:else if session.messages.error}<p role="alert" class="text-sm text-destructive">
        {session.messages.error}
      </p>
      <Button class="mt-3" variant="outline" onclick={prepare}>Try again</Button>{/if}
  </main>
</div>
<Dialog.Root bind:open={discardOpen}>
  <Dialog.Content>
    <Dialog.Header
      ><Dialog.Title>Discard connection changes?</Dialog.Title><Dialog.Description
        >Your changes have not been saved.</Dialog.Description
      ></Dialog.Header
    >
    <Dialog.Footer
      ><Button variant="outline" onclick={() => (discardOpen = false)}>Keep editing</Button><Button
        variant="destructive"
        onclick={discard}>Discard changes</Button
      ></Dialog.Footer
    >
  </Dialog.Content>
</Dialog.Root>
