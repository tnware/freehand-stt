<script lang="ts">
  import { Select as SelectPrimitive } from "bits-ui";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import { fieldControl } from "$lib/utils/controlStyles";
  import { cn, type WithoutChild } from "$lib/utils.js";

  let {
    ref = $bindable(null),
    class: className,
    children,
    size = "default",
    ...restProps
  }: WithoutChild<SelectPrimitive.TriggerProps> & {
    size?: "sm" | "default";
  } = $props();
</script>

<SelectPrimitive.Trigger
  bind:ref
  data-slot="select-trigger"
  data-size={size}
  class={cn(
    fieldControl,
    "flex w-fit max-w-full items-center justify-between gap-1.5 py-1 pr-2 pl-2.5 enabled:hover:bg-control-fill-hover data-placeholder:text-muted-foreground data-[size=default]:h-8 data-[size=sm]:h-7 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4",
    className,
  )}
  {...restProps}
>
  <span data-slot="select-value" class="min-w-0 flex-1 truncate text-left"
    >{@render children?.()}</span
  >
  <ChevronDownIcon
    class="size-4 text-muted-foreground pointer-events-none"
    aria-hidden="true"
  />
</SelectPrimitive.Trigger>
