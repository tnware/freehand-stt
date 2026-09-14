<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    actions,
  }: {
    title: string;
    /** Trailing controls, kept to icon buttons so the row height holds. */
    actions?: Snippet;
  } = $props();
</script>

<!--
  One header for every left column in the app - the workflow settings, the
  runtime inventory, the settings sections. They sit at the same height under
  the title bar and share one type size, so moving between panes does not
  shift the horizon.
-->
<div
  class="flex h-[33px] shrink-0 items-center justify-between gap-2 border-b border-hairline pr-2 pl-3.5"
>
  <span
    class="truncate text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
    >{title}</span
  >
  {#if actions}
    <div class="flex shrink-0 items-center gap-1">{@render actions()}</div>
  {/if}
</div>
