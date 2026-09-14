<script lang="ts">
  import type { InstanceStatus } from "$bindings/managedruntime";
  import { type Catalog, type Connection } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import {
    connectionTargetLabel,
    connectionProvider,
    connectionMatches,
    connectionWorkflows,
  } from "$lib/utils/connectionChoices";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import SearchIcon from "@lucide/svelte/icons/search";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  let {
    catalog,
    instances = [],
    selected = "",
    creating = false,
    busy = false,
    onSelect,
    onAdd,
  }: {
    catalog: Catalog;
    instances?: InstanceStatus[];
    selected?: string;
    creating?: boolean;
    busy?: boolean;
    onSelect: (connection: Connection) => void;
    onAdd: () => void;
  } = $props();
  let query = $state("");
  const entries = $derived(
    (catalog.entries ?? [])
      .filter((c) => connectionMatches(c, query, instances))
      .sort((a, b) => a.name.localeCompare(b.name)),
  );
</script>

<div class="flex h-full min-h-0 flex-col">
  <div
    class="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline p-3"
  >
    <Button
      variant={creating ? "soft" : "default"}
      class="justify-start"
      disabled={busy}
      onclick={() => {
        query = "";
        onAdd();
      }}><PlusIcon />Add connection</Button
    >
    <div class="relative min-w-40 flex-1">
      <SearchIcon
        class="pointer-events-none absolute left-2.5 top-2.5 size-4 text-muted-foreground"
      />
      <input
        aria-label="Search saved connections"
        placeholder="Search connections…"
        bind:value={query}
        class="h-9 w-full rounded-md border border-input bg-background pl-8 pr-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
    </div>
  </div>
  <nav
    aria-label="Saved connections"
    class="connection-list-content min-h-0 flex-1 overflow-y-auto overscroll-contain space-y-1.5 p-3"
  >
    {#each entries as connection (connection.id)}
      {@const active = connectionWorkflows.filter(
        (role) => catalog.selected?.[role.id] === connection.id,
      )}
      {@const instance = instances.find(
        (item) => item.instance.id === connection.details.managedInstanceID,
      )}
      {@const runtimeView = runtimePresentation(instance?.status)}
      {@const metadata = connection.details.managedInstanceID
        ? [connection.builtIn ? "Built-in" : "Local runtime", runtimeView.backend]
            .filter(Boolean)
            .join(" · ")
        : connectionTargetLabel(connection, instances)}
      {@const usedBy = active.map((role) => role.label).join(", ")}
      <button
        type="button"
        disabled={busy}
        aria-current={connection.id === selected ? "true" : undefined}
        onclick={() => onSelect(connection)}
        class={`connection-row w-full items-center gap-x-3 gap-y-1.5 rounded-xl border px-3 py-3 text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 ${connection.id === selected ? "border-accent-edge bg-accent-wash" : "border-hairline bg-card hover:border-accent-edge hover:bg-subtle-fill-hover"}`}
      >
        <span
          class="connection-icon flex size-10 shrink-0 items-center justify-center rounded-lg border border-hairline bg-background"
        >
          <ProviderIcon
            profile={connectionProvider(connection, instances)}
            size={24}
          />
        </span>
        <span class="connection-copy min-w-0">
          <span class="block truncate text-sm font-semibold text-foreground"
            >{connection.name}</span
          >
          <span
            class="mt-1 block truncate text-xs text-secondary-foreground"
            title={metadata}>{metadata}</span
          >
          <span class="mt-0.5 block truncate text-[11px]" title={usedBy}>
            <span class="text-ink-quiet">Used by</span>
            <span class={usedBy ? "text-secondary-foreground" : "text-ink-quiet"}
              >{usedBy || "nothing yet"}</span
            >
          </span>
        </span>
        <span class="connection-status min-w-0">
          {#if connection.details.managedInstanceID}
            <StatusBadge
              tone={instance?.status.state === "running"
                ? "success"
                : instance?.status.state === "error"
                  ? "danger"
                  : instance?.status.state === "starting"
                    ? "accent"
                    : "neutral"}
              dot>{runtimeView.label}</StatusBadge
            >
          {:else if active.length}
            <span title={active.map((role) => role.label).join(" · ")}
              ><StatusBadge tone="accent">In use</StatusBadge></span
            >
          {/if}
        </span>
        <span class="connection-chevron"
          ><ChevronRightIcon class="size-4 text-secondary-foreground" /></span
        >
      </button>
    {:else}<p class="px-3 py-6 text-center text-sm text-muted-foreground">
        {query
          ? "No matching connections."
          : "Add a server connection or set up a local runtime. Its built-in connection appears automatically."}
      </p>{/each}
  </nav>
  <p
    class="shrink-0 border-t border-hairline px-4 py-2 text-xs text-muted-foreground"
  >
    {catalog.entries?.length ?? 0} connections
  </p>
</div>

<style>
  .connection-list-content {
    container-type: inline-size;
  }
  .connection-row {
    display: grid;
    grid-template-columns: 2.5rem minmax(0, 1fr) auto 1rem;
  }
  @container (max-width: 20rem) {
    .connection-row {
      grid-template-columns: 2.5rem minmax(0, 1fr) 1rem;
    }
    .connection-icon {
      grid-column: 1;
      grid-row: 1 / span 2;
    }
    .connection-copy {
      grid-column: 2;
      grid-row: 1;
    }
    .connection-status {
      grid-column: 2;
      grid-row: 2;
    }
    .connection-status:empty {
      display: none;
    }
    .connection-chevron {
      grid-column: 3;
      grid-row: 1 / span 2;
    }
  }
</style>
