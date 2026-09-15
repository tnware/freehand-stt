<script lang="ts">
  import type { InstanceStatus } from "$bindings/managedruntime";
  import { type Catalog, type Connection } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import {
    connectionTargetLabel,
    connectionProvider,
    connectionMatches,
    connectionWorkflows,
  } from "$lib/utils/connectionChoices";
  import { endpointHost } from "$lib/utils/endpoint";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import { runtimePresentation } from "$lib/utils/managedRuntime";
  import SearchIcon from "@lucide/svelte/icons/search";
  import PlusIcon from "@lucide/svelte/icons/plus";
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
    class="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-5 py-2"
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
        class="pointer-events-none absolute left-2.5 top-2 size-4 text-muted-foreground"
      />
      <Input
        aria-label="Search saved connections"
        placeholder="Search connections…"
        bind:value={query}
        class="pl-8 pr-2"
      />
    </div>
  </div>
  <div
    class="flex h-[26px] shrink-0 items-center px-5 text-[10px] font-semibold tracking-[0.07em] text-ink-quiet uppercase"
  >
    <span class="min-w-0 flex-1">Name</span>
    <span class="hidden w-[190px] shrink-0 min-[900px]:block">Endpoint</span>
    <span class="hidden w-[130px] shrink-0 min-[1060px]:block">Provider</span>
    <span class="hidden w-[140px] shrink-0 min-[780px]:block">Used by</span>
    <span class="w-[104px] shrink-0">State</span>
  </div>
  <nav
    aria-label="Saved connections"
    class="connection-list-content min-h-0 flex-1 overflow-y-auto overscroll-contain"
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
        ? ["Managed runtime", runtimeView.backend].filter(Boolean).join(" · ")
        : connection.details.compatibilityProfile || "OpenAI-compatible"}
      {@const usedBy = active.map((role) => role.label).join(", ")}
      {@const endpoint = connection.details.managedInstanceID
        ? endpointHost(connection.details.baseURL ?? "")
        : connectionTargetLabel(connection, instances)}
      <button
        type="button"
        disabled={busy}
        aria-current={connection.id === selected ? "true" : undefined}
        onclick={() => onSelect(connection)}
        class={`connection-row flex h-[52px] w-full items-center border-b border-hairline px-5 text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring disabled:opacity-50 ${connection.id === selected ? "bg-accent-wash" : "hover:bg-subtle-fill-hover"}`}
      >
        <span class="flex min-w-0 flex-1 items-center gap-2.5">
          <ProviderIcon
            profile={connectionProvider(connection, instances)}
            size={16}
          />
          <span class="min-w-0 truncate text-[12.5px] text-foreground"
            >{connection.name}</span
          >
          {#if connection.builtIn}
            <span
              class="shrink-0 rounded-sm border border-border px-1.5 py-0.5 text-[10px] text-ink-quiet"
              >built-in</span
            >
          {/if}
        </span>
        <span
          class="hidden w-[190px] shrink-0 truncate font-mono text-[10px] text-secondary-foreground min-[900px]:block"
          title={endpoint}>{endpoint}</span
        >
        <span
          class="hidden w-[130px] shrink-0 truncate text-[11.5px] text-muted-foreground min-[1060px]:block"
          title={metadata}>{metadata}</span
        >
        <span
          class="hidden w-[140px] shrink-0 truncate text-[11.5px] min-[780px]:block {usedBy
            ? 'text-secondary-foreground'
            : 'text-ink-quiet'}"
          title={usedBy || "Not in use"}>{usedBy || "Not in use"}</span
        >
        <span class="flex w-[104px] shrink-0 items-center">
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
            <StatusBadge tone="accent">In use</StatusBadge>
          {/if}
        </span>
      </button>
    {:else}<p class="px-4 py-6 text-center text-[13px] text-muted-foreground">
        {query
          ? "No matching connections."
          : "Add a server connection or set up a local runtime. Its built-in connection appears automatically."}
      </p>{/each}
  </nav>
  <p
    class="shrink-0 border-t border-hairline px-5 py-2 text-xs text-muted-foreground"
  >
    {catalog.entries?.length ?? 0} connections
  </p>
</div>

<style>
  .connection-list-content {
    container-type: inline-size;
  }
</style>
