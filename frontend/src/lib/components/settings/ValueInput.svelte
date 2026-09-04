<script lang="ts">
  import type { HTMLInputAttributes, HTMLInputTypeAttribute } from "svelte/elements";
  import { Input } from "$lib/components/ui/input";
  import { cn } from "$lib/utils";

  type ValueInputProps = Omit<HTMLInputAttributes, "type" | "value" | "files"> & {
    value?: string | number;
    type?: Exclude<HTMLInputTypeAttribute, "file">;
    mono?: boolean;
  };

  let {
    value = $bindable(),
    mono = true,
    type,
    class: className,
    ...rest
  }: ValueInputProps = $props();
</script>

<Input
  bind:value
  {type}
  class={cn(
    mono && "font-mono text-sm tracking-[-0.01em]",
    className,
  )}
  {...rest}
/>
