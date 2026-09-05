<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import SlidersIcon from "@lucide/svelte/icons/sliders-horizontal";
  import type { Snippet } from "svelte";

  let {
    label,
    /** Tailwind background class for the state LED, or "" for no lamp. */
    dot = "",
    /** One right-aligned fact: latency, profile name, count. */
    meta = "",
    metaTone = "quiet",
    settingsLabel = "",
    onSettings,
    open,
    controls,
    onToggle,
    actions,
    icon,
    children,
  }: {
    label: string;
    dot?: string;
    meta?: string;
    metaTone?: "quiet" | "ok" | "warn" | "bad";
    settingsLabel?: string;
    onSettings?: () => void;
    open?: boolean;
    controls?: string;
    onToggle?: () => void;
    actions?: Snippet;
    icon?: Snippet;
    children: Snippet;
  } = $props();

  const collapsible = $derived(open !== undefined && Boolean(controls) && Boolean(onToggle));

  const metaClass = $derived(
    metaTone === "ok"
      ? "text-success"
      : metaTone === "warn"
        ? "text-warning"
        : metaTone === "bad"
          ? "text-destructive"
          : "text-muted-foreground",
  );
</script>

<!--
  One rack module: a lamp, a name, one fact, and a door to the full settings.
  The rack replaced a single collapsing panel of seven unlabelled icon toggles,
  so every module here says what it is in words.
-->
<section class="module-card shrink-0 rounded-lg border border-card-stroke bg-card p-3 shadow-lift">
  <div class="flex items-center gap-2">
    {#if collapsible}
      <button
        type="button"
        class="module-trigger flex h-5 min-w-0 flex-1 items-center gap-2 text-left"
        aria-expanded={open}
        aria-controls={controls}
        onclick={onToggle}
      >
        {#if dot}
          <span class="size-1.5 shrink-0 rounded-full {dot}" aria-hidden="true"></span>
        {/if}
        {@render icon?.()}
        <h2 class="caption shrink-0">{label}</h2>
        <span class="flex-1"></span>
        {#if meta}
          <span class="figure min-w-0 max-w-[58%] truncate text-[10px] {metaClass}">{meta}</span>
        {/if}
      </button>
    {:else}
      {#if dot}
        <span class="size-1.5 shrink-0 rounded-full {dot}" aria-hidden="true"></span>
      {/if}
      {@render icon?.()}
      <h2 class="caption">{label}</h2>
      <span class="flex-1"></span>
      {#if meta}
        <span class="figure min-w-0 max-w-[58%] truncate text-[10px] {metaClass}">{meta}</span>
      {/if}
    {/if}
    {@render actions?.()}
    {#if onSettings}
      <button
        type="button"
        class="door"
        aria-label={settingsLabel || `Open ${label} settings`}
        title={settingsLabel || `Open ${label} settings`}
        onclick={onSettings}
      >
        <SlidersIcon class="size-[13px]" />
      </button>
    {/if}
    {#if collapsible}
      <button
        type="button"
        class="chevron"
        aria-label={open ? `Collapse ${label}` : `Expand ${label}`}
        aria-expanded={open}
        aria-controls={controls}
        onclick={onToggle}
      >
        <ChevronRightIcon
          class="size-[14px] transition-transform duration-200 {open ? 'rotate-90' : ''}"
        />
      </button>
    {/if}
  </div>
  <div
    class:closed={collapsible && !open}
    class="drawer"
    id={controls}
    inert={collapsible && !open}
  >
    <div class="drawer-inner">
      <div class="module-body">{@render children()}</div>
    </div>
  </div>
</section>

<style>
  .module-card {
    display: flex;
    flex-direction: column;
  }
  .module-trigger:focus-visible,
  .chevron:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 1px;
  }
  .drawer {
    display: grid;
    grid-template-rows: 1fr;
    transition: grid-template-rows 220ms cubic-bezier(0.4, 0, 0.2, 1);
  }
  .drawer.closed {
    grid-template-rows: 0fr;
  }
  .drawer-inner {
    min-height: 0;
    overflow: hidden;
  }
  .module-body {
    padding-top: 0.625rem;
  }
  .door {
    display: grid;
    place-items: center;
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
    border-radius: var(--radius-sm);
    color: var(--ink-quiet);
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .door:hover {
    background-color: var(--subtle-fill-hover);
    color: var(--foreground);
  }
  .door:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 1px;
  }
  .chevron {
    display: grid;
    place-items: center;
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
    border-radius: var(--radius-sm);
    color: var(--muted-foreground);
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }
  .chevron:hover {
    background-color: var(--subtle-fill-hover);
    color: var(--foreground);
  }
  @media (prefers-reduced-motion: reduce) {
    .drawer {
      transition: none;
    }
  }
</style>
