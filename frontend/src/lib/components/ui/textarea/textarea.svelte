<script lang="ts">
  import { fieldControl } from "$lib/utils/controlStyles";
  import { cn, type WithElementRef, type WithoutChildren } from "$lib/utils.js";
  import type { HTMLTextareaAttributes } from "svelte/elements";

  let {
    ref = $bindable(null),
    value = $bindable(),
    class: className,
    "data-slot": dataSlot = "textarea",
    ...restProps
  }: WithoutChildren<WithElementRef<HTMLTextareaAttributes>> = $props();
</script>

<textarea
  bind:this={ref}
  data-slot={dataSlot}
  class={cn(
    fieldControl,
    "flex field-sizing-content min-h-16 w-full px-2.5 py-2 leading-relaxed",
    className,
  )}
  bind:value
  {...restProps}
></textarea>
