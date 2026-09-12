<script lang="ts">
  import * as WindowingService from "$bindings/windowing/service";
  import Settings2Icon from "@lucide/svelte/icons/settings-2";
  import { tick } from "svelte";
  import { session } from "$lib/stores/session.svelte";
  import { Combobox } from "bits-ui";
  import { Action, type Catalog, type Change, type Purpose } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import { connectionMatches } from "$lib/utils/connectionChoices";
  import CheckIcon from "@lucide/svelte/icons/check";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";
  let {
    id,
    catalog,
    purpose,
    disabled = false,
    compact = false,
    onChange,
    onAdd,
    onManage,
  }: {
    id: string;
    catalog: Catalog;
    purpose: Purpose;
    disabled?: boolean;
    compact?: boolean;
    onChange: (change: Change) => Promise<boolean>;
    onAdd?: () => void;
    onManage?: () => void;
  } = $props();
  let open = $state(false),
    query = $state(""),
    choosing = $state(false);
  const entries = $derived((catalog.entries ?? []).filter((c) => c.uses?.includes(purpose)));
  const selected = $derived(entries.find((c) => c.id === catalog.selected?.[purpose]));
  const matches = $derived(
    entries
      .filter((c) => connectionMatches(c, query))
      .sort(
        (a, b) =>
          Number(b.id === selected?.id) - Number(a.id === selected?.id) ||
          a.name.localeCompare(b.name),
      ),
  );
  const choices = $derived([
    ...matches.map((c) => ({ value: c.id, label: c.name })),
    ...(!query.trim() ? [{ value: "none", label: "None selected" }] : []),
  ]);
  async function select(value: string) {
    if (disabled || choosing) return;
    const next = value === "none" ? "" : value;
    if (next === (selected?.id ?? "")) {
      open = false;
      return;
    }
    choosing = true;
    try {
      if (await onChange({ action: Action.Select, purpose, id: next, name: "" })) open = false;
    } finally {
      choosing = false;
    }
  }
  async function manage() {
    open = false;
    await tick();
    document.getElementById(id)?.focus();
    if (onManage) onManage();
    else {
      try {
        await WindowingService.OpenConnectionManager({ id: "", purpose, create: false });
      } catch (cause) {
        session.messages.fail(cause);
      }
    }
  }
  async function add() {
    open = false;
    await tick();
    document.getElementById(id)?.focus();
    onAdd?.();
  }
</script>

<Combobox.Root
  type="single"
  value={selected?.id ?? "none"}
  inputValue={open ? query : (selected?.name ?? "")}
  bind:open
  items={choices}
  onValueChange={select}
  allowDeselect={false}
  onOpenChange={(next) => {
    if (!next) query = "";
  }}
  disabled={disabled || choosing}
>
  <div class="relative min-w-0 flex-1">
    {#if selected && !open}<span
        class="pointer-events-none absolute inset-y-0 left-3 flex items-center"
      >
        <ProviderIcon profile={selected.details.compatibilityProfile} size={16} />
      </span>{/if}
    <Combobox.Input
      {id}
      aria-label="Choose connection"
      placeholder={open ? "Search connections…" : "Choose or add a connection…"}
      class={`w-full min-w-0 rounded-lg border border-input bg-background pr-9 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 ${compact ? "h-8 text-xs" : "h-9"} ${selected && !open ? "pl-9" : "pl-3"}`}
      oninput={(event) => {
        query = event.currentTarget.value;
        open = true;
      }}
    >
      {#snippet child({ props })}<input
          {...props}
          value={open ? query : (selected?.name ?? "")}
        />{/snippet}
    </Combobox.Input>
    <Combobox.Trigger
      aria-label="Show connections"
      class="absolute inset-y-0 right-0 flex w-9 items-center justify-center text-muted-foreground"
    >
      <ChevronsUpDownIcon class="size-4" />
    </Combobox.Trigger>
  </div>
  <Combobox.Portal
    ><Combobox.Content
      data-slot="combobox-content"
      sideOffset={4}
      class="z-50 flex max-h-[min(24rem,var(--bits-combobox-content-available-height))] w-[var(--bits-combobox-anchor-width)] min-w-64 max-w-[calc(100vw-24px)] flex-col overflow-hidden rounded-lg border border-border bg-popover text-popover-foreground shadow-md"
    >
      <div class="min-h-0 overflow-y-auto overscroll-contain p-1">
        {#each matches as c (c.id)}
          <Combobox.Item
            value={c.id}
            label={c.name}
            class="flex cursor-default items-center gap-3 rounded-sm px-3 py-2.5 outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
          >
            <ProviderIcon profile={c.details.compatibilityProfile} size={20} />
            <span class="min-w-0 flex-1"
              ><span class="block truncate text-sm font-medium">{c.name}</span><span
                class="block truncate text-xs text-muted-foreground">{c.details.baseURL}</span
              ></span
            >
            {#if c.id === selected?.id}<CheckIcon class="size-4 shrink-0" />{/if}
          </Combobox.Item>
        {:else}<p class="px-3 py-4 text-sm text-muted-foreground">
            {query.trim() ? "No matching connections." : "No connections for this workflow yet."}
          </p>{/each}
        {#if !query.trim()}<Combobox.Item
            value="none"
            label="None selected"
            class="mt-1 flex cursor-default items-center justify-between rounded-sm border-t border-hairline px-3 py-2 text-xs text-muted-foreground outline-none data-highlighted:bg-accent"
          >
            None selected {#if !selected}<CheckIcon class="size-3.5" />{/if}
          </Combobox.Item>{/if}
      </div>
      <div class="shrink-0 border-t border-hairline p-1">
        {#if onAdd}<Button
            variant="ghost"
            class="w-full justify-start"
            disabled={disabled || choosing}
            onclick={add}><PlusIcon />Add connection…</Button
          >{/if}
        <Button
          variant="ghost"
          class="w-full justify-start"
          disabled={disabled || choosing}
          onclick={manage}><Settings2Icon />Manage connections…</Button
        >
      </div>
    </Combobox.Content></Combobox.Portal
  >
</Combobox.Root>
