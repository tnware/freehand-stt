<script lang="ts">
  import { type Catalog, type Connection } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import { connectionMatches, connectionWorkflows } from "$lib/utils/connectionChoices";
  import SearchIcon from "@lucide/svelte/icons/search";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  let {
    catalog,
    selected = "",
    creating = false,
    busy = false,
    onSelect,
    onAdd,
  }: {
    catalog: Catalog;
    selected?: string;
    creating?: boolean;
    busy?: boolean;
    onSelect: (connection: Connection) => void;
    onAdd: () => void;
  } = $props();
  let query = $state("");
  const entries = $derived(
    (catalog.entries ?? [])
      .filter((c) => connectionMatches(c, query))
      .sort((a, b) => a.name.localeCompare(b.name)),
  );
</script>

<div class="flex h-full min-h-0 flex-col">
  <div class="shrink-0 space-y-3 border-b border-hairline p-3">
    <Button
      variant={creating ? "secondary" : "outline"}
      class="w-full justify-start"
      disabled={busy}
      onclick={() => {
        query = "";
        onAdd();
      }}><PlusIcon />Add connection</Button
    >
    <div class="relative">
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
  <nav aria-label="Saved connections" class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-2">
    {#each entries as connection (connection.id)}
      {@const active = connectionWorkflows.filter(
        (role) => catalog.selected?.[role.id] === connection.id,
      )}
      <button
        type="button"
        disabled={busy}
        aria-current={connection.id === selected ? "true" : undefined}
        onclick={() => onSelect(connection)}
        class={`flex w-full items-center gap-3 rounded-md px-3 py-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 ${connection.id === selected ? "bg-accent text-accent-foreground" : "hover:bg-subtle-fill-hover"}`}
      >
        <ProviderIcon profile={connection.details.compatibilityProfile} size={22} />
        <span class="min-w-0 flex-1"
          ><span class="block truncate text-sm font-medium">{connection.name}</span>
          <span
            class="mt-0.5 block truncate text-xs text-muted-foreground"
            title={connection.details.baseURL}>{connection.details.baseURL}</span
          >
          {#if active.length}<span class="mt-1 block truncate text-[11px] text-muted-foreground"
              >In use · {active.map((role) => role.label).join(" · ")}</span
            >{/if}
        </span><ChevronRightIcon class="size-3.5 shrink-0 text-muted-foreground" />
      </button>
    {:else}<p class="px-3 py-6 text-center text-sm text-muted-foreground">
        {query ? "No matching connections." : "Add a server to get started."}
      </p>{/each}
  </nav>
  <p class="shrink-0 border-t border-hairline px-4 py-2 text-xs text-muted-foreground">
    {catalog.entries?.length ?? 0} saved connections
  </p>
</div>
