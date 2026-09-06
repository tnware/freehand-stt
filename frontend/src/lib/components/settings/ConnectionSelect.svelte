<script lang="ts">
  import { Action, type Catalog, type Change, type Purpose } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import * as Select from "$lib/components/ui/select";
  let {
    id,
    catalog,
    purpose,
    disabled = false,
    compact = false,
    onChange,
  }: {
    id: string;
    catalog: Catalog;
    purpose: Purpose;
    disabled?: boolean;
    compact?: boolean;
    onChange: (change: Change) => Promise<boolean>;
  } = $props();
  const entries = $derived((catalog.entries ?? []).filter((c) => c.purpose === purpose));
  const selected = $derived(entries.find((c) => c.id === catalog.selected?.[purpose]));
  async function select(id: string) {
    const next = id === "none" ? "" : id;
    if (next === (selected?.id ?? "")) return;
    await onChange({ action: Action.Select, purpose, id: next, name: "", replacementID: "" });
  }
</script>

<Select.Root type="single" value={selected?.id ?? "none"} onValueChange={select} {disabled}>
  <Select.Trigger {id} class={compact ? "h-[30px] min-w-0 flex-1 text-xs" : "w-full"}>
    <span class="flex min-w-0 items-center gap-2">
      {#if selected}<ProviderIcon
          profile={selected.details.compatibilityProfile}
          size={compact ? 16 : 20}
        />{/if}
      <span class="truncate">{selected?.name ?? "None selected"}</span>
    </span>
  </Select.Trigger>
  <Select.Content>
    <Select.Item value="none" label="None selected">None selected</Select.Item>
    {#each entries as c (c.id)}
      <Select.Item value={c.id} label={c.name}>
        <ProviderIcon profile={c.details.compatibilityProfile} size={20} />{c.name}
      </Select.Item>
    {/each}
  </Select.Content>
</Select.Root>
