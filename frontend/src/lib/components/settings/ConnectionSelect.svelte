<script lang="ts">
  import { tick } from "svelte";
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
    onAdd,
  }: {
    id: string;
    catalog: Catalog;
    purpose: Purpose;
    disabled?: boolean;
    compact?: boolean;
    onChange: (change: Change) => Promise<boolean>;
    onAdd?: () => void;
  } = $props();
  const entries = $derived((catalog.entries ?? []).filter((c) => c.uses?.includes(purpose)));
  const selected = $derived(entries.find((c) => c.id === catalog.selected?.[purpose]));
  async function select(value: string) {
    if (value === "add-connection") {
      await tick();
      document.getElementById(id)?.focus();
      onAdd?.();
      return;
    }
    const next = value === "none" ? "" : value;
    if (next === (selected?.id ?? "")) return;
    await onChange({ action: Action.Select, purpose, id: next, name: "" });
  }
</script>

<Select.Root
  type="single"
  bind:value={
    () => selected?.id ?? "none",
    (value) => {
      void select(value);
    }
  }
  {disabled}
>
  <Select.Trigger {id} class={compact ? "h-[30px] min-w-0 flex-1 text-xs" : "w-full"}>
    <span class="flex min-w-0 items-center gap-2">
      {#if selected}<ProviderIcon
          profile={selected.details.compatibilityProfile}
          size={compact ? 16 : 20}
        />{/if}
      <span class="truncate"
        >{selected?.name ?? (onAdd ? "Choose or add a connection" : "None selected")}</span
      >
    </span>
  </Select.Trigger>
  <Select.Content>
    <Select.Item value="none" label="None selected">None selected</Select.Item>
    {#each entries as c (c.id)}
      <Select.Item value={c.id} label={c.name}>
        <ProviderIcon profile={c.details.compatibilityProfile} size={20} />{c.name}
      </Select.Item>
    {/each}
    {#if onAdd}
      <Select.Separator />
      <Select.Item value="add-connection" label="Add connection…">Add connection…</Select.Item>
    {/if}
  </Select.Content>
</Select.Root>
