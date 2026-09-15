<script lang="ts">
  import { tick, untrack, type Snippet } from "svelte";
  import type {
    SidebarID,
    WorkbenchLayout,
  } from "$lib/workbench-layout.svelte";
  import LayoutDivider from "./LayoutDivider.svelte";
  let {
    layout,
    area,
    children,
    panel,
  }: {
    layout: WorkbenchLayout;
    area: SidebarID;
    children: Snippet;
    panel: Snippet;
  } = $props();
  const sidebar = $derived(layout.sidebars[area]);
  const primaryVisible = $derived(layout.primaryVisible && Boolean(sidebar));
  const rightVisible = $derived(area === "history" && layout.secondaryVisible);
  const rightOverlay = $derived(
    rightVisible && !layout.secondaryAvailable.current,
  );
  let primary = $state<HTMLElement>();
  let secondary = $state<HTMLElement>();
  let bottom = $state<HTMLElement>();
  let center = $state<HTMLDivElement>();
  let columns = $state<HTMLDivElement>();

  function restoreFocus(label: string) {
    void tick().then(() => {
      const button = document.querySelector<HTMLButtonElement>(
        `button[aria-label="${label}"]`,
      );
      const fallback =
        document.querySelector<HTMLButtonElement>(
          'button[aria-label="Toggle primary sidebar"]:enabled',
        ) ??
        document.querySelector<HTMLButtonElement>(
          'nav[aria-label="Workspace"] button[aria-current="page"]:enabled',
        );
      (button && !button.disabled ? button : fallback)?.focus();
    });
  }
  // Navigation closes a compact drawer; wide-screen visibility is a workspace preference.
  $effect(() => {
    area;
    untrack(() => {
      layout.compactPrimaryOpen = false;
      layout.compactSecondaryOpen = false;
    });
  });
  $effect(() => {
    if (!layout.compact.current) layout.compactPrimaryOpen = false;
  });
  $effect(() => {
    if (layout.secondaryAvailable.current) layout.compactSecondaryOpen = false;
  });
  $effect.pre(() => {
    const active = document.activeElement;
    const separator =
      active?.getAttribute("role") === "separator"
        ? active.getAttribute("aria-label")
        : "";
    if (!primaryVisible && primary?.contains(document.activeElement))
      restoreFocus("Toggle primary sidebar");
    if (
      !layout.bottomVisible &&
      (bottom?.contains(active) ||
        separator === "Resize editor and bottom panel")
    )
      restoreFocus("Toggle bottom panel");
    if (
      (!rightVisible && secondary?.contains(active)) ||
      (!layout.secondaryAvailable.current &&
        separator === "Resize editor and secondary sidebar")
    )
      restoreFocus("Toggle secondary sidebar");
  });
  $effect(() => {
    if (!layout.compact.current || !layout.compactPrimaryOpen) return;
    void tick().then(() =>
      primary
        ?.querySelector<HTMLElement>(
          'input, [aria-current="true"], [aria-pressed="true"], button',
        )
        ?.focus(),
    );
  });
  function closePrimary() {
    layout.compactPrimaryOpen = false;
    restoreFocus("Toggle primary sidebar");
  }
  function closeSecondary() {
    layout.compactSecondaryOpen = false;
    restoreFocus("Toggle secondary sidebar");
  }
  $effect(() => {
    if (rightOverlay) void tick().then(() => secondary?.focus());
  });
</script>

<svelte:window
  onkeydown={(event) => {
    if (event.key === "Escape" && !event.defaultPrevented && rightOverlay) {
      event.preventDefault();
      closeSecondary();
      return;
    }
    if (
      event.key === "Escape" &&
      !event.defaultPrevented &&
      layout.compact.current &&
      layout.compactPrimaryOpen
    ) {
      event.preventDefault();
      closePrimary();
    }
  }}
/>

<div
  class="workbench-frame relative flex min-h-0 min-w-0 flex-1 overflow-hidden"
>
  {#if layout.compact.current && primaryVisible}
    <button
      type="button"
      class="absolute inset-0 z-40 bg-black/30"
      style:inset-inline-start="min(280px, 85%)"
      aria-label="Dismiss primary sidebar"
      onclick={closePrimary}
    ></button>
  {/if}
  {#if rightOverlay}
    <button
      type="button"
      class="absolute inset-0 z-40 bg-black/30"
      style:inset-inline-end="min(420px, 85%)"
      aria-label="Dismiss secondary sidebar"
      onclick={closeSecondary}
    ></button>
  {/if}
  <aside
    bind:this={primary}
    id="workbench-primary-sidebar"
    aria-label="Primary sidebar"
    inert={rightOverlay}
    class="primary-sidebar"
    class:compact={layout.compact.current}
    style:display={primaryVisible ? "flex" : "none"}
  >
    {#if sidebar}{@render sidebar()}{/if}
  </aside>

  <div
    bind:this={columns}
    class="workbench-columns grid min-h-0 min-w-0 flex-1"
    inert={layout.compact.current && primaryVisible}
    style:grid-template-columns={rightVisible && !rightOverlay
      ? `minmax(0, ${100 - layout.secondarySize}fr) 1px minmax(320px, ${layout.secondarySize}fr)`
      : "minmax(0, 1fr)"}
  >
    <div
      bind:this={center}
      class="grid min-h-0 min-w-0 overflow-hidden"
      inert={rightOverlay}
      style:grid-template-rows={layout.bottomVisible
        ? `minmax(0, ${100 - layout.bottomSize}fr) 1px minmax(120px, ${layout.bottomSize}fr)`
        : "minmax(0, 1fr)"}
    >
      <div class="flex min-h-0 min-w-0 flex-col overflow-hidden">
        {@render children()}
      </div>
      {#if layout.bottomVisible}
        <LayoutDivider
          orientation="horizontal"
          label="Resize editor and bottom panel"
          value={layout.bottomSize}
          min={18}
          max={55}
          container={center}
          onResize={(value) => (layout.bottomSize = value)}
        />
        <section
          bind:this={bottom}
          id="workbench-bottom-panel"
          aria-label="Bottom panel"
          class="flex min-h-0 min-w-0 flex-col overflow-hidden"
        >
          {@render panel()}
        </section>
      {/if}
    </div>
    {#if rightVisible}
      {#if !rightOverlay}
        <LayoutDivider
          orientation="vertical"
          label="Resize editor and secondary sidebar"
          value={layout.secondarySize}
          min={30}
          max={55}
          container={columns}
          onResize={(value) => (layout.secondarySize = value)}
        />
      {/if}
      <aside
        bind:this={secondary}
        id="workbench-secondary-sidebar"
        aria-label="Secondary sidebar"
        tabindex="-1"
        class="secondary-sidebar flex min-h-0 min-w-0 flex-col overflow-hidden bg-background outline-none"
        class:overlay={rightOverlay}
      >
        {#if layout.details}{@render layout.details()}{/if}
      </aside>
    {/if}
  </div>
</div>

<style>
  .primary-sidebar {
    width: 252px;
    min-width: 0;
    min-height: 0;
    flex-shrink: 0;
    flex-direction: column;
    overflow: hidden;
    background: var(--layer-fill);
  }
  .primary-sidebar.compact {
    position: absolute;
    inset: 0 auto 0 0;
    z-index: 41;
    width: min(280px, 85%);
    box-shadow: 8px 0 24px #0003;
  }
  .primary-sidebar :global(> *) {
    width: 100%;
    height: 100%;
  }
  .secondary-sidebar.overlay {
    position: absolute;
    inset: 0 0 0 auto;
    z-index: 41;
    width: min(420px, 85%);
    border-left: 1px solid var(--hairline);
    box-shadow: -8px 0 24px #0003;
  }
</style>
