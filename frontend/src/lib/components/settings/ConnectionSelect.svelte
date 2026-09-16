<script lang="ts">
  import * as Picker from "$lib/components/ui/combobox";
  import * as WindowingService from "$bindings/windowing/service";
  import Settings2Icon from "@lucide/svelte/icons/settings-2";
  import { tick, getContext } from "svelte";
  import { TASK_CONNECTION_NAVIGATION } from "$lib/shell-navigation.svelte";
  import type { ConnectionManagerRequest } from "$bindings/windowing";
  const openTaskConnection = getContext<
    ((request: ConnectionManagerRequest) => void) | undefined
  >(TASK_CONNECTION_NAVIGATION);
  import { session } from "$lib/stores/session.svelte";
  import type { InstanceStatus } from "$bindings/managedruntime";
  import { Combobox } from "bits-ui";
  import {
    Action,
    type Catalog,
    type Change,
    type Purpose,
  } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import {
    connectionTargetLabel,
    connectionProvider,
    connectionMatches,
  } from "$lib/utils/connectionChoices";
  import CheckIcon from "@lucide/svelte/icons/check";
  import PlusIcon from "@lucide/svelte/icons/plus";
  let {
    id,
    catalog,
    runtimeInstances,
    purpose,
    disabled = false,
    compact = false,
    onChange,
    onAdd,
    onManage,
  }: {
    id: string;
    catalog: Catalog;
    runtimeInstances?: InstanceStatus[];
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
  const instances = $derived(runtimeInstances ?? session.runtime.instances);
  const entries = $derived(
    (catalog.entries ?? []).filter(
      (c) => c.uses?.includes(purpose) || c.id === catalog.selected?.[purpose],
    ),
  );
  const selected = $derived(
    entries.find((c) => c.id === catalog.selected?.[purpose]),
  );
  const matches = $derived(
    entries
      .filter((c) => connectionMatches(c, query, instances))
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
      if (
        await onChange({ action: Action.Select, purpose, id: next, name: "" })
      )
        open = false;
    } finally {
      choosing = false;
    }
  }
  async function manage() {
    open = false;
    await tick();
    document.getElementById(id)?.focus();
    if (onManage) onManage();
    else if (openTaskConnection)
      openTaskConnection({ id: "", purpose, create: false });
    else {
      try {
        await WindowingService.OpenConnectionManager({
          id: "",
          purpose,
          create: false,
        });
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
  bind:value={() => selected?.id ?? "none", (next) => void select(next)}
  inputValue={open ? query : (selected?.name ?? "")}
  bind:open
  items={choices}
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
        <ProviderIcon
          profile={connectionProvider(selected, instances)}
          size={16}
        />
      </span>{/if}
    <Combobox.Input
      data-slot="combobox-input"
      {id}
      aria-label="Choose connection"
      placeholder={open ? "Search connections…" : "Choose or add a connection…"}
      class={`w-full min-w-0 rounded-md border border-input bg-well pr-9 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 ${compact ? "h-7 text-xs" : "h-8"} ${selected && !open ? "pl-9" : "pl-3"}`}
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
    <Picker.Trigger aria-label="Show connections" />
  </div>
  <Picker.Content>
    {#each matches as c (c.id)}
      <Picker.Item value={c.id} label={c.name}>
        <ProviderIcon profile={connectionProvider(c, instances)} size={20} />
        <span class="min-w-0 flex-1"
          ><span class="block truncate text-sm font-medium">{c.name}</span><span
            class="block truncate text-xs text-muted-foreground"
            >{connectionTargetLabel(c, instances)}</span
          ></span
        >
        {#if c.id === selected?.id}<CheckIcon class="size-4 shrink-0" />{/if}
      </Picker.Item>
    {:else}<p class="px-3 py-4 text-sm text-muted-foreground">
        {query.trim()
          ? "No matching connections."
          : "No connections for this workflow yet."}
      </p>{/each}
    {#if !query.trim()}<Picker.Item
        value="none"
        label="None selected"
        class="mt-1 justify-between border-t border-hairline py-2 text-xs text-muted-foreground"
      >
        None selected {#if !selected}<CheckIcon class="size-3.5" />{/if}
      </Picker.Item>{/if}
    {#snippet footer()}
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
    {/snippet}
  </Picker.Content>
</Combobox.Root>
