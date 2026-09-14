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
    hasHistory,
    historyCount,
    working,
  }: {
    result: Snippet;
    history: Snippet;
    hasHistory: boolean;
    historyCount: number;
    working: boolean;
  } = $props();

  // The chain already owns the full width above this, so history sits under
  // the transcript rather than beside it: a side pane would squeeze the one
  // column that actually holds prose.
  const roomy = new MediaQuery("(min-height: 560px)");
  let collapsed = $state(false);
  let selectedView = $state<"result" | "history">("result");
  const visibleView = $derived(working ? "result" : selectedView);
  const stacked = $derived(roomy.current && hasHistory);
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
          <span
            class="px-3 text-[10px] font-semibold tracking-[0.08em] text-secondary-foreground uppercase"
            >Recent · {historyCount}</span
          >
          <Button
            variant="ghost"
            size="xs"
            class="mr-1 size-6 rounded-sm p-0"
            aria-expanded={!collapsed}
            aria-label={collapsed ? "Show recent history" : "Hide recent history"}
            title={collapsed ? "Show recent history" : "Hide recent history"}
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
          <div class="flex min-h-0 flex-1 flex-col">{@render history()}</div>
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
