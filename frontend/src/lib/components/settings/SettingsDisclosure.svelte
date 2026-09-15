<script lang="ts">
  import type { Snippet } from "svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  let {
    title,
    description,
    children,
  }: { title: string; description: string; children: Snippet } = $props();
</script>

<details class="group/disclosure min-w-0 rounded-sm border border-hairline">
  <summary
    class="content-disclosure flex cursor-pointer list-none items-center gap-3 px-3 py-3 [&::-webkit-details-marker]:hidden"
  >
    <span class="min-w-0 flex-1">
      <span class="content-section-title block">{title}</span>
      <span class="content-meta mt-1 block">{description}</span>
    </span>
    <ChevronDownIcon
      class="size-4 shrink-0 text-secondary-foreground transition-transform group-open/disclosure:rotate-180 motion-reduce:transition-none"
    />
  </summary>
  <div
    class="settings-group border-t border-hairline bg-well px-3 [&>*+*]:border-t [&>*+*]:border-hairline"
  >
    {@render children()}
  </div>
</details>
