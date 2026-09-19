<script lang="ts">
  import { Combobox } from "bits-ui";
  import type { Snippet } from "svelte";
  import { cn } from "$lib/utils";

  let {
    class: className,
    sideOffset = 4,
    collisionPadding = 12,
    children,
    footer,
    ...props
  }: Combobox.ContentProps & { footer?: Snippet } = $props();
</script>

<Combobox.Portal>
  <Combobox.Content
    {...props}
    data-slot="combobox-content"
    {sideOffset}
    {collisionPadding}
    class={cn(
      "z-50 flex max-h-[min(20rem,var(--bits-combobox-content-available-height))] w-(--bits-combobox-anchor-width) min-w-[min(16rem,calc(100vw-24px))] max-w-[calc(100vw-24px)] flex-col overflow-hidden rounded-lg border border-border bg-popover text-popover-foreground shadow-float",
      className,
    )}
  >
    <div class="min-h-0 overflow-y-auto overscroll-contain p-1">
      {@render children?.()}
    </div>
    {#if footer}
      <div class="shrink-0 border-t border-hairline p-1">
        {@render footer()}
      </div>
    {/if}
  </Combobox.Content>
</Combobox.Portal>
