<script lang="ts">
  import type { Snippet } from "svelte";
  import { MediaQuery } from "svelte/reactivity";
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
  const desktop = new MediaQuery("(min-width: 900px)");
  let selectedView = $state<"result" | "history">("result");
  const visibleView = $derived(working ? "result" : selectedView);
</script>

{#if desktop.current && hasHistory}
  <Resizable.PaneGroup
    direction="horizontal"
    autoSaveId="freehand-workspace-v1"
    keyboardResizeBy={2}
  >
    <Resizable.Pane id="current-result-pane" defaultSize={68} minSize={50}>
      {@render result()}
    </Resizable.Pane>
    <Resizable.Handle
      withHandle
      aria-label="Resize current result and history"
    />
    <Resizable.Pane id="recent-history-pane" defaultSize={32} minSize={25}>
      {@render history()}
    </Resizable.Pane>
  </Resizable.PaneGroup>
{:else}
  {#if hasHistory}
    <div class="flex shrink-0 gap-1" role="group" aria-label="Workspace view">
      <Button
        variant={visibleView === "result" ? "secondary" : "ghost"}
        size="sm"
        aria-pressed={visibleView === "result"}
        onclick={() => (selectedView = "result")}>Result</Button
      >
      <Button
        variant={visibleView === "history" ? "secondary" : "ghost"}
        size="sm"
        aria-pressed={visibleView === "history"}
        disabled={working}
        onclick={() => (selectedView = "history")}
        >History · {historyCount}</Button
      >
    </div>
  {/if}
  <div class="flex min-h-0 flex-1 flex-col">
    {#if hasHistory && visibleView === "history"}{@render history()}{:else}{@render result()}{/if}
  </div>
{/if}
