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
    "relative flex w-4 shrink-0 items-center justify-center rounded-sm text-muted-foreground outline-none transition-colors hover:bg-accent/10 hover:text-accent focus-visible:bg-accent/10 focus-visible:ring-2 focus-visible:ring-ring data-[active]:bg-accent/10 data-[active]:text-accent data-[direction=vertical]:h-4 data-[direction=vertical]:w-full",
    className,
  )}
  {...props}
>
  {#if withHandle}<GripVerticalIcon class="size-3.5" aria-hidden="true" />{/if}
</PaneResizer>
