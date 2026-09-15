<script lang="ts">
  import type { Snippet } from "svelte";
  import {
    getWorkbenchLayout,
    type SidebarID,
  } from "$lib/workbench-layout.svelte";
  let { id, children }: { id: SidebarID; children: Snippet } = $props();
  const layout = getWorkbenchLayout();
  $effect(() => {
    if (!layout) return;
    const content = children;
    const area = id;
    layout.sidebars[area] = content;
    return () => {
      if (layout.sidebars[area] === content) delete layout.sidebars[area];
    };
  });
</script>

{#if !layout}{@render children()}{/if}
