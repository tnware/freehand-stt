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
  }: {
    catalog: Catalog;
    purpose: Purpose;
    dirty: boolean;
    busy: boolean;
    onChange: (change: Change) => Promise<boolean>;
    onManage: () => void;
  } = $props();
  const entries = $derived((catalog.entries ?? []).filter((c) => c.purpose === purpose));
  const selected = $derived(entries.find((c) => c.id === catalog.selected?.[purpose]));
</script>

<div class="space-y-3 rounded-xl border border-hairline bg-layer-fill p-4">
  <div class="flex items-center justify-between gap-2">
    <label for={`saved-connection-${purpose}`} class="text-xs font-medium">Active connection</label
    ><Button variant="link" size="sm" onclick={onManage}
      >{selected ? "Edit connection" : "Manage connections"}</Button
    >
  </div>
  <ConnectionSelect
    id={`saved-connection-${purpose}`}
    {catalog}
    {purpose}
    disabled={busy || dirty}
    {onChange}
  />
  {#if selected}<p class="break-all text-xs text-muted-foreground">
      {selected.details.baseURL}
    </p>{:else}<p class="text-xs text-muted-foreground">
      {entries.length
        ? "Choose a saved connection to configure this feature."
        : "Create a connection on the Connections page, then select it here."}
    </p>{/if}
  <p class="text-[11px] leading-relaxed text-muted-foreground">
    {dirty
      ? "Save or discard feature settings before switching connections."
      : "Switching applies immediately and clears this feature’s model choice. Options below save to this feature, not to the connection."}
  </p>
</div>
