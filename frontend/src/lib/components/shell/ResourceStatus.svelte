<script lang="ts">
  import { onMount } from "svelte";
  import { Events, Window } from "@wailsio/runtime";
  import { SettingsVisible } from "$bindings/windowing/service";
  import { CPUState } from "$bindings/resources";
  import ActivityIcon from "@lucide/svelte/icons/activity";
  import * as Popover from "$lib/components/ui/popover";
  import ResourceGPU from "./ResourceGPU.svelte";
  import {
    formatMemory,
    gpuPercent,
    gpuMemoryPercent,
    memoryPercent,
    type ResourceState,
  } from "$lib/stores/resources.svelte";

  let { resources }: { resources: ResourceState } = $props();
  let open = $state(false);
  const sample = $derived(resources.snapshot);
  const cpu = $derived(
    sample?.cpuState === CPUState.CPUReady ? sample.cpuPercent : null,
  );
  const memory = $derived(memoryPercent(sample));
  const cpuLabel = $derived(cpu === null ? "—" : `${Math.round(cpu)}%`);
  const memoryLabel = $derived(
    memory === null ? "—" : `${Math.round(memory)}%`,
  );
  const highCPU = $derived(cpu !== null && cpu >= 90);
  const highMemory = $derived(memory !== null && memory >= 90);
  const gpu = $derived(gpuPercent(sample));
  const gpuLabel = $derived(gpu === null ? "—" : `${Math.round(gpu)}%`);
  const highGPU = $derived(
    (gpu !== null && gpu >= 90) ||
      (sample?.gpus ?? []).some((row) => (gpuMemoryPercent(row) ?? 0) >= 90),
  );

  onMount(() => {
    let alive = true;
    let revision = 0;
    const hidden = () => {
      revision++;
      open = false;
      resources.setVisible(false);
    };
    const readVisibility = async () => {
      const request = ++revision;
      if (document.hidden) return hidden();
      try {
        const [shown, minimized] = await Promise.all([
          SettingsVisible(),
          Window.IsMinimised(),
        ]);
        if (!alive || request !== revision) return;
        const visible = shown && !minimized && !document.hidden;
        resources.setVisible(visible);
        if (!visible) open = false;
      } catch {
        if (alive && request === revision) hidden();
      }
    };
    const visibilityChanged = () => {
      if (document.hidden) hidden();
      else void readVisibility();
    };
    const off = [
      Events.On("shell:hidden", hidden),
      Events.On(Events.Types.Common.WindowHide, hidden),
      Events.On(Events.Types.Common.WindowMinimise, hidden),
      ...[
        Events.Types.Common.WindowShow,
        Events.Types.Common.WindowRestore,
        Events.Types.Common.WindowUnMinimise,
        Events.Types.Common.WindowFocus,
      ].map((event) => Events.On(event, () => void readVisibility())),
    ];
    document.addEventListener("visibilitychange", visibilityChanged);
    void readVisibility();
    return () => {
      alive = false;
      hidden();
      for (const unsubscribe of off) unsubscribe();
      document.removeEventListener("visibilitychange", visibilityChanged);
    };
  });
</script>

<Popover.Root bind:open>
  <Popover.Trigger
    aria-label={`This computer: CPU ${cpuLabel}, RAM ${memoryLabel}, GPU ${gpuLabel}`}
    title="This computer’s resource use"
    class="flex h-6 shrink-0 items-center gap-2 px-1.5 text-xs hover:bg-subtle-fill-hover aria-expanded:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring {highCPU ||
    highMemory ||
    highGPU
      ? 'text-warning'
      : ''}"
  >
    <ActivityIcon class="size-3 shrink-0" />
    <span class="resource-values figure flex gap-3">
      <span class="inline-flex w-[8ch] justify-between"
        ><span>CPU</span><span>{cpuLabel}</span></span
      >
      <span class="inline-flex w-[8ch] justify-between"
        ><span>RAM</span><span>{memoryLabel}</span></span
      >
      <span class="inline-flex w-[8ch] justify-between"
        ><span>GPU</span><span>{gpuLabel}</span></span
      >
    </span>
  </Popover.Trigger>
  <Popover.Content
    side="top"
    align="end"
    role="dialog"
    aria-label="This computer’s resources"
    class="w-[360px] max-w-[calc(100vw-24px)] max-h-[calc(100dvh-64px)] overflow-y-auto p-0"
  >
    <div class="border-b border-hairline px-4 py-3">
      <h2 class="text-sm font-medium">This computer</h2>
      <p class="mt-1 text-xs text-muted-foreground">
        Resource use across all apps · updates every 2 seconds
      </p>
    </div>
    <div class="space-y-4 p-4">
      <div class="space-y-1.5">
        <div class="flex items-center justify-between text-xs">
          <span class="font-medium">CPU</span>
          <span class="figure"
            >{cpu === null
              ? sample?.cpuState === CPUState.CPUMeasuring ||
                (!sample && !resources.unavailable)
                ? "Measuring…"
                : "Unavailable"
              : cpuLabel}</span
          >
        </div>
        {#if cpu !== null}
          <div
            role="meter"
            aria-label="CPU use"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-valuenow={Math.round(cpu)}
            class="h-1.5 overflow-hidden rounded-sm bg-subtle-fill"
          >
            <div
              class={highCPU ? "h-full bg-warning" : "h-full bg-primary"}
              style:width={`${cpu}%`}
            ></div>
          </div>
        {/if}
        <p class="text-xs text-muted-foreground">
          {highCPU
            ? "High CPU use · 90% or more in the latest sample."
            : "Average activity across all processor cores."}
        </p>
      </div>
      <div class="space-y-1.5">
        <div class="flex items-center justify-between text-xs">
          <span class="font-medium">RAM</span><span class="figure"
            >{memory === null ? "Unavailable" : memoryLabel}</span
          >
        </div>
        {#if memory !== null && sample}
          <div
            role="meter"
            aria-label="RAM use"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-valuenow={Math.round(memory)}
            class="h-1.5 overflow-hidden rounded-sm bg-subtle-fill"
          >
            <div
              class={highMemory ? "h-full bg-warning" : "h-full bg-primary"}
              style:width={`${memory}%`}
            ></div>
          </div>
          <p class="figure text-xs text-secondary-foreground">
            {formatMemory(
              sample.memoryTotalBytes - sample.memoryAvailableBytes,
            )} / {formatMemory(sample.memoryTotalBytes)} used
          </p>
          <p class="text-xs text-muted-foreground">
            {formatMemory(sample.memoryAvailableBytes)} available{highMemory
              ? " · 10% or less remaining."
              : "."} Availability includes reclaimable memory and is approximate.
          </p>
        {/if}
      </div>
      <div class="space-y-4 border-t border-hairline pt-4">
        {#each sample?.gpus ?? [] as adapter, index (index)}
          <ResourceGPU gpu={adapter} />
        {:else}
          <div class="flex justify-between text-xs">
            <span class="font-medium">GPU</span><span
              >{!sample && !resources.unavailable
                ? "Measuring…"
                : "Unavailable"}</span
            >
          </div>
          <p class="text-xs text-muted-foreground">
            {!sample && !resources.unavailable
              ? "Waiting for this computer’s graphics readings."
              : "GPU readings are unavailable from this computer’s graphics driver."}
          </p>
        {/each}
      </div>
    </div>
    <p
      class="border-t border-hairline px-4 py-3 text-xs leading-relaxed text-muted-foreground"
    >
      High use can help explain slowdowns. GPU activity reflects the busiest
      reported adapter; driver support varies. Remote servers aren’t measured
      here.
    </p>
  </Popover.Content>
</Popover.Root>

<style>
  @media (max-width: 639px) {
    .resource-values {
      display: none;
    }
  }
</style>
