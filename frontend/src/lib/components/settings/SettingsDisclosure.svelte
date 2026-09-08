<script lang="ts">
  import type { Snippet } from "svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  let { title, description, children }: { title: string; description: string; children: Snippet } =
    $props();
</script>

<details class="group/disclosure overflow-hidden rounded-xl border border-hairline bg-layer-fill">
  <summary
    class="flex cursor-pointer list-none items-center gap-3 px-5 py-3.5 hover:bg-subtle-fill-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring [&::-webkit-details-marker]:hidden"
  >
    <span class="min-w-0 flex-1">
      <span class="block text-sm font-medium">{title}</span>
      <span class="mt-0.5 block text-xs text-muted-foreground">{description}</span>
    </span>
    <ChevronDownIcon
      class="size-4 shrink-0 text-muted-foreground transition-transform group-open/disclosure:rotate-180 motion-reduce:transition-none"
    />
  </summary>
  <div class="border-t border-hairline [&>*+*]:border-t [&>*+*]:border-hairline">
    {@render children()}
  </div>
</details>
