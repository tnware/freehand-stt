<script lang="ts">
  import { PaneResizer, type PaneResizerProps } from "paneforge";
  import GripVerticalIcon from "@lucide/svelte/icons/grip-vertical";
  import { cn } from "$lib/utils";
  let {
    class: className,
    withHandle = false,
    ...props
  }: PaneResizerProps & { withHandle?: boolean } = $props();
</script>

<PaneResizer
  class={cn(
    "relative flex w-4 shrink-0 items-center justify-center rounded-lg bg-transparent text-ink-quiet outline-none transition-colors hover:bg-accent-wash hover:text-accent-text focus-visible:bg-accent-wash focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring data-[active]:bg-accent-wash data-[active]:text-accent-text data-[direction=vertical]:h-4 data-[direction=vertical]:w-full",
    className,
  )}
  {...props}
>
  {#if withHandle}<GripVerticalIcon class="relative size-3" aria-hidden="true" />{/if}
</PaneResizer>
