<script lang="ts">
  import { type Catalog, type Change, type Purpose } from "$bindings/savedconnection";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Button } from "$lib/components/ui/button";
  let {
    catalog,
    purpose,
    dirty,
    busy,
    onChange,
    onManage,
    onBrowse,
    onAdd,
  }: {
    catalog: Catalog;
    purpose: Purpose;
    dirty: boolean;
    busy: boolean;
    onChange: (change: Change) => Promise<boolean>;
    onManage: () => void;
    onBrowse: () => void;
    onAdd: () => void;
  } = $props();
  const entries = $derived((catalog.entries ?? []).filter((c) => c.uses?.includes(purpose)));
  const selected = $derived(entries.find((c) => c.id === catalog.selected?.[purpose]));
</script>

<section
  class="rounded-xl border border-hairline bg-layer-fill px-5 py-4"
  aria-label="Active connection"
>
  <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
    <label for={`saved-connection-${purpose}`} class="text-sm font-medium">Connection</label>
    <div class="min-w-44 flex-1">
      <ConnectionSelect
        id={`saved-connection-${purpose}`}
        {catalog}
        {purpose}
        disabled={busy}
        {onChange}
        {onAdd}
        onManage={onBrowse}
      />
    </div>
    <Button variant="ghost" size="sm" disabled={busy} onclick={selected ? onManage : onAdd}
      >{selected ? "Edit connection" : "Add connection"}</Button
    >
  </div>
  {#if selected}<p
      class="mt-2 truncate text-xs text-muted-foreground"
      title={selected.details.baseURL}
    >
      {selected.details.baseURL}
    </p>
  {:else}<p class="mt-2 text-[13px] text-muted-foreground">
      {entries.length
        ? "Choose a saved connection to configure this feature."
        : "Add a server, then choose its model here."}
    </p>{/if}
  {#if dirty}<p class="mt-2 text-xs text-muted-foreground">
      Save or discard your edits before switching connections.
    </p>{/if}
</section>
