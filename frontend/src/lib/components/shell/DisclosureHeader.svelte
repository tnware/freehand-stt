<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import type { Snippet } from "svelte";

  let {
    label,
    /** Right-aligned line of facts, in mono. */
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
  The one panel header on the main screen. The label is the same mono caption
  the rack modules use, so a panel that collapses and a module that does not
  still read as the same kind of thing.
-->
<div class="flex h-[34px] w-full shrink-0 items-center border-b border-hairline pr-1.5 pl-3">
  <button
    type="button"
    class="head-trigger flex h-full min-w-0 flex-1 items-center gap-3 pr-2 text-left"
    class:fade={fadeWhenOpen}
    aria-expanded={open}
    aria-controls={controls}
    onclick={onToggle}
  >
    <span class="caption shrink-0">{label}</span>
    {#if summaryContent || summary}
      <span class="summary figure min-w-0 flex-1 truncate text-[10px] text-ink-quiet">
        {#if summaryContent}
          {@render summaryContent()}
        {:else}
          {summary}
        {/if}
      </span>
    {/if}
  </button>
  {@render actions?.()}
  <button
    type="button"
    class="chevron-trigger flex size-7 shrink-0 items-center justify-center rounded-sm text-muted-foreground transition-colors hover:bg-subtle-fill-hover hover:text-foreground"
    aria-label={open ? `Collapse ${label}` : `Expand ${label}`}
    aria-expanded={open}
    aria-controls={controls}
    onclick={onToggle}
  >
    <ChevronRightIcon
      class="size-[14px] transition-transform duration-200 {open ? 'rotate-90' : ''}"
    />
  </button>
</div>

<style>
  .head-trigger:focus-visible,
  .chevron-trigger:focus-visible {
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
