<script lang="ts">
  import { Combobox } from "bits-ui";
  import { cn } from "$lib/utils";
  let { class: className, ...props }: Combobox.ItemProps = $props();
</script>

<Combobox.Item
  {...props}
  data-slot="combobox-item"
  class={cn(
    "flex cursor-default items-center gap-3 rounded-md px-3 py-2.5 text-sm outline-none data-highlighted:bg-accent-wash data-highlighted:text-accent-text data-disabled:pointer-events-none data-disabled:opacity-50 forced-colors:data-highlighted:outline-2 forced-colors:data-highlighted:-outline-offset-2 forced-colors:data-highlighted:outline-[Highlight]",
    className,
  )}
/>
