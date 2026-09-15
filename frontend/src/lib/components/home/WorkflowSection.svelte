<script lang="ts">
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import SlidersIcon from "@lucide/svelte/icons/sliders-horizontal";
  import type { Snippet } from "svelte";

  let {
    label,
    embedded = false,
    sidebar = false,
    /** Tailwind background class for the status indicator, or an empty string. */
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
    embedded?: boolean;
    sidebar?: boolean;
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

  const collapsible = $derived(
    open !== undefined && Boolean(controls) && Boolean(onToggle),
  );

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
  Workflow controls share a flat section header and one disclosure target.
-->
<section
  class="workflow-section shrink-0"
  class:divided={!embedded}
  class:sidebar
>
  <div class="flex min-h-7 min-w-0 items-center gap-1">
    {#if collapsible}
      <button
        type="button"
        class="module-trigger flex min-h-7 min-w-0 flex-1 items-center gap-2 text-left"
        aria-expanded={open}
        aria-controls={controls}
        onclick={onToggle}
      >
        <ChevronRightIcon
          class="size-3.5 shrink-0 text-muted-foreground transition-transform duration-150 motion-reduce:transition-none {open
            ? 'rotate-90'
            : ''}"
          aria-hidden="true"
        />
        {#if dot}
          <span class="size-1.5 shrink-0 rounded-full {dot}" aria-hidden="true"
          ></span>
        {/if}
        {@render icon?.()}
        <h2 class="content-section-title min-w-0 truncate" title={label}>
          {label}
        </h2>
        <span class="flex-1"></span>
        {#if meta}
          <span
            class="min-w-0 max-w-[48%] truncate text-xs {metaClass}"
            title={meta}>{meta}</span
          >
        {/if}
      </button>
    {:else}
      {#if dot}
        <span class="size-1.5 shrink-0 rounded-full {dot}" aria-hidden="true"
        ></span>
      {/if}
      {@render icon?.()}
      <h2 class="content-section-title min-w-0 truncate" title={label}>
        {label}
      </h2>
      <span class="flex-1"></span>
      {#if meta}
        <span
          class="min-w-0 max-w-[48%] truncate text-xs {metaClass}"
          title={meta}>{meta}</span
        >
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
  .sidebar h2 {
    font-size: 12px;
  }
  .sidebar .module-body {
    padding-top: 0.375rem;
  }
  .divided {
    border-top: 1px solid var(--hairline);
    padding-block: 0.5rem;
  }
  .workflow-section {
    display: flex;
    min-width: 0;
    flex-direction: column;
  }
  .module-trigger:focus-visible {
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
    width: 1.5rem;
    height: 1.5rem;
    flex-shrink: 0;
    border-radius: var(--radius-sm);
    color: var(--muted-foreground);
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
  @media (prefers-reduced-motion: reduce) {
    .drawer {
      transition: none;
    }
  }
</style>
