<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import {
    catalogGroups,
    modelSize,
    runtimePresentation,
  } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";
  import { Badge } from "$lib/components/ui/badge";
  import { Input } from "$lib/components/ui/input";
  import { Switch } from "$lib/components/ui/switch";
  import * as Dialog from "$lib/components/ui/dialog";
  import * as Select from "$lib/components/ui/select";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import PlayIcon from "@lucide/svelte/icons/play";
  import SquareIcon from "@lucide/svelte/icons/square";
  import RefreshIcon from "@lucide/svelte/icons/refresh-cw";
  import PlusIcon from "@lucide/svelte/icons/plus";
  import TrashIcon from "@lucide/svelte/icons/trash-2";

  let {
    runtime,
    disabled = false,
    workBusy = false,
    onAction,
    onConnections,
  }: {
    runtime: ManagedRuntimeState;
    disabled?: boolean;
    workBusy?: boolean;
    onAction: (action: () => void) => void;
    onConnections: () => void;
  } = $props();
  const uid = $props.id();
  let selectedID = $state("");
  let adding = $state(false);
  let providerID = $state("");
  let modelID = $state("");
  let name = $state("");
  let addID = $state("");
  let rename = $state("");
  let renaming = $state(false);
  const row = $derived(runtime.statusFor(selectedID) ?? runtime.instances[0]);
  const id = $derived(row?.instance.id ?? "");
  const status = $derived(row?.status);
  const provider = $derived(
    runtime.providers.find((p) => p.id === row?.instance.provider),
  );
  const view = $derived(runtimePresentation(status));
  const models = $derived(
    (status?.models ?? []).filter((m) =>
      provider?.models?.some((q) => q.id === m.id),
    ),
  );
  const groups = $derived(catalogGroups(models));
  const selected = $derived(models.find((m) => m.id === row?.instance.model));
  const operating = $derived(runtime.isBusy(id));
  const locked = $derived(
    disabled || workBusy || runtime.loading || operating || !status?.supported,
  );
  const running = $derived(status?.state === "running");
  const problem = $derived(
    runtime.errorFor(id) || runtime.error || status?.error || "",
  );
  const availableProviders = $derived(
    runtime.providers.filter((p) => p.supported),
  );
  const addProvider = $derived(
    availableProviders.find((p) => p.id === providerID) ??
      availableProviders[0],
  );
  const addModel = $derived(
    addProvider?.models?.find((m) => m.id === modelID) ??
      addProvider?.models?.find((m) => m.recommended) ??
      addProvider?.models?.[0],
  );
  type Confirmation = {
    instanceID: string;
    kind: "files" | "instance" | "model";
    model?: string;
    name: string;
  };
  let confirmation = $state<Confirmation | null>(null);
  function act(action: () => Promise<unknown>) {
    if (!locked)
      onAction(() => {
        void action();
      });
  }
  function beginAdd() {
    addID = crypto.randomUUID();
    name = "";
    modelID = "";
    providerID = "";
    adding = true;
  }
  function add() {
    if (
      !addProvider ||
      !addModel ||
      !name.trim() ||
      disabled ||
      workBusy ||
      runtime.busy
    )
      return;
    const instance = {
      id: addID,
      name: name.trim(),
      provider: addProvider.id,
      model: addModel.id,
      autoStart: false,
    };
    onAction(() => {
      void runtime.saveInstance(instance).then((ok) => {
        if (ok) {
          selectedID = instance.id;
          adding = false;
        }
      });
    });
  }
  function confirm() {
    if (!confirmation || locked) return;
    const target = confirmation;
    confirmation = null;
    act(() =>
      target.kind === "files"
        ? runtime.run(target.instanceID, "Remove")
        : target.kind === "instance"
          ? runtime.deleteInstance(target.instanceID)
          : runtime.removeModel(target.instanceID, target.model!),
    );
  }
</script>

<div class="@container/runtime flex min-w-0 flex-col gap-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-base font-semibold">Managed runtimes</h2>
      <p class="mt-1 text-[13px] text-muted-foreground">
        Install and run models here. Choose their tasks in Connections.
      </p>
    </div>
    <div class="flex items-center gap-1">
      <Button variant="ghost" size="sm" onclick={onConnections}
        >Manage connections</Button
      ><Button
        variant="outline"
        size="sm"
        disabled={disabled ||
          workBusy ||
          runtime.loading ||
          runtime.busy ||
          !availableProviders.length}
        onclick={beginAdd}><PlusIcon class="size-4" />Add runtime</Button
      >
    </div>
  </div>
  {#if runtime.instances.length}
    <div class="flex flex-wrap gap-2" aria-label="Runtime inventory">
      {#each runtime.instances as item (item.instance.id)}
        <Button
          variant={id === item.instance.id ? "secondary" : "outline"}
          size="sm"
          aria-pressed={id === item.instance.id}
          onclick={() => {
            selectedID = item.instance.id;
            renaming = false;
          }}
        >
          {item.instance.name}<span class="text-xs text-muted-foreground"
            >{runtimePresentation(item.status).label}</span
          >
        </Button>
      {/each}
    </div>
  {:else}
    <div class="rounded-xl border border-dashed border-hairline p-5">
      <h3 class="text-sm font-medium">
        {runtime.loading
          ? "Reading runtime inventory…"
          : "No managed runtimes yet"}
      </h3>
      <p class="mt-2 text-[13px] text-muted-foreground">
        Add an available provider and choose a qualified model. Nothing
        downloads or starts until you ask.
      </p>
      {#if !runtime.loading && !availableProviders.length}<p
          class="mt-2 text-xs text-muted-foreground"
        >
          No managed provider is available on this platform. You can use your
          own server in Connections.
        </p>{/if}
    </div>
  {/if}
  {#if row && status}
    <section
      class="overflow-hidden rounded-xl border border-hairline bg-card p-5"
      aria-label="Runtime setup"
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 class="font-semibold">{row.instance.name}</h3>
          <p class="mt-1 text-xs text-muted-foreground">
            {provider?.name ?? row.instance.provider} · {status.version ||
              provider?.version ||
              "Not installed"}
          </p>
        </div>
        <Badge variant="secondary">{view.label}</Badge>
      </div>
      <p class="mt-4 text-[13px] text-muted-foreground">
        {operating
          ? runtime.pendingFor(id) || "Runtime operation in progress…"
          : !view.installed
            ? "First, install the runtime on this PC."
            : !selected?.installed
              ? `Next, download ${selected?.name ?? row.instance.model}.`
              : running
                ? "Process ready. Assign this runtime through Connections to use it for a task."
                : "Your selected model is downloaded. Start the runtime when you need it."}
      </p>
      <div class="mt-3 flex flex-wrap items-center gap-2">
        {#if operating}<Button
            variant="outline"
            disabled={disabled || runtime.pendingFor(id) === "Cancelling"}
            onclick={() => void runtime.cancel(id)}>Cancel operation</Button
          >
        {:else if running}<Button
            variant="outline"
            disabled={locked}
            onclick={() => act(() => runtime.run(id, "Stop"))}
            ><SquareIcon class="size-4" />Stop runtime</Button
          >
        {:else if !view.installed}<Button
            disabled={locked}
            onclick={() => act(() => runtime.run(id, "Install"))}
            ><DownloadIcon class="size-4" />Install runtime</Button
          >
        {:else if !selected?.installed}<Button
            disabled={locked || !selected}
            onclick={() =>
              act(() => runtime.downloadModel(id, row.instance.model))}
            ><DownloadIcon class="size-4" />Download selected model</Button
          >
        {:else}<Button
            disabled={locked}
            onclick={() => act(() => runtime.run(id, "Start"))}
            ><PlayIcon class="size-4" />Start runtime</Button
          >{/if}
        {#if runtime.canRetry(id) && !operating}<Button
            variant="outline"
            disabled={locked}
            onclick={() => act(() => runtime.retry(id))}>Retry</Button
          >{/if}
        <Button variant="ghost" size="sm" onclick={onConnections}
          >Connect to a task</Button
        >
      </div>
      {#if operating}<div class="mt-3" role="status">
          <p class="text-xs text-muted-foreground">
            {status.phase || runtime.pendingFor(id)}{view.percent !== null
              ? ` · ${view.percent}%`
              : ""}
          </p>
          <progress
            class="mt-2 h-1.5 w-full accent-primary"
            max="100"
            value={view.percent ?? undefined}
            aria-label="Runtime operation progress"
          ></progress>
        </div>{/if}
      <div class="mt-4 space-y-1 text-xs text-muted-foreground">
        <p>Selected: {selected?.name ?? row.instance.model}</p>
        <p class="break-all">Active API model: {row.activeModel || "None"}</p>
        {#if row.activeModel && row.activeModel !== row.instance.model}<p>
            The API model identity differs from the selected catalog key.
          </p>{/if}
        <p>
          Stopping or removing files keeps saved Connections selected; there is
          no automatic fallback.
        </p>
      </div>
    </section>
    <details class="rounded-xl border border-hairline bg-card p-4">
      <summary class="cursor-pointer text-sm font-medium"
        >Instance preferences</summary
      >
      <div class="mt-4 space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-sm">{row.instance.name}</span><Button
            variant="ghost"
            size="sm"
            disabled={locked}
            onclick={() => {
              rename = row.instance.name;
              renaming = true;
            }}>Rename</Button
          >
        </div>
        {#if renaming}<div class="flex gap-2">
            <Input
              aria-label="Runtime name"
              bind:value={rename}
              maxlength={80}
            /><Button
              size="sm"
              disabled={locked || !rename.trim()}
              onclick={() =>
                act(async () => {
                  if (
                    await runtime.saveInstance({
                      ...row.instance,
                      name: rename.trim(),
                    })
                  )
                    renaming = false;
                })}>Save name</Button
            >
          </div>{/if}
        <div class="flex items-center justify-between gap-3">
          <div>
            <label for={`${uid}-autostart`} class="text-sm font-medium"
              >Start when Freehand launches</label
            >
            <p class="mt-1 text-xs text-muted-foreground">
              Uses this instance’s selected model. Does not download missing
              files.
            </p>
          </div>
          <Switch
            id={`${uid}-autostart`}
            checked={row.instance.autoStart}
            disabled={locked}
            onCheckedChange={(autoStart) =>
              act(() => runtime.saveInstance({ ...row.instance, autoStart }))}
          />
        </div>
        <div class="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={locked || !view.installed}
            onclick={() => {
              confirmation = {
                instanceID: id,
                kind: "files",
                name: row.instance.name,
              };
            }}>Remove runtime files</Button
          ><Button
            variant="ghost"
            size="sm"
            disabled={locked}
            onclick={() => {
              confirmation = {
                instanceID: id,
                kind: "instance",
                name: row.instance.name,
              };
            }}><TrashIcon class="size-3.5" />Delete instance</Button
          >
        </div>
      </div>
    </details>
    <section class="space-y-3" aria-label="Model catalog">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-sm font-semibold">Model catalog</h3>
          <p class="mt-1 text-xs text-muted-foreground">
            Browsing is metadata-only. Only Download fetches model files.
          </p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          disabled={locked || !view.installed}
          onclick={() => act(() => runtime.run(id, "RefreshCatalog"))}
          ><RefreshIcon class="size-3.5" />Refresh catalog</Button
        >
      </div>
      {#if running}<p class="text-xs text-muted-foreground">
          Stop the runtime before downloading or changing models.
        </p>{/if}
      {#each groups as group (group.label)}{#if group.models.length}<div
            class="space-y-2"
          >
            <h4 class="text-xs font-medium text-muted-foreground">
              {group.label}
            </h4>
            {#each group.models as model (model.id)}
              <article
                class="rounded-xl border border-hairline bg-card p-4"
                aria-label={model.name}
              >
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-0 flex-[1_1_12rem]">
                    <div class="flex flex-wrap items-center gap-2">
                      <h5 class="text-sm font-semibold">{model.name}</h5>
                      {#if model.recommended}<Badge variant="secondary"
                          >Recommended</Badge
                        >{/if}{#if model.id === row.instance.model}<Badge
                          variant="outline">Selected</Badge
                        >{/if}
                    </div>
                    <p class="mt-1.5 text-[13px] text-muted-foreground">
                      {model.description}
                    </p>
                    <p class="mt-2 text-xs text-muted-foreground">
                      {modelSize(model.sizeBytes)} · {model.realtime
                        ? "Realtime + completed speech"
                        : "Completed speech"}
                    </p>
                  </div>
                  <div class="flex flex-wrap gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={locked ||
                        running ||
                        model.id === row.instance.model}
                      onclick={() =>
                        act(() =>
                          runtime.saveInstance({
                            ...row.instance,
                            model: model.id,
                          }),
                        )}
                      >{model.id === row.instance.model
                        ? "Selected"
                        : "Select model"}</Button
                    >
                    {#if model.installed}<Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Delete ${model.name}`}
                        title="Delete model"
                        disabled={locked || running}
                        onclick={() => {
                          confirmation = {
                            instanceID: id,
                            kind: "model",
                            model: model.id,
                            name: model.name,
                          };
                        }}><TrashIcon class="size-3.5" /></Button
                      >{:else}<Button
                        variant="outline"
                        size="sm"
                        disabled={locked || running || !view.installed}
                        onclick={() =>
                          act(() => runtime.downloadModel(id, model.id))}
                        ><DownloadIcon class="size-3.5" />Download</Button
                      >{/if}
                  </div>
                </div>
              </article>
            {/each}
          </div>{/if}{/each}
      {#if !models.length}<p class="text-sm text-muted-foreground">
          Install the runtime, then refresh its qualified model catalog.
        </p>{/if}
    </section>
  {/if}
  {#if problem}<p role="alert" class="text-sm text-destructive">
      {problem}
    </p>{/if}
  <div>
    <Button
      variant="ghost"
      size="sm"
      disabled={runtime.loading}
      onclick={() => void runtime.load()}>Refresh inventory</Button
    >
  </div>
</div>

<Dialog.Root bind:open={adding}
  ><Dialog.Content
    ><Dialog.Header
      ><Dialog.Title>Add runtime</Dialog.Title><Dialog.Description
        >Choose an available provider and a qualified model. Installation,
        download, and Start remain separate explicit steps.</Dialog.Description
      ></Dialog.Header
    >
    <div class="space-y-4">
      <div class="space-y-1.5">
        <label for={`${uid}-provider`} class="text-sm">Provider</label
        ><Select.Root
          type="single"
          value={addProvider?.id ?? ""}
          onValueChange={(value) => {
            providerID = value;
            modelID = "";
          }}
          ><Select.Trigger id={`${uid}-provider`} class="w-full"
            >{addProvider?.name ?? "No available providers"}</Select.Trigger
          ><Select.Content
            >{#each availableProviders as p (p.id)}<Select.Item
                value={p.id}
                label={p.name}>{p.name}</Select.Item
              >{/each}</Select.Content
          ></Select.Root
        >
      </div>
      <div class="space-y-1.5">
        <label for={`${uid}-new-name`} class="text-sm">Runtime name</label
        ><Input
          id={`${uid}-new-name`}
          bind:value={name}
          maxlength={80}
          placeholder={addProvider?.name ?? "Runtime name"}
        />
      </div>
      <div class="space-y-1.5">
        <label for={`${uid}-new-model`} class="text-sm">Model</label
        ><Select.Root
          type="single"
          value={addModel?.id ?? ""}
          onValueChange={(value) => {
            modelID = value;
          }}
          ><Select.Trigger id={`${uid}-new-model`} class="w-full"
            >{addModel?.name ?? "Choose a model"}</Select.Trigger
          ><Select.Content
            >{#each addProvider?.models ?? [] as m (m.id)}<Select.Item
                value={m.id}
                label={m.name}
                >{m.name}{m.recommended ? " · Recommended" : ""}</Select.Item
              >{/each}</Select.Content
          ></Select.Root
        >
        <p class="text-xs text-muted-foreground">{addModel?.description}</p>
      </div>
      {#if runtime.errorFor(addID)}<p
          role="alert"
          class="text-sm text-destructive"
        >
          {runtime.errorFor(addID)}
        </p>{/if}
    </div>
    <Dialog.Footer
      ><Button
        variant="outline"
        onclick={() => {
          adding = false;
        }}>Cancel</Button
      ><Button
        disabled={disabled ||
          workBusy ||
          runtime.busy ||
          !name.trim() ||
          !addModel}
        onclick={add}>Add runtime</Button
      ></Dialog.Footer
    ></Dialog.Content
  ></Dialog.Root
>

<Dialog.Root
  open={confirmation !== null}
  onOpenChange={(open) => {
    if (!open) confirmation = null;
  }}
  ><Dialog.Content showCloseButton={false}
    ><Dialog.Header
      ><Dialog.Title
        >{confirmation?.kind === "files"
          ? "Remove runtime files"
          : confirmation?.kind === "instance"
            ? "Delete runtime instance"
            : "Delete model"}: {confirmation?.name}?</Dialog.Title
      ><Dialog.Description
        >{confirmation?.kind === "files"
          ? "Stops this instance and removes its runtime and downloaded models. The instance and saved Connections remain for repair. Other instances are untouched."
          : confirmation?.kind === "instance"
            ? "Removes this instance from the inventory. First remove or reassign every Connection referencing it. This does not delete downloaded files; remove runtime files separately if wanted."
            : "Deletes this downloaded model from this instance. If selected, this connection becomes unavailable until the model is downloaded again or explicitly changed."}</Dialog.Description
      ></Dialog.Header
    ><Dialog.Footer
      ><Button
        variant="outline"
        onclick={() => {
          confirmation = null;
        }}>Keep</Button
      ><Button variant="destructive" disabled={locked} onclick={confirm}
        >Confirm removal</Button
      ></Dialog.Footer
    ></Dialog.Content
  ></Dialog.Root
>
