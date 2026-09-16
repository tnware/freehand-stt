<script lang="ts">
  import { Combobox } from "bits-ui";
  import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";
  import { cn } from "$lib/utils";
  let { class: className, ...props }: Combobox.TriggerProps = $props();
</script>

<Combobox.Trigger
  {...props}
  class={cn(
    "absolute inset-y-0 right-0 flex w-9 items-center justify-center text-muted-foreground",
    className,
  )}
>
  <ChevronsUpDownIcon class="size-4" aria-hidden="true" />
</Combobox.Trigger>
