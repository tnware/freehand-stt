<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as WindowingService from "$bindings/windowing/service";
  import { Session } from "$lib/stores/session.svelte";
  import { subscribeSessionEvents } from "$lib/stores/session-events";
  import { ShellNavigation } from "$lib/shell-navigation.svelte";
  import { activeAppearanceMode } from "$lib/appearance";
  import { windowMaterial } from "$lib/platform";
  import SettingsWindow from "./SettingsWindow.svelte";
  import ConfigurationRecoveryDialog from "$lib/components/settings/ConfigurationRecoveryDialog.svelte";

  let { session = new Session() }: { session?: Session } = $props();
  const navigation = new ShellNavigation();
  let configuration = $state<SettingsWindow>();
  let visible = $state(false);
  let alive = true;
  let generation = 0;
  let finishing = false;
  $effect(() => {
    document.documentElement.dataset.material = windowMaterial(
      session.editor.applied,
    );
    const mode = activeAppearanceMode(session.editor.applied);
    untrack(() => setMode(mode));
  });
  function hidden() {
    generation++;
    visible = false;
    navigation.done();
    session.editor.discardSettingsDraft();
    session.editor.cancelConnectionEdit();
    session.editor.clearCredentialDraft();
  }
  async function takeRequest() {
    const current = generation;
    try {
      const state = await WindowingService.TakeSettingsRequest();
      if (!alive || current !== generation || finishing || !state.pending)
        return;
      const accepted = configuration
        ? configuration.acceptRequest(state.request)
        : navigation.acceptRequest(state.request, false);
      if (accepted) visible = true;
    } catch (cause) {
      if (alive) session.messages.fail(cause);
    }
  }
  async function finish() {
    if (finishing || !alive) return;
    finishing = true;
    try {
      await WindowingService.FinishSettings(navigation.origin);
    } catch (cause) {
      if (alive) session.messages.fail(cause);
    } finally {
      finishing = false;
    }
  }
  onMount(() => {
    const offSession = subscribeSessionEvents(session, Events.On);
    const offOpen = Events.On("settings:open", () => {
      void takeRequest();
    });
    const offVisibility = Events.On(
      "settings:visibility",
      (event: { data: boolean }) => {
        if (!event.data) hidden();
        // Only accepted requests mount an editor; reveals cannot replace drafts.
      },
    );
    const offHide = Events.On("common:WindowHide", hidden);
    const offClose = Events.On("settings:close-requested", () => {
      if (configuration)
        configuration.requestClose(() => {
          void finish();
        });
      else void finish();
    });
    void session
      .load()
      .then(async () => {
        if (!alive) return;
        await WindowingService.SettingsReady();
        if (alive) await takeRequest();
      })
      .catch((cause) => {
        if (alive) session.messages.fail(cause);
      });
    return () => {
      alive = false;
      offSession();
      offOpen();
      offVisibility();
      offHide();
      offClose();
      hidden();
      session.dispose();
    };
  });
</script>

<ModeWatcher defaultMode="system" disableTransitions />
<div
  class="fixed inset-0 flex flex-col overflow-hidden bg-transparent text-foreground"
>
  {#if visible}
    <ConfigurationRecoveryDialog {session} />
    <SettingsWindow
      bind:this={configuration}
      {session}
      {navigation}
      onReturn={() => {
        void finish();
      }}
    />
  {/if}
</div>
