<script lang="ts">
  import { PANES, workflowBlockedReason, type PaneID } from "$lib/panes";

  let {
    pane,
    onSelect,
    voiceActive = false,
    fileWorking = false,
  }: {
    pane: PaneID;
    onSelect: (id: PaneID) => void;
    voiceActive?: boolean;
    fileWorking?: boolean;
  } = $props();

  const top = PANES.filter((entry) => entry.place === "top");
  const foot = PANES.filter((entry) => entry.place === "foot");
</script>

<!--
  One navigation for the whole window. The rail replaced a row of tabs that
  could only reach the three workflows; configuration is a place on the same
  rail rather than a separate window, so there is never a second navigation
  competing with this one.
-->
<nav
  class="flex w-12 shrink-0 flex-col border-r border-hairline bg-well"
  aria-label="Workspace"
>
  {#each [top, foot] as group, index (index)}
    {#if index === 1}<div class="flex-1"></div>{/if}
    {#each group as entry (entry.id)}
      {@const blocked = workflowBlockedReason(
        entry.id,
        voiceActive,
        fileWorking,
      )}
      {@const active = pane === entry.id}
      <button
        type="button"
        class="rail-item relative grid h-[42px] w-12 place-items-center text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-40 aria-[current=page]:bg-accent-wash aria-[current=page]:text-accent-text"
        aria-current={active ? "page" : undefined}
        aria-label={entry.label}
        title={blocked || entry.label}
        disabled={!!blocked}
        onclick={() => onSelect(entry.id)}
      >
        <entry.icon class="size-[18px]" aria-hidden="true" />
      </button>
    {/each}
  {/each}
</nav>

<style>
  /* The selected place is marked on the window edge rather than by a filled
     pill, so the rail keeps reading as chrome instead of as a control. */
  .rail-item[aria-current="page"]::before {
    content: "";
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0;
    width: 2px;
    background: var(--primary);
  }
  @media (forced-colors: active) {
    .rail-item[aria-current="page"] {
      outline: 2px solid Highlight;
      outline-offset: -2px;
    }
  }
</style>
