<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import {
    ProviderID,
    type ProviderDescriptor,
    type BinaryOptions,
  } from "$bindings/managedruntime";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import RuntimeDownloadSource from "$lib/components/settings/RuntimeDownloadSource.svelte";
  import ModelDownloadSource from "$lib/components/settings/ModelDownloadSource.svelte";

  import { backendLabel, runtimePresentation } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";

  import * as Dialog from "$lib/components/ui/dialog";

  import DownloadIcon from "@lucide/svelte/icons/download";
  import PlayIcon from "@lucide/svelte/icons/play";
  import SquareIcon from "@lucide/svelte/icons/square";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";

  import LocalRuntimeDetails from "./LocalRuntimeDetails.svelte";

  let {
    runtime,
    disabled = false,
    workBusy = false,
    onAction,
    onConnections,
    focus = "",
    chrome = true,
  }: {
    runtime: ManagedRuntimeState;
    disabled?: boolean;
    workBusy?: boolean;
    onAction: (action: () => void) => void;
    onConnections: () => void;
    /**
     * Render only this provider, expanded. The Runtimes pane puts the
     * inventory in its own sidebar, so the body shows one runtime at a time.
     */
    focus?: string;
    /** False when the surrounding pane already supplies a heading. */
    chrome?: boolean;
  } = $props();
  const uid = $props.id();
  let now = $state(Date.now());
  $effect(() => {
    const timer = setInterval(() => {
      now = Date.now();
    }, 1000);
    return () => clearInterval(timer);
  });
  let recommendation = $state<BinaryOptions | null>(null);
  let probing = $state(false);
  let binaryChoice = $state("auto");
  const chosenBackend = $derived(
    binaryChoice === "auto"
      ? (recommendation?.recommendedBackend ?? "")
      : binaryChoice,
  );
  const choiceAvailable = $derived(
    recommendation?.supported &&
      recommendation.options?.some(
        (o) => o.backend === chosenBackend && o.supported && o.available,
      ),
  );
  let selectedID = $state("");
  // A focused provider owns the selection: the pane's sidebar is the list, so
  // the body must follow it rather than keep a second, private one.
  $effect(() => {
    if (focus) selectedID = focus;
  });
  const shown = $derived(
    focus
      ? runtime.providers.filter((entry) => entry.id === focus)
      : runtime.providers,
  );
  let recoveryID = $state("");
  const provider = $derived(runtime.providers.find((p) => p.id === selectedID));
  const providerRows = $derived(
    runtime.instances.filter((item) => item.instance.provider === provider?.id),
  );
  const row = $derived(
    providerRows.find((item) => item.instance.id === recoveryID) ??
      providerRows[0],
  );
  const id = $derived(row?.instance.id ?? provider?.id ?? "");
  const status = $derived(row?.status);
  const view = $derived(runtimePresentation(status, now));
  const operating = $derived(runtime.isBusy(id));
  const locked = $derived(
    disabled || workBusy || runtime.loading || operating || !status?.supported,
  );
  function switchableProvider(providerID: ProviderID) {
    return (
      providerID === ProviderID.LlamaCPP || providerID === ProviderID.WhisperCPP
    );
  }

  const problem = $derived(
    runtime.errorFor(id) || runtime.error || status?.error || "",
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
  function install(provider: ProviderDescriptor, backend?: string) {
    if (
      disabled ||
      workBusy ||
      runtime.loading ||
      runtime.busy ||
      probing ||
      !provider.supported
    )
      return;
    selectedID = provider.id;
    recoveryID = "";
    if (switchableProvider(provider.id) && !backend) {
      probing = true;
      recommendation = null;
      void runtime
        .binaryOptions(provider.id)
        .then((options) => {
          recommendation = options;
          binaryChoice = "auto";
        })
        .finally(() => {
          probing = false;
        });
      return;
    }
    onAction(() => {
      void (async () => {
        let instance = runtime.instances.find(
          (item) => item.instance.provider === provider.id,
        )?.instance;
        if (!instance) {
          const model =
            provider.models?.find((m) => m.recommended) ?? provider.models?.[0];
          if (!model) return;
          instance = {
            id: provider.id,
            name: provider.name,
            provider: provider.id,
            model: model.id,
            autoStart: false,
          };
          if (!(await runtime.saveInstance(instance))) return;
        }
        recommendation = null;
        if (backend) await runtime.installBackend(instance.id, backend);
        else await runtime.run(instance.id, "Install");
      })();
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

<div class="@container/runtime flex min-w-0 flex-col gap-4">
  {#if chrome}
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-base font-semibold">Managed runtimes</h2>
        <p class="mt-1 text-[13px] text-secondary-foreground">
          Each installed runtime appears automatically as a built-in Connection.
        </p>
      </div>
      <div class="flex items-center gap-1">
        <Button variant="soft" size="sm" onclick={onConnections}
          >Manage connections</Button
        >
      </div>
    </div>
  {/if}
  {#if runtime.providers.length}
    <div class="space-y-3" aria-label="Runtime inventory">
      {#each shown as entry (entry.id)}
        {@const item =
          selectedID === entry.id
            ? row
            : runtime.instances.find(
                (item) => item.instance.provider === entry.id,
              )}
        {@const presentation = runtimePresentation(item?.status, now)}
        {@const itemID = item?.instance.id ?? entry.id}
        {@const itemBusy = runtime.isBusy(itemID)}
        {@const itemLocked =
          disabled ||
          workBusy ||
          runtime.loading ||
          itemBusy ||
          !item?.status.supported}
        {@const expanded = selectedID === entry.id}
        {@const itemModel = item?.status.models?.find(
          (model) => model.id === item.instance.model,
        )}
        <div
          class={`rounded-xl border p-3 ${expanded ? "border-accent-edge bg-card" : "border-hairline bg-card"}`}
        >
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex min-w-0 flex-[1_1_15rem] items-center gap-3">
              <span
                class={`flex size-10 shrink-0 items-center justify-center rounded-lg border ${expanded ? "border-accent-edge bg-accent-wash" : "border-hairline bg-background"}`}
              >
                <ProviderIcon profile={entry.id} size={24} />
              </span>
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                  <h3 class="text-sm font-semibold">{entry.name}</h3>
                  <StatusBadge
                    tone={!entry.supported
                      ? "warning"
                      : item?.status.state === "running"
                        ? "success"
                        : item?.status.state === "error"
                          ? "danger"
                          : itemBusy
                            ? "accent"
                            : "neutral"}
                    dot={!itemBusy}
                  >
                    <span class="inline-flex items-center gap-1.5"
                      >{#if itemBusy}<LoaderCircleIcon
                          class="size-3 animate-spin motion-reduce:animate-none"
                        />{/if}
                      {!entry.supported
                        ? "Unavailable"
                        : item
                          ? presentation.label
                          : "Not installed"}</span
                    >
                  </StatusBadge>
                </div>
                <p
                  class={`mt-1 text-xs text-secondary-foreground ${presentation.startup || !entry.supported ? "break-words" : "truncate"}`}
                  title={!entry.supported
                    ? entry.unavailableReason
                    : presentation.startup || itemModel?.name}
                >
                  {!entry.supported
                    ? entry.unavailableReason || "Unavailable on this platform"
                    : presentation.startup ||
                      (itemModel
                        ? `${itemModel.name}${switchableProvider(entry.id) && item?.status.backend ? ` · ${backendLabel(item.status.backend)} binary` : ""}`
                        : "Install a runtime, then download a model.")}
                </p>
              </div>
            </div>
            <div class="ml-auto flex shrink-0 items-center gap-2">
              {#if !presentation.installed && !runtime.isBusy(item?.instance.id ?? entry.id)}
                <Button
                  variant="soft"
                  size="sm"
                  disabled={disabled ||
                    workBusy ||
                    runtime.loading ||
                    runtime.busy ||
                    probing ||
                    !entry.supported ||
                    !entry.models?.length}
                  onclick={() => install(entry)}
                  ><DownloadIcon class="size-4" />Install</Button
                >
              {/if}
              {#if item}
                {#if itemBusy}
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={disabled ||
                      runtime.pendingFor(itemID) === "Cancelling"}
                    aria-label="Cancel operation"
                    onclick={() => void runtime.cancel(itemID)}>Cancel</Button
                  >
                {:else if presentation.installed}
                  {#if item.status.state === "running"}
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={itemLocked}
                      onclick={() =>
                        onAction(() => {
                          void runtime.run(itemID, "Stop");
                        })}><SquareIcon class="size-3.5" />Stop</Button
                    >
                  {:else if !itemModel?.installed}
                    <Button
                      variant="outline"
                      size="sm"
                      aria-label="Download selected model"
                      disabled={itemLocked || !itemModel}
                      onclick={() =>
                        onAction(() => {
                          void runtime.downloadModel(
                            itemID,
                            item.instance.model,
                          );
                        })}><DownloadIcon class="size-3.5" />Download</Button
                    >
                  {:else}
                    <Button
                      variant="default"
                      size="sm"
                      disabled={itemLocked}
                      onclick={() =>
                        onAction(() => {
                          void runtime.run(itemID, "Start");
                        })}><PlayIcon class="size-3.5" />Start</Button
                    >
                  {/if}
                {/if}
              {/if}
              {#if item && runtime.canRetry(itemID) && !itemBusy}
                <Button
                  variant="outline"
                  size="sm"
                  disabled={itemLocked}
                  onclick={() =>
                    onAction(() => {
                      void runtime.retry(itemID);
                    })}>Retry</Button
                >
              {/if}
              {#if item}
                <Button
                  variant="outline"
                  size="sm"
                  onclick={() => void runtime.openOutput(itemID)}
                  >View output</Button
                >
              {/if}
              <Button
                variant={expanded ? "soft" : "ghost"}
                size="icon-sm"
                aria-label="Manage"
                title={expanded
                  ? "Hide runtime details"
                  : "Show runtime details"}
                aria-expanded={expanded}
                aria-controls={`${uid}-${entry.id}-details`}
                onclick={() => {
                  selectedID = selectedID === entry.id ? "" : entry.id;
                  recoveryID = "";
                }}
                ><ChevronDownIcon
                  class={`size-4 transition-transform motion-reduce:transition-none ${expanded ? "rotate-180" : ""}`}
                /></Button
              >
            </div>
            {#if item && !expanded}
              {#if itemBusy && item.status.operation?.kind === "download"}
                <div class="w-full space-y-1" role="status">
                  <p class="text-xs tabular-nums text-secondary-foreground">
                    {presentation.operationModel}{presentation.transferred
                      ? ` · ${presentation.transferred}`
                      : ""}{presentation.percent !== null
                      ? ` · ${presentation.percent}%`
                      : ""}
                  </p>
                  {#if presentation.percent !== null}<progress
                      class="h-1.5 w-full accent-primary"
                      max="100"
                      value={presentation.percent}
                      aria-label="Runtime download progress"
                    ></progress>{/if}
                </div>
              {:else if !itemBusy && (runtime.errorFor(itemID) || item.status.error || (presentation.completion && !(item.status.operation?.outcome === "succeeded" && ["start", "startup", "stop"].includes(item.status.operation.kind))))}
                <p
                  class="w-full text-xs text-secondary-foreground"
                  role="status"
                >
                  {runtime.errorFor(itemID) ||
                    item.status.error ||
                    presentation.completion}
                </p>
              {/if}
            {/if}
          </div>
          {#if entry.source && (expanded || !item)}
            <div class="mt-3 border-t border-hairline pt-3">
              <RuntimeDownloadSource
                source={entry.source}
                backend={recommendation?.provider === entry.id
                  ? chosenBackend
                  : (item?.status.backend ?? "")}
              />
            </div>
          {/if}
          {#if expanded && !item}
            <section
              id={`${uid}-${entry.id}-details`}
              aria-label="Available models"
              class="mt-4 space-y-3 border-t border-hairline pt-4"
            >
              <h4 class="text-sm font-semibold">Models</h4>
              {#each entry.models ?? [] as model (model.id)}
                <div class="space-y-1">
                  <p class="text-sm font-medium">{model.name}</p>
                  <p class="text-xs text-secondary-foreground">
                    {model.description}
                  </p>
                  {#if model.source}<ModelDownloadSource
                      source={model.source}
                    />{/if}
                </div>
              {/each}
            </section>
          {/if}
          {#if selectedID === entry.id && probing}<p
              role="status"
              class="py-3 text-sm"
            >
              Checking host binary options… No files are downloaded.
            </p>{/if}
          {#if recommendation?.provider === entry.id}
            <section
              aria-label="Runtime binary recommendation"
              class="mt-4 space-y-3 border-t border-hairline py-4"
            >
              <p class="text-sm font-medium">
                Recommended: {backendLabel(recommendation.recommendedBackend)}
              </p>
              <p class="text-xs text-secondary-foreground">
                {recommendation.os} · {recommendation.architecture} · {recommendation.reason}
              </p>
              <p class="text-xs text-secondary-foreground">
                No files downloaded yet. Choose a binary, then explicitly
                download and install it.
              </p>
              <div class="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  class="aria-pressed:border-primary/50 aria-pressed:bg-accent-wash aria-pressed:text-primary"
                  aria-pressed={binaryChoice === "auto"}
                  onclick={() => {
                    binaryChoice = "auto";
                  }}>Auto (recommended)</Button
                >
                {#each recommendation.options ?? [] as option (option.backend)}
                  <Button
                    variant="outline"
                    size="sm"
                    class="aria-pressed:border-primary/50 aria-pressed:bg-accent-wash aria-pressed:text-primary"
                    aria-pressed={binaryChoice === option.backend}
                    disabled={!option.supported || !option.available}
                    title={option.reason}
                    onclick={() => {
                      binaryChoice = option.backend;
                    }}>{backendLabel(option.backend)}</Button
                  >
                {/each}
                <Button
                  disabled={!choiceAvailable ||
                    disabled ||
                    workBusy ||
                    runtime.busy ||
                    runtime.loading}
                  onclick={() => install(entry, chosenBackend)}
                  >Download and install</Button
                >
                <Button
                  variant="ghost"
                  onclick={() => {
                    recommendation = null;
                  }}>Not now</Button
                >
              </div>
            </section>
          {/if}
          {#if row && status && selectedID === entry.id}
            {#if providerRows.length > 1}
              <div
                class="my-3 space-y-2 rounded-lg border border-warning/30 bg-warning/10 p-3"
                role="status"
              >
                <p class="text-sm text-secondary-foreground">
                  Multiple saved installations need review. Keep one; remove
                  unwanted files and reassign their Connections before deleting
                  duplicate entries. Nothing is removed automatically.
                </p>
                <div class="flex flex-wrap gap-2">
                  {#each providerRows as legacy (legacy.instance.id)}
                    <Button
                      variant="ghost"
                      size="sm"
                      aria-pressed={id === legacy.instance.id}
                      onclick={() => {
                        recoveryID = legacy.instance.id;
                      }}
                      >{legacy.instance.name} · {legacy.instance.model}</Button
                    >
                  {/each}
                </div>
              </div>
            {/if}
            <div
              id={`${uid}-${entry.id}-details`}
              class="mt-4 space-y-4 border-t border-hairline pt-4"
            >
              <LocalRuntimeDetails
                {runtime}
                {row}
                {entry}
                {view}
                {uid}
                {locked}
                {operating}
                {disabled}
                switchable={switchableProvider(entry.id)}
                hasDuplicates={providerRows.length > 1}
                {act}
                onRemove={(kind) => {
                  confirmation = {
                    instanceID: id,
                    kind,
                    name: row.instance.name,
                  };
                }}
                onRemoveModel={(model) => {
                  confirmation = {
                    instanceID: id,
                    kind: "model",
                    model: model.id,
                    name: model.name,
                  };
                }}
              />
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {:else}
    <p class="text-sm text-secondary-foreground">
      {runtime.loading
        ? "Reading runtime inventory…"
        : "No runtime adapters available. Use your own server in Connections."}
    </p>
  {/if}
  {#if problem}<p role="alert" class="text-sm text-destructive">
      {problem}
    </p>{/if}
  <div>
    <Button
      variant="outline"
      size="sm"
      disabled={runtime.loading}
      onclick={() => void runtime.load()}>Refresh inventory</Button
    >
  </div>
</div>

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
