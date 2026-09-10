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
    "relative flex w-3 shrink-0 items-center justify-center bg-layer-fill text-muted-foreground outline-none before:absolute before:inset-y-0 before:left-1/2 before:w-px before:bg-hairline transition-colors hover:bg-accent-wash hover:text-accent-text focus-visible:bg-accent-wash focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring data-[active]:bg-accent-wash data-[active]:text-accent-text data-[direction=vertical]:h-3 data-[direction=vertical]:w-full data-[direction=vertical]:before:inset-x-0 data-[direction=vertical]:before:top-1/2 data-[direction=vertical]:before:h-px data-[direction=vertical]:before:w-full",
    className,
  )}
  {...props}
>
  {#if withHandle}<GripVerticalIcon class="relative size-3 bg-layer-fill" aria-hidden="true" />{/if}
</PaneResizer>
