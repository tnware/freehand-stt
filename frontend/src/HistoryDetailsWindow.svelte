<script lang="ts">
  import { onMount } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as HistoryService from "$bindings/history/service";
  import * as SettingsService from "$bindings/settings/service";
  import HistoryDetails from "$lib/components/history/HistoryDetails.svelte";
  import { Button } from "$lib/components/ui/button";
  import { activeAppearanceMode } from "$lib/appearance";
  import type { HistoryEntry, Settings } from "$lib/state";

  let entry = $state<HistoryEntry | null>(null);
  let loading = $state(true);
  let error = $state("");
  let closing = $state(false);

  function applyAppearance(settings: Settings) {
    setMode(activeAppearanceMode(settings));
    document.documentElement.dataset.material = settings.micaActive ? "mica" : "solid";
  }

  async function closeDetails() {
    if (closing) return;
    closing = true;
    error = "";
    try {
      await HistoryService.CloseDetails();
    } catch (cause) {
      error = String(cause);
    } finally {
      closing = false;
    }
  }

  onMount(() => {
    let active = true;
    let revision = 0;
    let appearanceRevision = 0;
    setMode("system");

    async function refresh(clear = false) {
      const request = ++revision;
      if (clear) {
        entry = null;
        loading = true;
      }
      error = "";
      try {
        const current = await HistoryService.CurrentDetails();
        if (active && request === revision) entry = current;
      } catch (cause) {
        if (active && request === revision) {
          entry = null;
          error = String(cause);
        }
      } finally {
        if (active && request === revision) loading = false;
      }
    }

    const subscriptions = [
      Events.On("history:details-changed", () => void refresh(true)),
      Events.On("dictation:status", () => void refresh()),
      Events.On("file-transcription:status", () => void refresh()),
      Events.On("settings:changed", (event: { data: Settings }) => {
        appearanceRevision++;
        applyAppearance(event.data);
        void refresh(!event.data.historyEnabled);
      }),
    ];
    void SettingsService.GetSettings()
      .then((settings) => {
        if (active && appearanceRevision === 0) applyAppearance(settings);
      })
      .catch((cause) => {
        if (active) error = String(cause);
      });
    void refresh();
    return () => {
      active = false;
      for (const off of subscriptions) off();
    };
  });
</script>

<svelte:window
  onkeydown={(event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      void closeDetails();
    }
  }}
/>

<ModeWatcher defaultMode="system" disableTransitions />

<div class="flex h-screen flex-col overflow-hidden bg-transparent text-foreground">
  {#if error}
    <p
      role="alert"
      class="shrink-0 border-b border-destructive/35 bg-destructive/10 px-5 py-3 text-sm text-destructive"
    >
      {error}
    </p>
  {/if}
  <main class="flex min-h-0 flex-1 flex-col" aria-label="Transcription details" aria-busy={loading}>
    {#if entry}
      {#key entry.id}
        <HistoryDetails {entry} />
      {/key}
    {:else}
      <div class="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center">
        <h1 class="text-base font-semibold">
          {loading ? "Loading details…" : "Details unavailable"}
        </h1>
        {#if !loading}
          <p class="max-w-sm text-sm text-muted-foreground">
            This run is no longer in session history. Open another entry to view its details.
          </p>
        {/if}
      </div>
    {/if}
  </main>
  <footer class="flex shrink-0 justify-end border-t border-hairline bg-layer-fill px-5 py-3.5">
    <Button variant="outline" disabled={closing} onclick={() => void closeDetails()}>
      {closing ? "Closing…" : "Close"}
    </Button>
  </footer>
</div>
