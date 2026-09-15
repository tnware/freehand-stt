<script lang="ts">
  import type { Snippet } from "svelte";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  let {
    title,
    description,
    children,
  }: { title: string; description: string; children: Snippet } = $props();
</script>

<details class="group/disclosure min-w-0 border-b border-hairline">
  <summary
    class="content-disclosure flex min-h-[38px] cursor-pointer list-none items-start gap-2 py-2 [&::-webkit-details-marker]:hidden"
  >
    <ChevronRightIcon
      class="mt-0.5 size-3.5 shrink-0 text-muted-foreground transition-transform group-open/disclosure:rotate-90 motion-reduce:transition-none"
      aria-hidden="true"
    />
    <span class="min-w-0 flex-1">
      <span class="content-section-title block">{title}</span>
      <span class="content-meta mt-1 block">{description}</span>
    </span>
  </summary>
  <div
    class="settings-group border-t border-hairline [&>*+*]:border-t [&>*+*]:border-hairline"
  >
    {@render children()}
  </div>
</details>
