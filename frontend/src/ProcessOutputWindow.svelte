<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { ModeWatcher, setMode } from "mode-watcher";
  import * as Manager from "$bindings/managedruntime/manager";
  import * as Windowing from "$bindings/windowing/service";
  import * as SettingsService from "$bindings/settings/service";
  import { ProcessOutputState } from "$lib/stores/process-output.svelte";
  import { Button } from "$lib/components/ui/button";
  import { activeAppearanceMode } from "$lib/appearance";
  import { windowMaterial } from "$lib/platform";
  import type { Settings } from "$lib/state";

  const output = new ProcessOutputState(Manager);
  let paused = $state(false);
  let closing = $state(false);
  let error = $state("");
  let runtimeName = $state("");
  let viewport: HTMLPreElement | undefined = $state();

  function appearance(settings: Settings) {
    setMode(activeAppearanceMode(settings));
    document.documentElement.dataset.material = windowMaterial(settings);
  }
  async function close() {
    if (closing) return;
    closing = true;
    output.select("");
    try {
      await Windowing.CloseProcessOutput();
    } catch {
      error = "Could not close process output.";
    } finally {
      closing = false;
    }
  }
  async function followTail() {
    await tick();
    if (!paused && viewport) viewport.scrollTop = viewport.scrollHeight;
  }
  function toggleScroll() {
    paused = !paused;
    if (!paused) void followTail();
  }
  onMount(() => {
    let active = true;
    let revision = 0;
    let appearanceRevision = 0;
    setMode("system");
    async function refresh() {
      const request = ++revision;
      output.select("");
      paused = false;
      error = "";
      runtimeName = "";
      try {
        const current = await Windowing.CurrentProcessOutput();
        if (!active || request !== revision) return;
        output.select(current.instanceID);
        if (current.instanceID) {
          const [instances, providers] = await Promise.all([
            Manager.GetInstances(),
            Manager.GetProviders(),
          ]);
          if (!active || request !== revision) return;
          const instance = instances?.find(
            (row) => row.instance.id === current.instanceID,
          );
          runtimeName =
            providers?.find(
              (provider) => provider.id === instance?.instance.provider,
            )?.name ?? "";
        }
      } catch {
        if (active && request === revision)
          error = "Process output is unavailable.";
      }
    }
    function visibility() {
      output.visible = document.visibilityState === "visible";
    }
    visibility();
    document.addEventListener("visibilitychange", visibility);
    const subscriptions = [
      Events.On("process-output:changed", () => void refresh()),
      Events.On("settings:changed", (event: { data: Settings }) => {
        appearanceRevision++;
        appearance(event.data);
      }),
    ];
    void SettingsService.GetSettings()
      .then((settings) => {
        if (active && appearanceRevision === 0) appearance(settings);
      })
      .catch(() => {});
    void refresh();
    const timer = setInterval(() => {
      void output.poll().then(() => {
        if (active) void followTail();
      });
    }, 500);
    return () => {
      active = false;
      revision++;
      clearInterval(timer);
      document.removeEventListener("visibilitychange", visibility);
      for (const off of subscriptions) off();
      output.dispose();
    };
  });
</script>

<svelte:window
  onkeydown={(event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      void close();
    }
  }}
/>
<ModeWatcher defaultMode="system" disableTransitions />
<div
  class="flex h-screen flex-col overflow-hidden bg-transparent text-foreground"
>
  <header
    class="flex shrink-0 flex-wrap items-center justify-between gap-3 px-5 py-4"
  >
    <h1 class="text-base font-semibold">
      {runtimeName ? `${runtimeName} output` : "Process output"}
    </h1>
    <div class="flex items-center gap-1">
      <Button
        variant="ghost"
        size="sm"
        disabled={!output.accepted}
        onclick={toggleScroll}
        >{paused ? "Resume scrolling" : "Pause scrolling"}</Button
      >
      <Button
        variant="ghost"
        size="sm"
        disabled={!output.accepted || output.busy}
        onclick={() => void output.clear()}>Clear</Button
      >
      <Button
        variant="ghost"
        size="sm"
        disabled={closing}
        onclick={() => void close()}>Close</Button
      >
    </div>
  </header>
  <main
    class="flex min-h-0 flex-1 flex-col gap-3 px-5 pb-5"
    aria-label="Process output"
  >
    <div class="relative min-h-0 flex-1 border border-hairline bg-layer-fill">
      <pre
        bind:this={viewport}
        role="region"
        aria-label="Plain-text process output"
        aria-describedby={!output.accepted ? "output-consent" : undefined}
        class="absolute inset-0 overflow-auto p-3 font-mono text-xs whitespace-pre-wrap break-words">{#if output.accepted}{#each output.chunks as chunk (chunk.sequence)}<span
              >{chunk.text}</span
            >{/each}{/if}</pre>
      {#if !output.accepted}
        <div
          class="absolute inset-x-0 top-0 z-10 flex items-center justify-between gap-4 border-b border-hairline bg-background px-4 py-3"
        >
          <p
            id="output-consent"
            class="text-sm leading-relaxed text-muted-foreground"
          >
            Output may include transcripts, prompts, and file paths.
            <span class="block">Kept in memory; never saved automatically.</span
            >
          </p>
          <Button
            class="shrink-0"
            disabled={!output.instanceID || output.busy}
            onclick={() => void output.show()}>Show output</Button
          >
        </div>
      {/if}
    </div>
    <p
      role={error || output.error ? "alert" : undefined}
      class="h-4 shrink-0 truncate text-xs text-muted-foreground"
      class:text-destructive={!!(error || output.error)}
    >
      {error ||
        output.error ||
        (!output.instanceID
          ? "Open this viewer from a runtime to select its output."
          : paused
            ? "Scrolling paused; output collection continues."
            : output.truncated
              ? "Earlier output was cleared or discarded from the bounded buffer."
              : "Closing this window leaves the runtime running.")}
    </p>
  </main>
</div>
