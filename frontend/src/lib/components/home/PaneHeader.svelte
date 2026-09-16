<script lang="ts">
  import type { Component, Snippet } from "svelte";
  import { cn } from "$lib/utils";

  let {
    title,
    summary = "",
    icon: Icon,
    actions,
    class: className = "",
    headingID,
    focusable = false,
    height = $bindable(0),
  }: {
    title: string;
    /** Short context that fits alongside the current workspace title. */
    summary?: string;
    icon?: Component;
    actions?: Snippet;
    class?: string;
    headingID?: string;
    focusable?: boolean;
    height?: number;
  } = $props();
</script>

<!-- Names the place. The window title bar carries identity only, and the rail
     is icon-only, so this is where a pane says what it is. -->
<header
  class={cn("workbench-header @container justify-between", className)}
  bind:clientHeight={height}
  data-page-header
>
  <div class="flex min-w-0 flex-1 items-center gap-2.5">
    {#if Icon}<span
        class="content-section-icon flex items-center justify-center [&>svg]:size-4"
        aria-hidden="true"><Icon /></span
      >{/if}
    <!-- Settings navigation can focus this heading without adding a tab stop. -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <h2
      class="content-title min-w-0 truncate"
      id={headingID}
      tabindex={focusable ? -1 : undefined}
      {title}
    >
      {title}
    </h2>
    {#if summary}
      <p
        class="content-meta hidden min-w-0 flex-1 truncate @min-[560px]:block"
        title={summary}
      >
        {summary}
      </p>
    {/if}
  </div>
  {#if actions}
    <div class="flex shrink-0 items-center gap-1">{@render actions()}</div>
  {/if}
</header>
