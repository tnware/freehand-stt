<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    heading = false,
    actions,
  }: {
    title: string;
    /** Give a detail region a heading without promoting every navigation label. */
    heading?: boolean;
    /** Trailing controls, kept to icon buttons so the row height holds. */
    actions?: Snippet;
  } = $props();
</script>

<!--
  Inventory and configuration columns share the editor header's height and
  inset, so changing places preserves the workbench's horizontal alignment.
-->
<div class="workbench-header justify-between pr-2">
  <svelte:element
    this={heading ? "h2" : "span"}
    class="content-kicker truncate"
  >
    {title}
  </svelte:element>
  {#if actions}
    <div class="flex shrink-0 items-center gap-1">{@render actions()}</div>
  {/if}
</div>
