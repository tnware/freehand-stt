<script lang="ts">
  import { onDestroy } from "svelte";
  import { Role } from "$bindings/compatibility";
  import {
    ProviderID,
    type BinaryOptions,
    type InstanceStatus,
    type Model,
    type ProviderDescriptor,
  } from "$bindings/managedruntime";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import * as Dialog from "$lib/components/ui/dialog";
  import RuntimeDownloadSource from "$lib/components/settings/RuntimeDownloadSource.svelte";
  import ModelDownloadSource from "$lib/components/settings/ModelDownloadSource.svelte";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import RuntimeOutputDrawer from "./RuntimeOutputDrawer.svelte";
  import PanelTabs from "$lib/components/shell/PanelTabs.svelte";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
  import {
    backendLabel,
    modelSize,
    runtimePresentation,
  } from "$lib/utils/managedRuntime";
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";

  let {
    runtime,
    row,
    entry,
    locked = false,
    workBusy = false,
    onOpenConnections,
  }: {
    runtime: ManagedRuntimeState;
    row: InstanceStatus | undefined;
    entry: ProviderDescriptor;
    locked?: boolean;
    workBusy?: boolean;
    onOpenConnections: () => void;
  } = $props();
  const uid = $props.id();
  const layout = getWorkbenchLayout();
  const status = $derived(row?.status);
  const instance = $derived(row?.instance);
  let now = $state(Date.now());
  $effect(() => {
    if (status?.state !== "starting") return;
    const timer = setInterval(() => (now = Date.now()), 1000);
    return () => clearInterval(timer);
  });
  const view = $derived(runtimePresentation(status, now));
  const running = $derived(status?.state === "running");
  // Keep the catalog visible during acquisition or binary switching.
  const installed = $derived(
    view.installed || (!!status?.backend && status.state !== "not_installed"),
  );
  const busy = $derived(!!instance && runtime.isBusy(instance.id));
  const actionLocked = $derived(
    locked ||
      workBusy ||
      busy ||
      !entry.supported ||
      status?.supported === false,
  );
  const models = $derived(
    (status?.models?.length ? status.models : (entry.models ?? [])).filter(
      (model) => entry.models?.some((qualified) => qualified.id === model.id),
    ),
  );
  const selectedModel = $derived(
    models.find((model) => model.id === instance?.model),
  );
  const problem = $derived(
    runtime.errorFor(instance?.id ?? entry.id) || status?.error || "",
  );
  let preferencesOpen = $state(false);
  let sourceOpen = $state(false);
  let manageOpen = $state(false);
  let panelTab = $state("output");
  let panelCollapsed = $state(false);
  let probing = $state(false);
  let recommendation = $state<BinaryOptions | null>(null);
  let binaryChoice = $state("auto");
  type Removal =
    { kind: "files" } | { kind: "instance" } | { kind: "model"; model: Model };
  let confirming = $state<Removal | null>(null);
  let alive = true;
  onDestroy(() => {
    alive = false;
  });
  const chosenBackend = $derived(
    binaryChoice === "auto"
      ? (recommendation?.recommendedBackend ?? "")
      : binaryChoice,
  );
  const choiceAvailable = $derived(
    !!recommendation?.supported &&
      !!recommendation.options?.some(
        (option) =>
          option.backend === chosenBackend &&
          option.supported &&
          option.available,
      ),
  );
  const switchable = $derived(
    entry.id === ProviderID.LlamaCPP || entry.id === ProviderID.WhisperCPP,
  );
  const sourceBackend = $derived(
    recommendation ? chosenBackend : (status?.backend ?? ""),
  );

  function install(backend?: string) {
    if (actionLocked || runtime.busy || probing || !entry.models?.length)
      return;
    if (switchable && !backend) {
      probing = true;
      recommendation = null;
      void runtime
        .binaryOptions(entry.id)
        .then((options) => {
          if (alive) {
            recommendation = options;
            binaryChoice = "auto";
          }
        })
        .finally(() => {
          if (alive) probing = false;
        });
      return;
    }
    void (async () => {
      let target = instance;
      if (!target) {
        const model =
          entry.models?.find((item) => item.recommended) ?? entry.models?.[0];
        if (!model) return;
        target = {
          id: entry.id,
          name: entry.name,
          provider: entry.id,
          model: model.id,
          autoStart: false,
        };
        if (!(await runtime.saveInstance(target))) return;
      }
      if (alive) recommendation = null;
      if (backend) await runtime.installBackend(target.id, backend);
      else await runtime.run(target.id, "Install");
    })();
  }
  function act(action: () => Promise<unknown>) {
    if (!actionLocked) void action();
  }
  function confirmRemoval() {
    if (!confirming || !instance || actionLocked || running) return;
    const target = confirming,
      id = instance.id;
    confirming = null;
    act(() =>
      target.kind === "files"
        ? runtime.run(id, "Remove")
        : target.kind === "instance"
          ? runtime.deleteInstance(id)
          : runtime.removeModel(id, target.model.id),
    );
  }
  function restart() {
    if (!instance || actionLocked || !running) return;
    const id = instance.id;
    act(async () => {
      if (await runtime.run(id, "Stop")) await runtime.run(id, "Start");
    });
  }
  const task = (model: Model): string =>
    model.contracts?.some((contract) => contract.role === Role.PostProcessing)
      ? "Cleanup"
      : model.realtime
        ? "Streaming"
        : "Batch";
</script>

<div class="flex min-h-0 min-w-0 flex-1 flex-col">
  <div
    class="flex min-h-11 shrink-0 flex-wrap items-center justify-between gap-2 border-b border-hairline px-5 py-1.5"
  >
    <div class="flex min-w-0 items-center gap-2.5">
      <h3
        class="truncate font-display text-[15px] font-semibold tracking-tight"
      >
        {entry.name}
      </h3>
      <StatusBadge
        tone={running
          ? "success"
          : status?.state === "error"
            ? "danger"
            : busy
              ? "accent"
              : "neutral"}
        dot
        >{status
          ? view.label
          : entry.supported
            ? "Not installed"
            : "Unavailable"}</StatusBadge
      >
    </div>
    <div class="flex shrink-0 items-center gap-1.5">
      {#if busy && instance}
        <Button
          variant="outline"
          size="xs"
          aria-label="Cancel operation"
          disabled={runtime.pendingFor(instance.id) === "Cancelling"}
          onclick={() => void runtime.cancel(instance.id)}>Cancel</Button
        >
      {:else if installed && instance}
        {#if running}<Button
            variant="outline"
            size="xs"
            disabled={actionLocked}
            onclick={restart}>Restart</Button
          >{/if}
        {#if !running && !selectedModel?.installed}
          <Button
            variant="outline"
            size="xs"
            aria-label="Download selected model"
            disabled={actionLocked || !selectedModel}
            onclick={() =>
              act(() => runtime.downloadModel(instance.id, instance.model))}
            ><DownloadIcon class="size-3" />Get model</Button
          >
        {:else}
          <Button
            variant="outline"
            size="xs"
            disabled={actionLocked}
            onclick={() =>
              act(() => runtime.run(instance.id, running ? "Stop" : "Start"))}
            >{running ? "Stop" : "Start"}</Button
          >
        {/if}
      {/if}
    </div>
  </div>
  <div
    class="flex min-h-[38px] shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b border-hairline px-5 py-2"
  >
    {#each [status?.version || entry.version, status?.backend ? backendLabel(status.backend) : "", status?.selectedModel].filter(Boolean) as fact, index (index)}
      {#if index > 0}<span
          class="h-3 w-px shrink-0 bg-border"
          aria-hidden="true"
        ></span>{/if}
      <span class="min-w-0 truncate font-mono text-[11px] text-ink-quiet"
        >{fact}</span
      >
    {/each}
    <span class="flex-1"></span>
    {#if layout && instance}
      <Button
        variant="ghost"
        size="xs"
        disabled={!layout.bottomAvailable.current}
        title={!layout.bottomAvailable.current
          ? "Increase window height to show the bottom panel"
          : undefined}
        onclick={() => layout.showOutput(instance.id)}>View output</Button
      >
    {/if}
    <button
      type="button"
      class="shrink-0 text-xs text-ink-quiet underline-offset-2 hover:text-accent-text hover:underline"
      onclick={onOpenConnections}>Open Connections</button
    >
  </div>
  <div class="min-h-0 flex-1 overflow-y-auto px-5">
    {#if busy || view.completion || problem}
      <div
        class="space-y-1.5 border-b border-hairline py-3"
        role="status"
        aria-label="Runtime operation"
      >
        {#if busy}
          <p class="text-[12px] text-secondary-foreground">
            {view.startup || view.activity}{view.operationModel
              ? ` · ${view.operationModel}`
              : ""}{view.transferred ? ` · ${view.transferred}` : ""}
          </p>
          {#if view.percent !== null}<progress
              class="h-1.5 w-full accent-primary"
              max="100"
              value={view.percent}
              aria-label="Runtime operation progress"
            ></progress>{/if}
        {:else if view.completion}<p
            class="text-[12px] text-secondary-foreground"
          >
            {view.operationModel
              ? `${view.operationModel}: `
              : ""}{view.completion}
          </p>{/if}
        {#if problem}<p class="text-[12px] text-destructive" role="alert">
            {problem}
          </p>{/if}
        {#if instance && runtime.canRetry(instance.id) && !busy}<Button
            variant="outline"
            size="xs"
            disabled={actionLocked}
            onclick={() => act(() => runtime.retry(instance.id))}>Retry</Button
          >{/if}
      </div>
    {/if}
    {#if !installed}
      <div class="flex flex-col gap-3 py-3">
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="text-[13px] font-medium">
              {entry.supported
                ? busy
                  ? "Installation in progress"
                  : "Not installed"
                : "Unavailable on this machine"}
            </p>
            <p class="mt-0.5 text-xs text-ink-quiet">
              {entry.unavailableReason ||
                "Install the runtime, then explicitly download a model."}
            </p>
          </div>
          <Button
            size="xs"
            disabled={actionLocked ||
              probing ||
              runtime.busy ||
              !entry.models?.length}
            onclick={() => install()}
            ><DownloadIcon class="size-3" />{probing
              ? "Checking…"
              : recommendation
                ? "Re-check"
                : "Install"}</Button
          >
        </div>
        {#if probing}<p class="text-xs text-ink-quiet" role="status">
            Checking host binary options · no files are downloaded
          </p>{/if}
        {#if recommendation}
          <section
            class="border-y border-hairline py-3"
            aria-label="Runtime binary recommendation"
          >
            <p class="text-[13px] font-medium">
              Recommended: {backendLabel(recommendation.recommendedBackend)}
            </p>
            <p class="mt-0.5 font-mono text-[11px] text-ink-quiet">
              {recommendation.os} · {recommendation.architecture} · {recommendation.reason}
            </p>
            <div class="mt-2.5 flex flex-wrap gap-1.5">
              <Button
                variant="outline"
                size="xs"
                aria-pressed={binaryChoice === "auto"}
                disabled={actionLocked}
                onclick={() => (binaryChoice = "auto")}
                >Auto (recommended)</Button
              >
              {#each recommendation.options ?? [] as option (option.backend)}<Button
                  variant="outline"
                  size="xs"
                  aria-pressed={binaryChoice === option.backend}
                  disabled={actionLocked ||
                    !option.supported ||
                    !option.available}
                  title={option.reason}
                  onclick={() => (binaryChoice = option.backend)}
                  >{backendLabel(option.backend)}</Button
                >{/each}
            </div>
            <div class="mt-3 flex flex-wrap items-center justify-between gap-2">
              <p class="text-xs text-ink-quiet">
                Downloads on install · checksum pinned
              </p>
              <Button
                size="xs"
                disabled={actionLocked || runtime.busy || !choiceAvailable}
                onclick={() => install(chosenBackend)}
                >Download and install</Button
              ><Button
                variant="ghost"
                size="xs"
                onclick={() => (recommendation = null)}>Not now</Button
              >
            </div>
          </section>
        {/if}
        <section aria-label="Available models">
          <p
            class="mb-1 text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
          >
            Models it can run
          </p>
          {#each entry.models ?? [] as model (model.id)}
            <article
              class="border-b border-hairline py-2"
              aria-label={model.name}
            >
              <div class="flex items-center justify-between gap-3">
                <span class="min-w-0"
                  ><span class="block truncate text-[13px]">{model.name}</span
                  ><span
                    class="block truncate font-mono text-[11px] text-ink-quiet"
                    >{model.id}</span
                  ></span
                ><span class="shrink-0 text-[10px] text-ink-quiet"
                  >{model.recommended ? "Recommended" : task(model)}</span
                >
              </div>
              {#if model.source}<div class="mt-2">
                  <ModelDownloadSource
                    source={model.source}
                    description={model.description}
                  />
                </div>{/if}
            </article>
          {/each}
        </section>
      </div>
    {:else}
      <div
        class="flex min-h-10 flex-wrap items-center justify-between gap-2 py-2"
      >
        <span
          class="text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
          >Models</span
        ><Button
          variant="outline"
          size="xs"
          disabled={actionLocked}
          onclick={() => act(() => runtime.run(instance!.id, "RefreshCatalog"))}
          ><RefreshCwIcon class="size-3" />Refresh catalog</Button
        >
      </div>
      <p class="pb-2 text-xs text-ink-quiet">
        {running
          ? "Stop to download, change or remove models."
          : "Browsing is metadata-only. Only Get fetches model files."}
      </p>
      <div class="overflow-x-auto" role="region" aria-label="Model catalog">
        <div class="min-w-[520px]">
          <div
            class="flex h-[26px] items-center border-b border-hairline text-[10px] font-semibold tracking-[0.07em] text-ink-quiet uppercase"
          >
            <span class="min-w-0 flex-1">Model</span><span
              class="w-[70px] shrink-0">Task</span
            ><span class="w-[70px] shrink-0 text-right">Size</span><span
              class="w-[90px] shrink-0 pl-3">State</span
            ><span class="w-[120px] shrink-0 text-right">Action</span>
          </div>
          {#each models as model (model.id)}
            {@const selected = model.id === instance?.model}
            {@const loaded = selected && running}
            {@const modelSource =
              entry.models?.find((qualified) => qualified.id === model.id)
                ?.source ?? model.source}
            {@const downloading =
              status?.operation?.kind === "download" &&
              status.operation.model === model.id}
            <article aria-label={model.name}>
              <div
                class="flex min-h-12 items-center border-b border-hairline {selected
                  ? 'bg-accent-wash'
                  : ''}"
              >
                <span class="min-w-0 flex-1 pl-2"
                  ><span class="block truncate text-[13px] text-foreground"
                    >{model.name}</span
                  ><span
                    class="block truncate font-mono text-[11px] text-ink-quiet"
                    >{model.id}</span
                  ></span
                >
                <span
                  class="w-[70px] shrink-0 text-xs text-secondary-foreground"
                  >{task(model)}</span
                ><span
                  class="w-[70px] shrink-0 text-right font-mono text-[11px]"
                  title={modelSize(model.sizeBytes)}
                  >{model.sizeBytes ? modelSize(model.sizeBytes) : "—"}</span
                >
                <span
                  class="w-[90px] shrink-0 pl-3 text-xs {loaded
                    ? 'text-success'
                    : 'text-ink-quiet'}"
                  >{loaded
                    ? "Loaded"
                    : selected
                      ? "Selected"
                      : model.installed
                        ? "Downloaded"
                        : "Not installed"}</span
                >
                <span
                  class="flex w-[120px] shrink-0 items-center justify-end gap-1"
                >
                  {#if downloading && busy}<Button
                      variant="outline"
                      size="xs"
                      disabled={runtime.pendingFor(instance!.id) ===
                        "Cancelling"}
                      onclick={() => void runtime.cancel(instance!.id)}
                      >Cancel</Button
                    >
                  {:else}
                    {#if !model.installed}<Button
                        variant="outline"
                        size="xs"
                        disabled={actionLocked || running}
                        onclick={() =>
                          act(() =>
                            runtime.downloadModel(instance!.id, model.id),
                          )}><DownloadIcon class="size-3" />Get</Button
                      >{/if}
                    {#if !selected}<Button
                        variant="outline"
                        size="xs"
                        disabled={actionLocked || running}
                        onclick={() =>
                          act(() =>
                            runtime.saveInstance({
                              ...instance!,
                              model: model.id,
                            }),
                          )}>Select</Button
                      >{/if}
                    {#if model.installed}<Button
                        variant="ghost"
                        size="xs"
                        class="size-6 p-0"
                        disabled={actionLocked || running}
                        aria-label={`Delete ${model.name}`}
                        title="Remove downloaded files"
                        onclick={() => (confirming = { kind: "model", model })}
                        ><Trash2Icon class="size-3.5" /></Button
                      >{/if}
                  {/if}
                </span>
              </div>
              {#if downloading}<div
                  class="space-y-1 px-2 py-2 text-xs text-secondary-foreground"
                  role="status"
                >
                  {#if busy}{#if view.percent !== null}<progress
                        class="h-1.5 w-full accent-primary"
                        max="100"
                        value={view.percent}
                        aria-label={`${model.name} download progress`}
                      ></progress>{/if}
                    <p>
                      {view.activity}{view.transferred
                        ? ` · ${view.transferred}`
                        : ""}
                    </p>{:else if view.completion}<p>{view.completion}</p>{/if}
                </div>{/if}
              <details
                class="border-b border-hairline px-2 py-1 text-xs text-ink-quiet"
              >
                <summary
                  class="w-fit cursor-pointer py-1 focus-visible:outline-ring"
                  >Model source and details</summary
                >
                <div class="py-2">
                  {#if modelSource}<ModelDownloadSource
                      source={modelSource}
                      description={model.description}
                    />{:else}<p>{model.description}</p>{/if}
                </div>
              </details>
            </article>
          {:else}<p class="py-6 text-center text-[13px] text-muted-foreground">
              No qualified models in this runtime's catalog yet.
            </p>{/each}
        </div>
      </div>
    {/if}
    {#if instance}
      <button
        type="button"
        class="flex min-h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
        aria-expanded={preferencesOpen}
        onclick={() => (preferencesOpen = !preferencesOpen)}
        ><span class="flex items-center gap-2"
          ><ChevronRightIcon
            class="size-3.5 text-muted-foreground {preferencesOpen
              ? 'rotate-90'
              : ''}"
          /><span class="text-[13px] text-secondary-foreground"
            >Runtime preferences</span
          ></span
        ><span class="truncate font-mono text-[11px] text-ink-quiet"
          >{instance.autoStart ? "starts with Freehand" : "manual start"}</span
        ></button
      >
      {#if preferencesOpen}
        <div class="space-y-3 border-b border-hairline py-3">
          <div class="flex items-center justify-between gap-3">
            <div>
              <label for={`${uid}-autostart`} class="text-[13px]"
                >Start when Freehand launches</label
              >
              <p class="mt-1 text-xs text-ink-quiet">
                Uses the selected model. Does not download missing files.
              </p>
            </div>
            <Switch
              id={`${uid}-autostart`}
              checked={instance.autoStart}
              disabled={actionLocked}
              onCheckedChange={(autoStart) =>
                act(() => runtime.saveInstance({ ...instance, autoStart }))}
            />
          </div>
          <p class="break-all font-mono text-[11px] text-ink-quiet">
            Active API model: {row?.activeModel || "None"}
          </p>
          {#if switchable && installed}<fieldset
              disabled={actionLocked || running}
            >
              <legend class="mb-2 text-[13px]">Runtime binary</legend>
              <div class="flex flex-wrap gap-1.5">
                {#each entry.backends ?? [] as backend (backend)}<Button
                    variant="outline"
                    size="xs"
                    aria-pressed={status?.backend === backend}
                    disabled={actionLocked ||
                      running ||
                      status?.backend === backend}
                    onclick={() => {
                      if (!running)
                        act(() => runtime.installBackend(instance.id, backend));
                    }}>{backendLabel(backend)}</Button
                  >{/each}
              </div>
            </fieldset>
            <p class="text-xs text-ink-quiet">
              Stop to change binary. Switching downloads the selected binary and
              keeps models and saved Connections.
            </p>{/if}
          <p class="text-xs text-ink-quiet">
            Stopping or removing files keeps saved Connections selected. There
            is no automatic fallback.
          </p>
        </div>
      {/if}
    {/if}
    {#if entry.source}
      <button
        type="button"
        class="flex min-h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
        aria-expanded={sourceOpen}
        onclick={() => (sourceOpen = !sourceOpen)}
        ><span class="flex items-center gap-2"
          ><ChevronRightIcon
            class="size-3.5 text-muted-foreground {sourceOpen
              ? 'rotate-90'
              : ''}"
          /><span class="text-[13px] text-secondary-foreground"
            >Binary download source</span
          ></span
        ><span class="truncate font-mono text-[11px] text-ink-quiet"
          >official release · checksum pinned</span
        ></button
      >
      {#if sourceOpen}<div class="border-b border-hairline py-3">
          <RuntimeDownloadSource
            source={entry.source}
            backend={sourceBackend}
          />
        </div>{/if}
    {/if}
    {#if instance}
      <button
        type="button"
        class="flex min-h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
        aria-expanded={manageOpen}
        onclick={() => (manageOpen = !manageOpen)}
        ><span class="flex items-center gap-2"
          ><ChevronRightIcon
            class="size-3.5 text-muted-foreground {manageOpen
              ? 'rotate-90'
              : ''}"
          /><span class="text-[13px] text-secondary-foreground"
            >Manage runtime</span
          ></span
        ><span class="font-mono text-[11px] text-ink-quiet"
          >remove files · delete entry</span
        ></button
      >
      {#if manageOpen}<div
          class="flex flex-wrap gap-1.5 border-b border-hairline py-3"
        >
          <Button
            variant="outline"
            size="xs"
            disabled={actionLocked || running || !installed}
            onclick={() => (confirming = { kind: "files" })}
            ><Trash2Icon class="size-3" />Remove downloaded files</Button
          ><Button
            variant="outline"
            size="xs"
            class="text-destructive"
            disabled={actionLocked || running}
            onclick={() => (confirming = { kind: "instance" })}
            >Delete runtime</Button
          >
        </div>{/if}
    {/if}
  </div>
  {#if !layout}
    <div
      class="flex shrink-0 flex-col border-t border-hairline"
      class:h-56={!panelCollapsed}
    >
      <PanelTabs
        tabs={[{ id: "output", label: "Runtime output" }]}
        bind:active={panelTab}
        bind:collapsed={panelCollapsed}
        note="memory only · not written to disk"
      />
      {#if !panelCollapsed}<RuntimeOutputDrawer
          instanceID={instance?.id ?? ""}
        />{/if}
    </div>
  {/if}
</div>
<Dialog.Root
  open={confirming !== null}
  onOpenChange={(open) => {
    if (!open) confirming = null;
  }}
>
  <Dialog.Content showCloseButton={false}>
    <Dialog.Header
      ><Dialog.Title
        >{confirming?.kind === "model"
          ? `Delete model: ${confirming.model.name}?`
          : confirming?.kind === "files"
            ? `Remove runtime files: ${instance?.name}?`
            : `Delete runtime instance: ${instance?.name}?`}</Dialog.Title
      ><Dialog.Description
        >{confirming?.kind === "model"
          ? "Deletes this downloaded model. If selected, this Connection becomes unavailable until you download it again or select another model."
          : confirming?.kind === "files"
            ? "Deletes this runtime's binary and downloaded models. The instance and its saved Connections remain for repair. Other installations are untouched."
            : "Remove or reassign every Connection referencing this instance first. Deleting its entry does not delete downloaded files; remove runtime files separately if wanted."}</Dialog.Description
      ></Dialog.Header
    >
    <Dialog.Footer
      ><Button variant="outline" onclick={() => (confirming = null)}
        >Keep</Button
      ><Button
        variant="destructive"
        disabled={actionLocked || running}
        onclick={confirmRemoval}>Confirm removal</Button
      ></Dialog.Footer
    >
  </Dialog.Content>
</Dialog.Root>
