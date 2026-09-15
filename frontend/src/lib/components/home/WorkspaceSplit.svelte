<script lang="ts">
  import { tick, type Snippet } from "svelte";
  import { MediaQuery } from "svelte/reactivity";
  import PanelTabs from "$lib/components/shell/PanelTabs.svelte";
  import * as Resizable from "$lib/components/ui/resizable";

  let {
    result,
    history,
    output,
    diagnostics,
    hasHistory,
    historyCount,
    working,
  }: {
    result: Snippet;
    history: Snippet;
    output?: Snippet;
    diagnostics?: Snippet;
    hasHistory: boolean;
    historyCount: number;
    working: boolean;
  } = $props();

  const uid = $props.id();
  const panelID = `${uid}-panel`;
  let tab = $state("recent");
  let compactTab = $state("result");
  let collapsed = $state(false);
  let workspace = $state<HTMLDivElement>();
  async function setCollapsed(value: boolean) {
    const focused = document.activeElement;
    const restore = !!focused && workspace?.contains(focused);
    const wasTab = focused?.getAttribute("role") === "tab";
    collapsed = value;
    await tick();
    if (!restore || collapsed !== value) return;
    workspace
      ?.querySelector<HTMLButtonElement>(
        wasTab
          ? '[role="tab"][aria-selected="true"]'
          : `button[aria-label="${value ? "Show panel" : "Hide panel"}"]`,
      )
      ?.focus();
  }
  const roomy = new MediaQuery("(min-height: 560px)");
  const tabs = $derived([
    { id: "recent", label: hasHistory ? `Recent · ${historyCount}` : "Recent" },
    ...(output ? [{ id: "output", label: "Runtime output" }] : []),
    ...(diagnostics ? [{ id: "diagnostics", label: "Diagnostics" }] : []),
  ]);
  // A new capture returns the compact view to its transcript. The user can
  // still inspect diagnostics after that initial transition.
  $effect(() => {
    if (working) compactTab = "result";
  });
</script>

{#snippet panelContent(selected: string)}
  {#if selected === "output" && output}{@render output()}
  {:else if selected === "diagnostics" && diagnostics}{@render diagnostics()}
  {:else}{@render history()}{/if}
{/snippet}

<div bind:this={workspace} class="flex min-h-0 flex-1 flex-col">
  {#if roomy.current && !collapsed}
    <Resizable.PaneGroup
      direction="vertical"
      autoSaveId="freehand-workspace-v2"
      keyboardResizeBy={2}
    >
      <Resizable.Pane id="current-result-pane" defaultSize={70} minSize={35}
        >{@render result()}</Resizable.Pane
      >
      <Resizable.Handle aria-label="Resize transcript and panel" />
      <Resizable.Pane id="recent-history-pane" defaultSize={30} minSize={18}>
        <div class="flex h-full min-h-0 flex-col">
          <PanelTabs
            {tabs}
            bind:active={tab}
            bind:collapsed={() => collapsed, setCollapsed}
            {panelID}
          />
          <div
            id={panelID}
            role="tabpanel"
            aria-label={tabs.find((item) => item.id === tab)?.label}
            class="flex min-h-0 flex-1 flex-col"
          >
            {@render panelContent(tab)}
          </div>
        </div>
      </Resizable.Pane>
    </Resizable.PaneGroup>
  {:else if roomy.current}
    <div class="flex min-h-0 flex-1 flex-col">{@render result()}</div>
    <PanelTabs
      {tabs}
      bind:active={tab}
      bind:collapsed={() => collapsed, setCollapsed}
    />
  {:else}
    <PanelTabs
      tabs={[{ id: "result", label: "Transcript" }, ...tabs]}
      bind:active={compactTab}
      collapsible={false}
      {panelID}
      label="Workspace view"
    />
    <div
      id={panelID}
      role="tabpanel"
      aria-label={compactTab === "result"
        ? "Transcript"
        : tabs.find((item) => item.id === compactTab)?.label}
      class="flex min-h-0 flex-1 flex-col"
    >
      {#if compactTab === "result"}{@render result()}{:else}{@render panelContent(
          compactTab,
        )}{/if}
    </div>
  {/if}
</div>
