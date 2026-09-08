<script lang="ts">
  import { Select as SelectPrimitive } from "bits-ui";
  import { cn, type WithoutChild } from "$lib/utils.js";
  import type { WithoutChildrenOrChild } from "$lib/utils.js";
  import SelectPortal from "./select-portal.svelte";
  import SelectScrollDownButton from "./select-scroll-down-button.svelte";
  import SelectScrollUpButton from "./select-scroll-up-button.svelte";
  import type { ComponentProps } from "svelte";

  let {
    ref = $bindable(null),
    class: className,
    sideOffset = 4,
    collisionPadding = 12,
    portalProps,
    children,
    preventScroll = true,
    ...restProps
  }: WithoutChild<SelectPrimitive.ContentProps> & {
    portalProps?: WithoutChildrenOrChild<ComponentProps<typeof SelectPortal>>;
  } = $props();
</script>

<SelectPortal {...portalProps}>
  <SelectPrimitive.Content
    bind:ref
    {sideOffset}
    {collisionPadding}
    {preventScroll}
    data-slot="select-content"
    class={cn(
      "min-w-36 rounded-md bg-popover text-popover-foreground shadow-md ring-1 ring-foreground/10 duration-100 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95 data-[side=inline-end]:slide-in-from-left-2 data-[side=inline-start]:slide-in-from-right-2 relative isolate z-50 flex max-h-[min(20rem,var(--bits-select-content-available-height))] flex-col overflow-hidden",
      className,
    )}
    {...restProps}
  >
    <SelectScrollUpButton />
    <SelectPrimitive.Viewport
      class={cn("min-h-0 w-full min-w-(--bits-select-anchor-width) overscroll-contain scroll-my-1")}
    >
      {@render children?.()}
    </SelectPrimitive.Viewport>
    <SelectScrollDownButton />
  </SelectPrimitive.Content>
</SelectPortal>
