<script lang="ts">
  import { onDestroy } from "svelte";
  import {
    ProviderID,
    type BinaryOptions,
    type InstanceStatus,
    type Model,
    type ProviderDescriptor,
  } from "$bindings/managedruntime";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import BoxIcon from "@lucide/svelte/icons/box";
  import CpuIcon from "@lucide/svelte/icons/cpu";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import PackageIcon from "@lucide/svelte/icons/package";
  import SlidersHorizontalIcon from "@lucide/svelte/icons/sliders-horizontal";
  import TagIcon from "@lucide/svelte/icons/tag";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import WrenchIcon from "@lucide/svelte/icons/wrench";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import * as Dialog from "$lib/components/ui/dialog";
  import RuntimeDownloadSource from "$lib/components/settings/RuntimeDownloadSource.svelte";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import RuntimeOutputDrawer from "./RuntimeOutputDrawer.svelte";
  import RuntimeModelCatalog from "./RuntimeModelCatalog.svelte";
  import PanelTabs from "$lib/components/shell/PanelTabs.svelte";
  import { getWorkbenchLayout } from "$lib/workbench-layout.svelte";
  import { backendLabel, runtimePresentation } from "$lib/utils/managedRuntime";
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
  const modelActionsLocked = $derived(actionLocked || !installed || !instance);
  const models = $derived(
    (status?.models?.length ? status.models : (entry.models ?? [])).filter(
      (model) => entry.models?.some((qualified) => qualified.id === model.id),
    ),
  );
  const selectedModel = $derived(
    models.find((model) => model.id === instance?.model),
  );
  const metadata = $derived(
    [
      {
        label: "Version",
        value: status?.version || entry.version,
        icon: TagIcon,
        technical: true,
      },
      {
        label: "Backend",
        value: status?.backend ? backendLabel(status.backend) : "",
        icon: CpuIcon,
        technical: false,
      },
      {
        label: "Selected model",
        value: status?.selectedModel,
        icon: BoxIcon,
        technical: true,
      },
    ].filter((fact) => fact.value),
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
</script>

<div class="flex min-h-0 min-w-0 flex-1 flex-col">
  <div
    class="flex min-h-11 shrink-0 flex-wrap items-center justify-between gap-2 border-b border-hairline px-5 py-1.5"
  >
    <div class="flex min-w-0 items-center gap-2.5">
      <h2 class="content-title truncate font-display tracking-tight">
        {entry.name}
      </h2>
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
    <dl class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
      {#each metadata as fact (fact.label)}
        <div
          class="flex min-w-0 items-center gap-1.5"
          title={`${fact.label}: ${fact.value}`}
        >
          <fact.icon
            class="size-3.5 shrink-0 text-muted-foreground"
            aria-hidden="true"
          />
          <dt class="sr-only">{fact.label}</dt>
          <dd
            class="min-w-0 truncate text-xs {fact.technical
              ? 'font-mono text-muted-foreground'
              : 'font-medium text-foreground'}"
          >
            {fact.value}
          </dd>
        </div>
      {/each}
    </dl>
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
      <div class="flex flex-col gap-3 border-b border-hairline py-3">
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="content-section-title">
              {entry.supported
                ? busy
                  ? "Installation in progress"
                  : "Not installed"
                : "Unavailable on this machine"}
            </p>
            <p class="content-meta mt-0.5">
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
            <p class="content-section-title">
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
      </div>
    {/if}
    <RuntimeModelCatalog
      {models}
      {entry}
      {row}
      {installed}
      {busy}
      {view}
      locked={modelActionsLocked}
      cancelling={runtime.pendingFor(instance?.id ?? "") === "Cancelling"}
      onRefresh={() => {
        if (instance && installed)
          act(() => runtime.run(instance.id, "RefreshCatalog"));
      }}
      onGet={(model) => {
        if (instance && installed && !running)
          act(() => runtime.downloadModel(instance.id, model.id));
      }}
      onSelect={(model) => {
        if (instance && installed && !running)
          act(() => runtime.saveInstance({ ...instance, model: model.id }));
      }}
      onRemove={(model) => {
        if (instance && installed && !running && !actionLocked)
          confirming = { kind: "model", model };
      }}
      onCancel={() => {
        if (instance) void runtime.cancel(instance.id);
      }}
    />
    {#if instance}
      <button
        type="button"
        class="content-disclosure flex min-h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
        aria-expanded={preferencesOpen}
        onclick={() => (preferencesOpen = !preferencesOpen)}
        ><span class="flex min-w-0 items-center gap-2"
          ><ChevronRightIcon
            aria-hidden="true"
            class="size-3.5 shrink-0 text-muted-foreground {preferencesOpen
              ? 'rotate-90'
              : ''}"
          /><SlidersHorizontalIcon
            class="content-section-icon"
            aria-hidden="true"
          /><span class="truncate">Runtime preferences</span></span
        ><span class="content-meta max-w-[45%] truncate text-right"
          >{instance.autoStart ? "starts with Freehand" : "manual start"}</span
        ></button
      >
      {#if preferencesOpen}
        <div class="space-y-3 border-b border-hairline py-3">
          <div class="flex items-center justify-between gap-3">
            <div>
              <label for={`${uid}-autostart`} class="content-value"
                >Start when Freehand launches</label
              >
              <p class="content-meta mt-1">
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
              <legend class="content-section-title mb-2">Runtime binary</legend>
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
            <p class="content-meta">
              Stop to change binary. Switching downloads the selected binary and
              keeps models and saved Connections.
            </p>{/if}
          <p class="content-meta">
            Stopping or removing files keeps saved Connections selected. There
            is no automatic fallback.
          </p>
        </div>
      {/if}
    {/if}
    {#if entry.source}
      <button
        type="button"
        class="content-disclosure flex min-h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
        aria-expanded={sourceOpen}
        onclick={() => (sourceOpen = !sourceOpen)}
        ><span class="flex min-w-0 items-center gap-2"
          ><ChevronRightIcon
            aria-hidden="true"
            class="size-3.5 shrink-0 text-muted-foreground {sourceOpen
              ? 'rotate-90'
              : ''}"
          /><PackageIcon class="content-section-icon" aria-hidden="true" /><span
            class="truncate">Binary download source</span
          ></span
        ><span class="content-meta max-w-[45%] truncate text-right"
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
        class="content-disclosure flex min-h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
        aria-expanded={manageOpen}
        onclick={() => (manageOpen = !manageOpen)}
        ><span class="flex min-w-0 items-center gap-2"
          ><ChevronRightIcon
            aria-hidden="true"
            class="size-3.5 shrink-0 text-muted-foreground {manageOpen
              ? 'rotate-90'
              : ''}"
          /><WrenchIcon class="content-section-icon" aria-hidden="true" /><span
            class="truncate">Manage runtime</span
          ></span
        ><span class="content-meta max-w-[45%] truncate text-right"
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
