<script lang="ts">
  import { Action, type Catalog, type Change, type Purpose } from "$bindings/savedconnection";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
  import * as Dialog from "$lib/components/ui/dialog";
  import { Input } from "$lib/components/ui/input";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import CopyIcon from "@lucide/svelte/icons/copy";
  import PencilIcon from "@lucide/svelte/icons/pencil";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

  let {
    catalog,
    purpose,
    dirty,
    busy,
    onChange,
    error = "",
  }: {
    catalog: Catalog;
    purpose: Purpose;
    dirty: boolean;
    busy: boolean;
    onChange: (change: Change) => Promise<boolean>;
    error?: string;
  } = $props();
  const entries = $derived((catalog.entries ?? []).filter((entry) => entry.purpose === purpose));
  const selected = $derived(entries.find((entry) => entry.id === catalog.selected?.[purpose]));
  let open = $state(false);
  let failed = $state(false);
  let action = $state<Action>(Action.Create);
  let name = $state("");
  let replacementID = $state("");
  const replacement = $derived(entries.find((entry) => entry.id === replacementID));
  const title = $derived(
    action === Action.Delete
      ? "Delete connection"
      : action === Action.Rename
        ? "Rename connection"
        : action === Action.Duplicate
          ? "Duplicate connection"
          : "Save as a new connection",
  );

  function begin(next: Action) {
    action = next;
    failed = false;
    name =
      next === Action.Rename
        ? (selected?.name ?? "")
        : next === Action.Duplicate
          ? `${selected?.name ?? "Connection"} copy`
          : "";
    replacementID = entries.find((entry) => entry.id !== selected?.id)?.id ?? "";
    open = true;
  }
  async function select(id: string) {
    if (!id || id === selected?.id) return;
    await onChange({ action: Action.Select, purpose, id, name: "", replacementID: "" });
  }
  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (busy) return;
    const success = await onChange({
      action,
      purpose,
      id: selected?.id ?? "",
      name: name.trim(),
      replacementID,
    });
    failed = !success;
    if (success) open = false;
  }
</script>

{#if selected}
  <div class="rounded-xl border border-hairline bg-layer-fill p-4">
    <div class="mb-2 flex items-center justify-between gap-3">
      <label for={`saved-connection-${purpose}`} class="text-xs font-medium">Saved connection</label
      >
      <span class="text-[11px] text-muted-foreground">{entries.length} saved</span>
    </div>
    <Select.Root type="single" value={selected.id} onValueChange={select} disabled={busy || dirty}>
      <Select.Trigger id={`saved-connection-${purpose}`} class="w-full">
        <span class="flex min-w-0 items-center gap-2"
          ><ProviderIcon profile={selected.details.compatibilityProfile} size={20} /><span
            class="truncate">{selected.name}</span
          ></span
        >
      </Select.Trigger>
      <Select.Content>
        {#each entries as entry (entry.id)}
          <Select.Item value={entry.id} label={entry.name}>
            <ProviderIcon profile={entry.details.compatibilityProfile} size={20} />
            <span class="min-w-0"
              ><span class="block truncate">{entry.name}</span><span
                class="block max-w-80 truncate text-[11px] text-muted-foreground"
                >{entry.details.model || "Model not selected"}</span
              ></span
            >
          </Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
    <div class="mt-3 flex flex-wrap gap-1.5">
      <Button
        variant="secondary"
        size="sm"
        disabled={busy || entries.length >= 32}
        onclick={() => begin(Action.Create)}
        ><PlusIcon data-icon="inline-start" />Save as new</Button
      >
      <Button
        variant="ghost"
        size="sm"
        disabled={busy || dirty || entries.length >= 32}
        onclick={() => begin(Action.Duplicate)}
        ><CopyIcon data-icon="inline-start" />Duplicate</Button
      >
      <Button
        variant="ghost"
        size="sm"
        disabled={busy || dirty}
        onclick={() => begin(Action.Rename)}><PencilIcon data-icon="inline-start" />Rename</Button
      >
      <Button
        variant="ghost"
        size="sm"
        disabled={busy || dirty || entries.length < 2}
        onclick={() => begin(Action.Delete)}><Trash2Icon data-icon="inline-start" />Delete</Button
      >
    </div>
    <p class="mt-2 text-[11px] leading-relaxed text-muted-foreground">
      {dirty
        ? "Save changes to update this connection, or Save as new to keep it separately. Discard changes before switching connections."
        : "Edits below update this connection when saved. Switching applies immediately to new requests."}
    </p>
  </div>
{/if}

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-[440px]">
    <Dialog.Header>
      <Dialog.Title>{title}</Dialog.Title>
      <Dialog.Description>
        {#if action === Action.Delete}
          Delete “{selected?.name}” and activate another saved connection. Requests already running
          keep their captured configuration.
        {:else if action === Action.Create}
          Save the connection values shown in Settings under a new name. Only a newly entered API
          key is attached; an existing saved key is not copied.
        {:else if action === Action.Duplicate}
          Copy “{selected?.name}”, including its saved credential reference. Future edits to either
          connection are independent.
        {:else}
          Choose a name that helps you identify this endpoint.
        {/if}
      </Dialog.Description>
    </Dialog.Header>
    <form onsubmit={submit} class="flex flex-col gap-4">
      {#if action === Action.Delete}
        <div class="space-y-2">
          <label for="replacement-connection" class="text-sm font-medium">Switch to</label>
          <Select.Root type="single" bind:value={replacementID} disabled={busy}>
            <Select.Trigger id="replacement-connection" class="w-full"
              >{replacement?.name ?? "Choose a replacement"}</Select.Trigger
            >
            <Select.Content
              >{#each entries.filter((entry) => entry.id !== selected?.id) as entry (entry.id)}<Select.Item
                  value={entry.id}
                  label={entry.name}>{entry.name}</Select.Item
                >{/each}</Select.Content
            >
          </Select.Root>
        </div>
      {:else}
        <div class="space-y-2">
          <label for="connection-name" class="text-sm font-medium">Connection name</label>
          <Input
            id="connection-name"
            bind:value={name}
            maxlength={80}
            required
            disabled={busy}
            autocomplete="off"
            placeholder="For example, Office speech server"
          />
        </div>
      {/if}
      {#if failed}<p role="alert" class="text-sm text-destructive">
          {error || "The connection could not be changed."}
        </p>{/if}
      <Dialog.Footer>
        <Button type="button" variant="outline" disabled={busy} onclick={() => (open = false)}
          >Cancel</Button
        >
        <Button
          type="submit"
          variant={action === Action.Delete ? "destructive" : "default"}
          disabled={busy || (action === Action.Delete ? !replacementID : !name.trim())}
        >
          {#if busy}<LoaderCircleIcon data-icon="inline-start" class="animate-spin" />{/if}
          {action === Action.Delete
            ? "Delete and switch"
            : action === Action.Create
              ? "Save new connection"
              : action === Action.Duplicate
                ? "Duplicate and use"
                : "Save name"}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
