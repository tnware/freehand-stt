<script lang="ts">
  import type { Snippet } from "svelte";
  import { MediaQuery } from "svelte/reactivity";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import ChevronUpIcon from "@lucide/svelte/icons/chevron-up";
  import * as Resizable from "$lib/components/ui/resizable";
  import { Button } from "$lib/components/ui/button";

  let {
    result,
    history,
    output,
    diagnostics,
    hasHistory,
    historyCount,
    working,
    outputAvailable = false,
  }: {
    result: Snippet;
    history: Snippet;
    /** Runtime output. */
    output?: Snippet;
    /** Endpoint diagnostics for the active workflow. */
    diagnostics?: Snippet;
    hasHistory: boolean;
    historyCount: number;
    working: boolean;
    outputAvailable?: boolean;
  } = $props();

  type Tab = "recent" | "output" | "diagnostics";
  let tab = $state<Tab>("recent");
  const active = $derived(tab);

  // The chain already owns the full width above this, so history sits under
  // the transcript rather than beside it: a side pane would squeeze the one
  // column that actually holds prose.
  const roomy = new MediaQuery("(min-height: 560px)");
  let collapsed = $state(false);
  let selectedView = $state<"result" | "history">("result");
  const visibleView = $derived(working ? "result" : selectedView);
  const stacked = $derived(roomy.current);
</script>

{#if stacked}
  <Resizable.PaneGroup
    direction="vertical"
    autoSaveId="freehand-workspace-v2"
    keyboardResizeBy={2}
  >
    <Resizable.Pane id="current-result-pane" defaultSize={70} minSize={35}>
      {@render result()}
    </Resizable.Pane>
    {#if !collapsed}
      <Resizable.Handle aria-label="Resize transcript and history" />
    {/if}
    <Resizable.Pane
      id="recent-history-pane"
      defaultSize={collapsed ? 0 : 30}
      minSize={collapsed ? 0 : 18}
    >
      <div class="flex h-full min-h-0 flex-col">
        <div
          class="flex h-7 shrink-0 items-center justify-between border-b border-hairline"
        >
          <div class="flex min-w-0" role="tablist" aria-label="Workspace panel">
            {#each [{ id: "recent", label: hasHistory ? `Recent · ${historyCount}` : "Recent" }, { id: "output", label: "Runtime output" }, { id: "diagnostics", label: "Diagnostics" }] as entry (entry.id)}
                <button
                  type="button"
                  role="tab"
                  aria-selected={active === entry.id}
                  class="panel-tab relative h-7 px-3 text-[10px] font-semibold tracking-[0.08em] uppercase transition-colors {active ===
                  entry.id
                    ? 'text-secondary-foreground'
                    : 'text-ink-quiet hover:text-secondary-foreground'}"
                  onclick={() => {
                    tab = entry.id as Tab;
                    collapsed = false;
                  }}>{entry.label}</button
                >
            {/each}
          </div>
          <Button
            variant="ghost"
            size="xs"
            class="mr-1 size-6 rounded-sm p-0"
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
        </div>
        {#if !collapsed}
          <div class="flex min-h-0 flex-1 flex-col">
            {#if active === "output" && output}
              {@render output()}
            {:else if active === "diagnostics" && diagnostics}
              {@render diagnostics()}
            {:else}
              {@render history()}
            {/if}
          </div>
        {/if}
      </div>
    </Resizable.Pane>
  </Resizable.PaneGroup>
{:else}
  {#if hasHistory}
    <div
      class="flex shrink-0 gap-1 px-3 py-2"
      role="group"
      aria-label="Workspace view"
    >
      <Button
        variant={visibleView === "result" ? "secondary" : "ghost"}
        size="sm"
        aria-pressed={visibleView === "result"}
        onclick={() => (selectedView = "result")}>Transcript</Button
      >
      <Button
        variant={visibleView === "history" ? "secondary" : "ghost"}
        size="sm"
        aria-pressed={visibleView === "history"}
        disabled={working}
        onclick={() => (selectedView = "history")}
        >Recent · {historyCount}</Button
      >
    </div>
  {/if}
  <div class="flex min-h-0 flex-1 flex-col">
    {#if hasHistory && visibleView === "history"}{@render history()}{:else}{@render result()}{/if}
  </div>
{/if}

<style>
  /* The selected tab is underlined rather than filled, so the strip reads as
     panel chrome instead of a row of buttons. */
  .panel-tab[aria-selected="true"]::after {
    content: "";
    position: absolute;
    inset-inline: 0.75rem;
    bottom: 0;
    height: 1px;
    background: var(--muted-foreground);
  }
</style>
