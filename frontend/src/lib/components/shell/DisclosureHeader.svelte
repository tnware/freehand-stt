<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import type { Snippet } from "svelte";

  let {
    label,
    /** Quiet summary of the panel's current contents. */
    summary = "",
    open = false,
    /** id of the region this header expands. */
    controls,
    /**
     * Hide the summary while open, for panels whose body already carries the
     * same values. It keeps its width as it fades, so the chevron never moves.
     */
    fadeWhenOpen = false,
    /** Optional structured summary for status dots and other rich glance data. */
    summaryContent,
    /** Independent controls that share the header row without nesting inside its button. */
    actions,
    onToggle,
  }: {
    label: string;
    summary?: string;
    open?: boolean;
    controls: string;
    fadeWhenOpen?: boolean;
    summaryContent?: Snippet;
    actions?: Snippet;
    onToggle: () => void;
  } = $props();
</script>

<!--
  A leading chevron and label form one disclosure target. Independent actions
  stay at the trailing edge without adding a second tab stop for collapsing.
-->
<div class="workbench-header w-full gap-1 pr-1.5">
  <button
    type="button"
    class="head-trigger -ml-1 flex h-full min-w-0 flex-1 items-center gap-2 pr-2 pl-1 text-left transition-colors hover:bg-subtle-fill-hover"
    class:fade={fadeWhenOpen}
    aria-expanded={open}
    aria-controls={controls}
    onclick={onToggle}
  >
    <ChevronRightIcon
      class="size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 {open
        ? 'rotate-90'
        : ''}"
      aria-hidden="true"
    />
    <span class="caption shrink-0">{label}</span>
    {#if summaryContent || summary}
      <span
        class="summary figure min-w-0 flex-1 truncate text-xs text-ink-quiet"
      >
        {#if summaryContent}
          {@render summaryContent()}
        {:else}
          {summary}
        {/if}
      </span>
    {/if}
  </button>
  {@render actions?.()}
</div>

<style>
  .head-trigger:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: -2px;
  }

  .summary {
    transition: opacity 160ms ease;
  }
  .head-trigger.fade[aria-expanded="true"] .summary {
    opacity: 0;
  }

  @media (prefers-reduced-motion: reduce) {
    .summary {
      transition: none;
    }
  }
</style>
