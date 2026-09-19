<script lang="ts">
  import { Combobox } from "bits-ui";
  import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";
  import { cn } from "$lib/utils";
  let { class: className, ...props }: Combobox.TriggerProps = $props();
</script>

<Combobox.Trigger
  {...props}
  class={cn(
    "absolute inset-y-0 right-0 flex w-8 items-center justify-center rounded-r-md text-muted-foreground outline-none transition-colors hover:bg-subtle-fill-hover hover:text-foreground aria-expanded:text-accent-text focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 motion-reduce:transition-none",
    className,
  )}
>
  <ChevronsUpDownIcon class="size-4" aria-hidden="true" />
</Combobox.Trigger>
