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
    onAdd,
  }: {
    catalog: Catalog;
    purpose: Purpose;
    dirty: boolean;
    busy: boolean;
    onChange: (change: Change) => Promise<boolean>;
    onManage: () => void;
    onAdd: () => void;
  } = $props();
  const entries = $derived((catalog.entries ?? []).filter((c) => c.uses?.includes(purpose)));
  const selected = $derived(entries.find((c) => c.id === catalog.selected?.[purpose]));
</script>

<div class="space-y-3 rounded-xl border border-hairline bg-layer-fill p-4">
  <div class="flex items-center justify-between gap-2">
    <label for={`saved-connection-${purpose}`} class="text-xs font-medium">Active connection</label
    ><Button variant="link" size="sm" disabled={busy} onclick={selected ? onManage : onAdd}
      >{selected ? "Edit connection" : "Add connection"}</Button
    >
  </div>
  <ConnectionSelect
    id={`saved-connection-${purpose}`}
    {catalog}
    {purpose}
    disabled={busy}
    {onChange}
    {onAdd}
  />
  {#if selected}<p class="break-all text-xs text-muted-foreground">
      {selected.details.baseURL}
    </p>{:else}<p class="text-xs text-muted-foreground">
      {entries.length
        ? "Choose a saved connection to configure this feature."
        : "Add a server to get started. You’ll return here to choose its model."}
    </p>{/if}
  <p class="text-[11px] leading-relaxed text-muted-foreground">
    {dirty
      ? "You’ll be asked to save or discard these edits before changing connections."
      : "Connection selection applies immediately. Model and task options below apply when you press Save settings."}
  </p>
</div>
