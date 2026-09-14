<script lang="ts">
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import ChevronUpIcon from "@lucide/svelte/icons/chevron-up";
  import { Button } from "$lib/components/ui/button";

  export type PanelTab = { id: string; label: string };

  let {
    tabs,
    active = $bindable(),
    collapsed = $bindable(false),
    note = "",
    collapsible = true,
  }: {
    tabs: PanelTab[];
    active: string;
    collapsed?: boolean;
    /** A quiet right-aligned fact about the panel, such as where output lives. */
    note?: string;
    collapsible?: boolean;
  } = $props();
</script>

<!--
  The bottom panel's strip, shared by every pane that has one. The selected tab
  is underlined rather than filled so the row reads as panel chrome, not a row
  of buttons.
-->
<div
  class="flex h-7 shrink-0 items-center justify-between border-b border-hairline"
>
  <div class="flex min-w-0" role="tablist" aria-label="Panel">
    {#each tabs as tab (tab.id)}
      <button
        type="button"
        role="tab"
        aria-selected={active === tab.id}
        class="panel-tab relative h-7 px-3 text-[10px] font-semibold tracking-[0.08em] whitespace-nowrap uppercase transition-colors {active ===
        tab.id
          ? 'text-secondary-foreground'
          : 'text-ink-quiet hover:text-secondary-foreground'}"
        onclick={() => {
          active = tab.id;
          collapsed = false;
        }}>{tab.label}</button
      >
    {/each}
  </div>
  <div class="flex shrink-0 items-center gap-2 pr-1.5">
    {#if note}
      <span class="hidden font-mono text-[10px] text-ink-quiet min-[900px]:inline"
        >{note}</span
      >
    {/if}
    {#if collapsible}
      <Button
        variant="ghost"
        size="xs"
        class="size-6 rounded-sm p-0"
        aria-expanded={!collapsed}
        aria-label={collapsed ? "Show panel" : "Hide panel"}
        title={collapsed ? "Show panel" : "Hide panel"}
        onclick={() => (collapsed = !collapsed)}
      >
        {#if collapsed}
          <ChevronUpIcon class="size-3.5" />
        {:else}
          <ChevronDownIcon class="size-3.5" />
        {/if}
      </Button>
    {/if}
  </div>
</div>

<style>
  .panel-tab[aria-selected="true"]::after {
    content: "";
    position: absolute;
    inset-inline: 0.75rem;
    bottom: 0;
    height: 1px;
    background: var(--muted-foreground);
  }
</style>
