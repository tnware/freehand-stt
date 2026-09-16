<script lang="ts">
  import type { Snippet } from "svelte";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  let { title, children }: { title: string; children: Snippet } = $props();
</script>

<details class="group/download min-w-0">
  <summary
    class="flex w-fit cursor-pointer list-none items-center gap-1 rounded-sm py-1 font-medium text-accent-text hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring [&::-webkit-details-marker]:hidden"
  >
    <ChevronRightIcon
      class="size-3.5 shrink-0 transition-transform group-open/download:rotate-90 motion-reduce:transition-none"
      aria-hidden="true"
    />
    {title}
  </summary>
  {@render children()}
</details>
