<script lang="ts">
  import { providerNotices } from "../../branding/providers/notices";
  import { onMount } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { RefreshCw } from "@lucide/svelte";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as BuildInfoService from "$bindings/buildinfo/service";
  import * as SettingsService from "$bindings/settings/service";
  import * as WindowingService from "$bindings/windowing/service";
  import * as UpdatesService from "$bindings/updates/service";
  import BrandMark from "$lib/components/shell/BrandMark.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import type { Info as BuildInfo } from "$bindings/buildinfo";
  import { State as UpdateState, type Status as UpdateStatus } from "$bindings/updates";
  import type { Settings } from "$lib/state";
  import { activeAppearanceMode } from "$lib/appearance";

  let settings = $state<Settings | null>(null);
  let build = $state<BuildInfo | null>(null);
  let closing = $state(false);
  let error = $state("");
  let updateStatus = $state<UpdateStatus | null>(null);

  let checkingForUpdates = $derived(updateStatus?.state === UpdateState.StateChecking);

  function updateSummary(status: UpdateStatus | null): string {
    if (!status) return "Loading update status…";
    switch (status.state) {
      case UpdateState.StateDevelopment:
        return "Update checks are available in packaged builds.";
      case UpdateState.StateDisabled:
        return "Automatic checks are off. You can still check manually.";
      case UpdateState.StateChecking:
        return "Checking GitHub Releases…";
      case UpdateState.StateAvailable:
        return status.latestVersion
          ? `Freehand ${status.latestVersion} is available.`
          : "A Freehand update is available.";
      case UpdateState.StateCurrent:
        return "Freehand is up to date.";
      case UpdateState.StateError:
        return "The last update check could not reach GitHub Releases.";
      default:
        return status.enabled ? "Automatic checks are on." : "Automatic checks are off.";
    }
  }

  $effect(() => {
    document.documentElement.dataset.material = settings?.micaActive ? "mica" : "solid";
  });

  async function closeAbout() {
    if (closing) return;
    closing = true;
    error = "";
    try {
      await WindowingService.HideAbout();
    } catch (cause) {
      error = String(cause);
    } finally {
      closing = false;
    }
  }

  async function checkForUpdates() {
    error = "";
    try {
      await UpdatesService.CheckForUpdates();
    } catch (cause) {
      error = String(cause);
    }
  }

  onMount(() => {
    setMode("system");
    const offChanged = Events.On("settings:changed", (event: { data: Settings }) => {
      settings = event.data;
      setMode(activeAppearanceMode(settings));
    });
    const offUpdates = Events.On("updates:status", (event: { data: UpdateStatus }) => {
      updateStatus = event.data;
    });
    void SettingsService.GetSettings()
      .then((current) => {
        settings = current;
        setMode(activeAppearanceMode(settings));
      })
      .catch((cause) => (error = String(cause)));
    void BuildInfoService.Current()
      .then((current) => (build = current))
      .catch((cause) => (error = String(cause)));
    void UpdatesService.Current()
      .then((current) => (updateStatus = current))
      .catch((cause) => (error = String(cause)));
    return () => {
      offChanged();
      offUpdates();
    };
  });
</script>

<ModeWatcher defaultMode="system" disableTransitions />

<div class="flex h-screen flex-col overflow-hidden bg-transparent text-foreground">
  <main class="min-h-0 flex-1 overflow-y-auto p-5" aria-label="About Freehand">
    <div class="mx-auto flex max-w-[560px] flex-col gap-4">
      {#if error}
        <p
          class="rounded-lg border border-destructive/35 bg-destructive/10 px-3 py-2 text-sm text-destructive"
          role="alert"
        >
          {error}
        </p>
      {/if}
      <SettingsCard>
        <div class="flex items-center gap-4 px-5 py-[19px]">
          <BrandMark size="large" />
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <p class="text-[15px] font-medium">{build?.productName || "Freehand"}</p>
              {#if build?.development}
                <Badge variant="secondary">Development</Badge>
              {/if}
            </div>
            <p class="mt-1 font-mono text-xs text-muted-foreground">
              Speech to text, anywhere you type.
            </p>
          </div>
          <span class="shrink-0 font-mono text-xs text-muted-foreground">
            {build?.version || "Loading…"}
          </span>
        </div>
        <div class="border-t border-hairline px-5 py-3">
          <p class="text-sm font-medium">Free forever. Open source.</p>
          <p class="mt-1 text-xs leading-relaxed text-muted-foreground">
            No Freehand subscription or account. Your chosen provider or hosting may have its own
            costs.
          </p>
        </div>
        {#if build}
          <div
            class="flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-2.5 font-mono text-[10.5px] text-muted-foreground"
          >
            <span>Windows {build.windowsVersion}</span>
            <span aria-hidden="true">·</span>
            <span>Wails {build.wailsVersion}</span>
            <span aria-hidden="true">·</span>
            <span>Go {build.goVersion}</span>
          </div>
        {/if}
        <div class="flex items-center gap-4 border-t border-hairline px-5 py-3">
          <div class="min-w-0 flex-1">
            <p class="text-xs font-medium text-foreground">Software updates</p>
            <p class="mt-0.5 text-[11px] leading-relaxed text-muted-foreground">
              {updateSummary(updateStatus)}
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            disabled={checkingForUpdates || updateStatus?.state === UpdateState.StateDevelopment}
            onclick={() => void checkForUpdates()}
          >
            <RefreshCw class={checkingForUpdates ? "size-3.5 animate-spin" : "size-3.5"} />
            Check now
          </Button>
        </div>
      </SettingsCard>

      <div class="flex flex-col gap-3 px-1 text-[13px] leading-relaxed text-muted-foreground">
        <p>
          Speech is captured locally, sent to the endpoint you configure, and inserted into the
          window that had focus when you started. If focus moved before the transcript came back,
          nothing is typed and the text waits for an explicit copy.
        </p>
        <p>
          Your API key is stored in Windows Credential Manager and is never returned to this screen.
          Recorded audio is deleted after every request, including failures and cancellation.
        </p>
      </div>
      <details class="rounded-lg border border-hairline px-4 py-3 text-xs text-muted-foreground">
        <summary class="cursor-pointer">Provider icon credits</summary>
        <p class="my-3">
          Brand marks identify providers; they do not imply endorsement. Some providers use neutral
          symbols.
        </p>
        {#each providerNotices as notice (notice.name)}
          <details class="my-2">
            <summary class="cursor-pointer">{notice.name}</summary>
            <pre
              class="mt-2 whitespace-pre-wrap break-words text-[10px] leading-relaxed">{notice.text}</pre>
          </details>
        {/each}
      </details>
    </div>
  </main>

  <footer class="flex shrink-0 justify-end border-t border-hairline bg-layer-fill px-5 py-3.5">
    <Button variant="outline" disabled={closing} onclick={() => void closeAbout()}>
      {closing ? "Closing…" : "Close"}
    </Button>
  </footer>
</div>
