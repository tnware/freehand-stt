<script lang="ts">
  import { textControl } from "$lib/utils/controlStyles";
  import { cn, type WithElementRef } from "$lib/utils.js";
  import type {
    HTMLInputAttributes,
    HTMLInputTypeAttribute,
  } from "svelte/elements";

  type InputType = Exclude<HTMLInputTypeAttribute, "file">;

  type Props = WithElementRef<
    Omit<HTMLInputAttributes, "type"> &
      (
        | { type: "file"; files?: FileList }
        | { type?: InputType; files?: undefined }
      )
  >;

  let {
    ref = $bindable(null),
    value = $bindable(),
    type,
    files = $bindable(),
    class: className,
    "data-slot": dataSlot = "input",
    ...restProps
  }: Props = $props();
</script>

{#if type === "file"}
  <input
    bind:this={ref}
    data-slot={dataSlot}
    class={cn(
      textControl,
      "file:h-7 file:inline-flex file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground",
      className,
    )}
    type="file"
    bind:files
    bind:value
    {...restProps}
  />
{:else}
  <input
    bind:this={ref}
    data-slot={dataSlot}
    class={cn(
      textControl,
      "file:h-7 file:inline-flex file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground",
      className,
    )}
    {type}
    bind:value
    {...restProps}
  />
{/if}
