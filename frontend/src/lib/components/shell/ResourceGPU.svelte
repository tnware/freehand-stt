<script lang="ts">
  import type { GPU } from "$bindings/resources";
  import { formatMemory, gpuMemoryPercent } from "$lib/stores/resources.svelte";
  let { gpu }: { gpu: GPU } = $props();
  const memory = $derived(gpuMemoryPercent(gpu));
</script>

<section class="space-y-2" aria-label={gpu.name}>
  <h3 class="text-xs font-medium break-words">{gpu.name}</h3>
  <div class="flex justify-between text-xs">
    <span class="text-muted-foreground">GPU activity</span>
    <span class="figure"
      >{gpu.utilizationAvailable
        ? `${Math.round(gpu.utilizationPercent)}%`
        : "Unavailable"}</span
    >
  </div>
  {#if gpu.utilizationAvailable}
    <div
      role="meter"
      aria-label={`${gpu.name} GPU use`}
      aria-valuemin="0"
      aria-valuemax="100"
      aria-valuenow={Math.round(gpu.utilizationPercent)}
      class="h-1.5 overflow-hidden rounded-sm bg-subtle-fill"
    >
      <div
        class={gpu.utilizationPercent >= 90
          ? "h-full bg-warning"
          : "h-full bg-primary"}
        style:width={`${gpu.utilizationPercent}%`}
      ></div>
    </div>
  {/if}
  <div class="flex flex-wrap justify-between gap-x-2 gap-y-1 text-xs">
    <span class="text-muted-foreground"
      >{gpu.unifiedMemory
        ? "GPU memory · shared RAM"
        : "Dedicated GPU memory"}</span
    >
    <span class="figure"
      >{gpu.memoryAvailable
        ? `${formatMemory(gpu.memoryUsedBytes)}${gpu.memoryTotalBytes > 0 && !gpu.unifiedMemory ? ` / ${formatMemory(gpu.memoryTotalBytes)}` : ""}`
        : "Unavailable"}</span
    >
  </div>
  {#if memory !== null}
    <div
      role="meter"
      aria-label={`${gpu.name} dedicated memory use`}
      aria-valuemin="0"
      aria-valuemax="100"
      aria-valuenow={Math.round(memory)}
      class="h-1.5 overflow-hidden rounded-sm bg-subtle-fill"
    >
      <div
        class={memory >= 90 ? "h-full bg-warning" : "h-full bg-primary"}
        style:width={`${memory}%`}
      ></div>
    </div>
  {/if}
  {#if gpu.unifiedMemory}
    <p class="text-xs text-muted-foreground">
      Included in this computer’s RAM; there is no separate VRAM pool.
    </p>
  {:else if gpu.sharedMemoryAvailable}
    <p class="text-xs text-muted-foreground">
      {formatMemory(gpu.sharedMemoryUsedBytes)} of system RAM also used by this adapter.
    </p>
  {/if}
  {#if gpu.utilizationAvailable && gpu.utilizationPercent >= 90}
    <p class="text-xs text-warning">High GPU activity · 90% or more.</p>
  {/if}
  {#if memory !== null && memory >= 90}
    <p class="text-xs text-warning">Dedicated GPU memory is nearly full.</p>
  {/if}
</section>
