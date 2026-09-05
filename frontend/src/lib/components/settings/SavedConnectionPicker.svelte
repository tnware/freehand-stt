<script lang="ts">
  import { Action, type Catalog, type Change, type Purpose } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
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
  async function select(id: string) {
    const next = id === "none" ? "" : id;
    if (next === (selected?.id ?? "")) return;
    await onChange({ action: Action.Select, purpose, id: next, name: "", replacementID: "" });
  }
</script>

<div class="space-y-3 rounded-xl border border-hairline bg-layer-fill p-4">
  <div class="flex items-center justify-between gap-2">
    <label for={`saved-connection-${purpose}`} class="text-xs font-medium">Active connection</label
    ><Button variant="link" size="sm" onclick={onManage}
      >{selected ? "Edit connection" : "Manage connections"}</Button
    >
  </div>
  <Select.Root
    type="single"
    value={selected?.id ?? "none"}
    onValueChange={select}
    disabled={busy || dirty}
  >
    <Select.Trigger id={`saved-connection-${purpose}`} class="w-full"
      ><span class="flex min-w-0 items-center gap-2"
        >{#if selected}<ProviderIcon
            profile={selected.details.compatibilityProfile}
            size={20}
          />{/if}<span class="truncate">{selected?.name ?? "None selected"}</span></span
      ></Select.Trigger
    >
    <Select.Content
      ><Select.Item value="none" label="None selected">None selected</Select.Item
      >{#each entries as c (c.id)}<Select.Item value={c.id} label={c.name}
          ><ProviderIcon profile={c.details.compatibilityProfile} size={20} />{c.name}</Select.Item
        >{/each}</Select.Content
    >
  </Select.Root>
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
