<script lang="ts">
  import type { Snippet } from "svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  let {
    title,
    description,
    children,
  }: { title: string; description: string; children: Snippet } = $props();
</script>

<details
  class="group/disclosure rounded-lg border border-hairline"
>
  <summary
    class="flex cursor-pointer list-none items-center gap-3 rounded-lg px-5 py-3 transition-colors group-open/disclosure:rounded-b-none group-open/disclosure:bg-subtle-fill hover:bg-subtle-fill-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring [&::-webkit-details-marker]:hidden"
  >
    <span class="min-w-0 flex-1">
      <span class="block text-sm font-semibold">{title}</span>
      <span class="mt-1 block text-[13px] leading-5 text-muted-foreground"
        >{description}</span
      >
    </span>
    <ChevronDownIcon
      class="size-4 shrink-0 text-secondary-foreground transition-transform group-open/disclosure:rotate-180 motion-reduce:transition-none"
    />
  </summary>
  <div
    class="settings-group border-t border-hairline [&>*+*]:border-t [&>*+*]:border-hairline"
  >
    {@render children()}
  </div>
</details>
