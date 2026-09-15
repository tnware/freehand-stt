<script lang="ts">
  import type { InstanceStatus } from "$bindings/managedruntime";
  import { runtimePresentation } from "$lib/utils/managedRuntime";

  let {
    instances,
    selected,
    panelID,
    onSelect,
  }: {
    instances: readonly InstanceStatus[];
    selected: string;
    panelID: string;
    onSelect: (id: string) => void;
  } = $props();

  const uid = $props.id();
  let strip: HTMLDivElement | undefined = $state();
  const unavailable = $derived(
    Boolean(selected) && !instances.some((row) => row.instance.id === selected),
  );

  function tone(row: InstanceStatus) {
    if (!row.status.supported) return "unavailable";
    if (row.status.state === "running") return "running";
    if (row.status.state === "error") return "error";
    if (
      ["starting", "installing", "stopping"].includes(row.status.state) ||
      row.status.operation?.outcome === "running"
    )
      return "busy";
    return "stopped";
  }

  const tabs = $derived([
    ...(unavailable
      ? [
          {
            id: selected,
            name: "Selected runtime unavailable",
            stateLabel: "Choose another runtime to inspect its output.",
            tone: "unavailable",
            available: false,
          },
        ]
      : []),
    ...instances.map((row) => ({
      id: row.instance.id,
      name: row.instance.name,
      stateLabel: runtimePresentation(row.status).label,
      tone: tone(row),
      available: true,
    })),
  ]);

  function reveal(id: string, focus = false) {
    const button = Array.from(
      strip?.querySelectorAll<HTMLButtonElement>("[data-runtime-id]") ?? [],
    ).find((item) => item.dataset.runtimeId === id);
    if (focus) button?.focus({ preventScroll: true });
    button?.scrollIntoView({ block: "nearest", inline: "nearest" });
  }

  function navigate(event: KeyboardEvent, id: string) {
    const index = instances.findIndex((row) => row.instance.id === id);
    let next: number;
    if (event.key === "ArrowRight") next = (index + 1) % instances.length;
    else if (event.key === "ArrowLeft")
      next = (Math.max(0, index) - 1 + instances.length) % instances.length;
    else if (event.key === "Home") next = 0;
    else if (event.key === "End") next = instances.length - 1;
    else return;
    event.preventDefault();
    const target = instances[next]?.instance.id;
    if (!target) return;
    onSelect(target);
    reveal(target, true);
  }

  // View output may select a runtime outside this strip. Reveal its tab without
  // moving keyboard focus from the control that opened the panel.
  $effect(() => {
    reveal(selected);
  });
</script>

<div
  bind:this={strip}
  class="runtime-tabs"
  role="tablist"
  aria-label="Output runtimes"
  aria-orientation="horizontal"
>
  {#each tabs as tab, index (tab.id)}
    <button
      type="button"
      role="tab"
      class="workbench-tab runtime-tab"
      aria-label={tab.name}
      aria-describedby={`${uid}-${index}-state`}
      aria-selected={selected === tab.id}
      aria-disabled={tab.available ? undefined : true}
      aria-controls={panelID}
      tabindex={selected === tab.id || (!selected && index === 0) ? 0 : -1}
      data-runtime-id={tab.id}
      title={`${tab.name} · ${tab.stateLabel}`}
      onclick={() => {
        if (tab.available) onSelect(tab.id);
      }}
      onkeydown={(event) => navigate(event, tab.id)}
    >
      <span class="state-dot {tab.tone}" aria-hidden="true"></span>
      <span class="runtime-name">{tab.name}</span>
      <span id={`${uid}-${index}-state`} class="sr-only">{tab.stateLabel}</span>
    </button>
  {/each}
  {#if !instances.length && !unavailable}
    <span class="empty-label">No runtime installed</span>
  {/if}
</div>

<style>
  .runtime-tabs {
    display: flex;
    min-width: 0;
    flex: none;
    overflow-x: auto;
    overscroll-behavior-x: contain;
    scrollbar-width: thin;
    border-bottom: 1px solid var(--hairline);
    background: var(--well);
  }

  .runtime-tab {
    height: 28px;
    max-width: 240px;
    flex: none;
    align-items: center;
    gap: 7px;
    padding: 0 12px;
    border-right: 1px solid var(--hairline);
  }

  .runtime-name {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .runtime-tab[aria-selected="true"] {
    background: var(--background);
  }

  .runtime-tab[aria-selected="true"]::after {
    inset-inline: 0;
  }

  .runtime-tab[aria-disabled="true"] {
    cursor: default;
  }

  .state-dot {
    width: 6px;
    height: 6px;
    flex: none;
    border-radius: 50%;
    background: var(--muted-foreground);
  }

  .running {
    background: var(--success);
  }

  .busy {
    background: var(--accent-text);
  }

  .error {
    background: var(--destructive);
  }

  .unavailable {
    background: transparent;
    box-shadow: inset 0 0 0 1px var(--muted-foreground);
  }

  .empty-label {
    display: flex;
    height: 28px;
    align-items: center;
    padding: 0 12px;
    color: var(--muted-foreground);
    font-size: 12px;
  }

  @media (forced-colors: active) {
    .runtime-tab[aria-selected="true"]::after {
      background: Highlight;
    }

    .state-dot {
      border: 1px solid CanvasText;
    }
  }
</style>
