<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    title,
    summary = "",
    actions,
  }: {
    title: string;
    /** The chain in one line, so the shape is legible before you read the cards. */
    summary?: string;
    actions?: Snippet;
  } = $props();
</script>

<!-- Names the place. The window title bar carries identity only, and the rail
     is icon-only, so this is where a pane says what it is. -->
<header
  class="flex h-11 shrink-0 items-center justify-between gap-4 border-b border-hairline px-5"
>
  <div class="flex min-w-0 items-baseline gap-2.5">
    <h2 class="font-display text-[15px] font-semibold tracking-tight">
      {title}
    </h2>
    {#if summary}
      <p class="hidden truncate font-mono text-[10px] text-ink-quiet min-[820px]:block">
        {summary}
      </p>
    {/if}
  </div>
  {#if actions}
    <div class="flex shrink-0 items-center gap-2">{@render actions()}</div>
  {/if}
</header>
