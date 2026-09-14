<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import {
    ProviderID,
    type ProviderDescriptor,
    type BinaryOptions,
  } from "$bindings/managedruntime";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import RuntimeDownloadSource from "$lib/components/settings/RuntimeDownloadSource.svelte";
  import ModelDownloadSource from "$lib/components/settings/ModelDownloadSource.svelte";

  import {
    backendLabel,
    modelSize,
    runtimePresentation,
  } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";
  import CheckIcon from "@lucide/svelte/icons/check";

  import { Switch } from "$lib/components/ui/switch";
  import * as Dialog from "$lib/components/ui/dialog";

  import DownloadIcon from "@lucide/svelte/icons/download";
  import PlayIcon from "@lucide/svelte/icons/play";
  import SquareIcon from "@lucide/svelte/icons/square";
  import RefreshIcon from "@lucide/svelte/icons/refresh-cw";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";

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
  const models = $derived(
    (status?.models?.length ? status.models : (provider?.models ?? [])).filter(
      (m) => provider?.models?.some((q) => q.id === m.id),
    ),
  );

  const operating = $derived(runtime.isBusy(id));
  const locked = $derived(
    disabled || workBusy || runtime.loading || operating || !status?.supported,
  );
  const running = $derived(status?.state === "running");
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

<div class="@container/runtime flex min-w-0 flex-col gap-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h2 class="text-base font-semibold">Managed runtimes</h2>
      <p class="mt-1 text-[13px] text-secondary-foreground">
        Each installed runtime appears automatically as a built-in Connection.
      </p>
    </div>
    <div class="flex items-center gap-1">
      <Button variant="ghost" size="sm" onclick={onConnections}
        >Manage connections</Button
      >
    </div>
  </div>
  {#if runtime.providers.length}
    <div class="divide-y divide-hairline" aria-label="Runtime inventory">
      {#each runtime.providers as entry (entry.id)}
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
        <div class="py-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="min-w-0 flex-1">
              <h3 class="flex items-center gap-2 text-sm font-semibold">
                <ProviderIcon profile={entry.id} size={24} />{entry.name}
              </h3>
              <p
                class="mt-2 flex items-center gap-2 text-[13px] text-secondary-foreground"
              >
                {#if itemBusy}<LoaderCircleIcon
                    class="size-3 shrink-0 animate-spin text-primary motion-reduce:animate-none"
                  />
                {:else}<span
                    class={`size-1.5 shrink-0 rounded-full ${item?.status.state === "running" ? "bg-success" : item?.status.state === "error" ? "bg-destructive" : "bg-secondary-foreground"}`}
                    aria-hidden="true"
                  ></span>{/if}
                <span
                  class={item?.status.state === "running"
                    ? "font-medium text-success"
                    : item?.status.state === "error"
                      ? "text-destructive"
                      : itemBusy
                        ? "text-primary"
                        : ""}
                >
                  {!entry.supported
                    ? entry.unavailableReason || "Unavailable on this platform"
                    : item
                      ? presentation.startup || presentation.label
                      : "Not installed"}
                </span>
                {#if !expanded && itemModel}
                  <span class="min-w-0 truncate">
                    · {itemModel.name}{switchableProvider(entry.id) &&
                    item?.status.backend
                      ? ` · ${backendLabel(item.status.backend)} binary`
                      : ""}
                  </span>
                {/if}
              </p>
            </div>
            <div class="ml-auto flex items-center gap-2">
              {#if !presentation.installed && !runtime.isBusy(item?.instance.id ?? entry.id)}
                <Button
                  variant="outline"
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
                      variant="ghost"
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
                  variant="ghost"
                  size="sm"
                  onclick={() => void runtime.openOutput(itemID)}
                  >View output</Button
                >
              {/if}
              <Button
                variant="ghost"
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
            <div class="mt-3">
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
              <div class="space-y-2 py-3" role="status">
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
            <div id={`${uid}-${entry.id}-details`} class="mt-5 space-y-5">
              <section class="space-y-3" aria-label="Runtime setup">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <h4 class="text-sm font-semibold">Runtime binary</h4>
                  <span class="text-xs text-secondary-foreground">
                    {status.version || provider?.version || "Not installed"}
                    {#if status.backend && !switchableProvider(entry.id)}
                      · {backendLabel(status.backend)}{/if}
                  </span>
                </div>
                {#if switchableProvider(entry.id) && status.supported && status.backend}
                  <fieldset disabled={locked || running}>
                    <legend class="sr-only">Runtime binary</legend>
                    <div class="flex flex-wrap items-center gap-2">
                      {#each entry.backends ?? [] as backend (backend)}
                        <Button
                          variant="outline"
                          class={status.backend === backend
                            ? "border-primary/50 bg-accent-wash text-primary disabled:opacity-100"
                            : ""}
                          size="sm"
                          aria-pressed={status.backend === backend}
                          disabled={locked ||
                            running ||
                            status.backend === backend}
                          onclick={() => {
                            if (!running)
                              act(() => runtime.installBackend(id, backend));
                          }}>{backendLabel(backend)}</Button
                        >
                      {/each}
                      {#if running}<span
                          class="text-xs text-secondary-foreground"
                          >Stop to change binary.</span
                        >{/if}
                    </div>
                  </fieldset>
                  <details class="text-xs text-secondary-foreground">
                    <summary
                      class="w-fit cursor-pointer rounded-sm py-1 font-medium text-primary focus-visible:outline-ring"
                      >Compatibility &amp; switching</summary
                    >
                    <p class="mt-2 max-w-prose leading-relaxed">
                      Switching keeps downloaded models and saved Connections.
                      Existing installations are never changed automatically.
                      {#if entry.backends?.includes("metal")}
                        Metal uses the Apple Silicon GPU and shares memory with
                        NeMo and other apps. Choose CPU to disable GPU offload.
                      {:else if entry.backends?.includes("cuda")}
                        CUDA 12.4 requires an NVIDIA GPU with compute capability
                        5.0+ and driver 551.78 or newer. It may share GPU memory
                        with NeMo or other apps.
                      {/if}
                    </p>
                  </details>
                {/if}
                {#if operating && !view.startup && (status.operation?.kind !== "download" || status.operation.model === row.instance.model)}<div
                    class="mt-3"
                    role="status"
                  >
                    <p class="text-xs text-secondary-foreground">
                      {view.operationModel || view.activity}{view.percent !==
                      null
                        ? ` · ${view.percent}%`
                        : ""}
                    </p>
                    {#if view.transferred}<p
                        class="mt-1 text-xs tabular-nums text-secondary-foreground"
                      >
                        {view.transferred}
                      </p>{/if}
                    {#if view.percent !== null}
                      <progress
                        class="mt-2 h-1.5 w-full accent-primary"
                        max="100"
                        value={view.percent}
                        aria-label="Runtime operation progress"
                      ></progress>
                    {:else}
                      <LoaderCircleIcon
                        class="mt-2 size-4 animate-spin motion-reduce:animate-none text-secondary-foreground"
                        aria-label={view.activity}
                      />
                    {/if}
                  </div>{/if}
                {#if !operating && view.completion && !(status.operation?.outcome === "succeeded" && ["start", "startup", "stop"].includes(status.operation.kind))}<p
                    class="mt-3 text-sm"
                    role="status"
                  >
                    {view.operationModel
                      ? `${view.operationModel}: `
                      : ""}{view.completion}
                  </p>{/if}
              </section>
              <section
                class="space-y-3 border-t border-hairline pt-5"
                aria-label="Model catalog"
              >
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h4 class="text-sm font-semibold">Models</h4>
                    <p class="mt-1 text-xs text-secondary-foreground">
                      Browsing is metadata-only. Only Download fetches model
                      files.
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
                {#if running}<p class="text-xs text-secondary-foreground">
                    Stop to download or change models.
                  </p>{/if}
                {#each models as model (model.id)}
                  {@const source =
                    entry.models?.find((candidate) => candidate.id === model.id)
                      ?.source ?? model.source}
                  {@const downloading =
                    status.operation?.kind === "download" &&
                    status.operation.model === model.id}
                  <article
                    class="border-b border-hairline py-3 last:border-b-0"
                    aria-label={model.name}
                  >
                    <div
                      class="flex flex-wrap items-start justify-between gap-3"
                    >
                      <div class="min-w-0 flex-[1_1_12rem]">
                        <h5
                          class="text-sm font-semibold"
                          title={model.description}
                        >
                          {model.name}
                        </h5>
                        <p class="mt-1.5 text-[13px] text-secondary-foreground">
                          {model.description}
                        </p>
                        <p class="mt-1 text-xs text-secondary-foreground">
                          {modelSize(model.sizeBytes)}
                          {#if model.installed}<span class="text-success">
                              · Downloaded</span
                            >{/if}
                          {#if model.recommended}
                            · Recommended{/if}
                        </p>
                      </div>
                      <div class="flex flex-wrap gap-1">
                        {#if model.id === row.instance.model}
                          <span
                            class="inline-flex h-8 items-center gap-1.5 px-2 text-xs font-medium text-primary"
                            ><CheckIcon class="size-3.5" />Selected</span
                          >
                        {:else}
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={locked || running}
                            onclick={() =>
                              act(() =>
                                runtime.saveInstance({
                                  ...row.instance,
                                  model: model.id,
                                }),
                              )}>Select model</Button
                          >
                        {/if}
                        {#if downloading && operating}<Button
                            variant="outline"
                            size="sm"
                            disabled={disabled ||
                              runtime.pendingFor(id) === "Cancelling"}
                            onclick={() => void runtime.cancel(id)}
                            >Cancel</Button
                          >{:else if model.installed}<Button
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
                    {#if source}<div class="mt-2">
                        <ModelDownloadSource {source} />
                      </div>{/if}
                    {#if downloading}
                      <div class="mt-3 space-y-1" role="status">
                        {#if operating}
                          <p class="text-xs text-secondary-foreground">
                            {runtime.pendingFor(id) ||
                              view.activity}{view.percent !== null
                              ? ` · ${view.percent}%`
                              : ""}
                          </p>
                          {#if view.transferred}<p
                              class="text-xs tabular-nums text-secondary-foreground"
                            >
                              {view.transferred}
                            </p>{/if}
                          {#if view.percent !== null}
                            <progress
                              class="h-1.5 w-full accent-primary"
                              max="100"
                              value={view.percent}
                              aria-label={`${model.name} download progress`}
                            ></progress>
                          {:else}
                            <LoaderCircleIcon
                              class="size-4 animate-spin motion-reduce:animate-none text-secondary-foreground"
                              aria-label={view.activity}
                            />
                          {/if}
                        {:else if view.completion}
                          <p class="text-sm">{view.completion}</p>
                          {#if status.operation.error}<p
                              class="text-xs text-destructive"
                            >
                              {status.operation.error}
                            </p>{/if}
                        {/if}
                      </div>
                    {/if}
                  </article>
                {/each}
                {#if !models.length}<p
                    class="text-sm text-secondary-foreground"
                  >
                    Install the runtime, then refresh its qualified model
                    catalog.
                  </p>{/if}
              </section>
              <details class="border-t border-hairline pt-4">
                <summary class="cursor-pointer text-sm font-medium"
                  >Runtime preferences</summary
                >
                <div class="mt-4 space-y-4">
                  <div class="space-y-1 text-xs text-secondary-foreground">
                    <p class="break-all">
                      Active API model: {row.activeModel || "None"}
                    </p>
                    {#if row.activeModel && row.activeModel !== row.instance.model}
                      <p>
                        The API model identity differs from the selected catalog
                        key.
                      </p>
                    {/if}
                    <p>
                      Stopping or removing files keeps saved Connections
                      selected; there is no automatic fallback.
                    </p>
                  </div>
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <label
                        for={`${uid}-autostart`}
                        class="text-sm font-medium"
                        >Start when Freehand launches</label
                      >
                      <p class="mt-1 text-xs text-secondary-foreground">
                        Uses this runtime’s selected model. Does not download
                        missing files.
                      </p>
                    </div>
                    <Switch
                      id={`${uid}-autostart`}
                      checked={row.instance.autoStart}
                      disabled={locked}
                      onCheckedChange={(autoStart) =>
                        act(() =>
                          runtime.saveInstance({ ...row.instance, autoStart }),
                        )}
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
                    >{#if providerRows.length > 1}<Button
                        variant="ghost"
                        size="sm"
                        disabled={locked}
                        onclick={() => {
                          confirmation = {
                            instanceID: id,
                            kind: "instance",
                            name: row.instance.name,
                          };
                        }}
                        ><TrashIcon class="size-3.5" />Delete duplicate entry</Button
                      >{/if}
                  </div>
                </div>
              </details>
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
      variant="ghost"
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
