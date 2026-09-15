<script lang="ts">
  import type { Component, Snippet } from "svelte";

  let {
    title,
    summary = "",
    icon: Icon,
    actions,
  }: {
    title: string;
    /** The chain in one line, so the shape is legible before you read the cards. */
    summary?: string;
    icon?: Component;
    actions?: Snippet;
  } = $props();
</script>

<!-- Names the place. The window title bar carries identity only, and the rail
     is icon-only, so this is where a pane says what it is. -->
<header
  class="flex h-11 shrink-0 items-center justify-between gap-4 border-b border-hairline px-5"
>
  <div class="flex min-w-0 flex-1 items-center gap-2.5">
    {#if Icon}<span
        class="content-section-icon flex items-center justify-center [&>svg]:size-4"
        aria-hidden="true"><Icon /></span
      >{/if}
    <h2 class="content-title min-w-0 truncate" {title}>
      {title}
    </h2>
    {#if summary}
      <p class="content-meta hidden min-w-0 flex-1 truncate min-[820px]:block">
        {summary}
      </p>
    {/if}
  </div>
  {#if actions}
    <div class="flex shrink-0 items-center gap-2">{@render actions()}</div>
  {/if}
</header>
