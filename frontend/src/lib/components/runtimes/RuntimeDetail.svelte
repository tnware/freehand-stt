<script lang="ts">
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
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import { Button } from "$lib/components/ui/button";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import RuntimeOutputDrawer from "./RuntimeOutputDrawer.svelte";
  import PanelTabs from "$lib/components/shell/PanelTabs.svelte";
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

  const status = $derived(row?.status);
  const instance = $derived(row?.instance);
  const view = $derived(runtimePresentation(status));
  const running = $derived(status?.state === "running");
  const installed = $derived(
    !!status && !["not_installed", "installing"].includes(status.state),
  );
  const busy = $derived(!!instance && runtime.isBusy(instance.id));
  const models = $derived(status?.models ?? entry.models ?? []);

  let preferencesOpen = $state(false);
  let sourceOpen = $state(false);
  let manageOpen = $state(false);
  let panelTab = $state("output");
  let panelCollapsed = $state(false);
  let probing = $state(false);
  let recommendation = $state<BinaryOptions | null>(null);
  let binaryChoice = $state("auto");
  let confirming = $state<"files" | "instance" | null>(null);

  const problem = $derived(
    (instance && runtime.errorFor(instance.id)) || status?.error || "",
  );
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
  // llama.cpp and whisper.cpp ship per-backend binaries, so they are probed
  // before anything is fetched. Everything else installs directly.
  const switchable = $derived(
    entry.id === ProviderID.LlamaCPP || entry.id === ProviderID.WhisperCPP,
  );

  function install(backend?: string) {
    if (locked || runtime.busy || probing || !entry.supported) return;
    if (switchable && !backend) {
      probing = true;
      recommendation = null;
      void runtime
        .binaryOptions(entry.id)
        .then((options) => {
          recommendation = options;
          binaryChoice = "auto";
        })
        .finally(() => (probing = false));
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
      recommendation = null;
      if (backend) await runtime.installBackend(target.id, backend);
      else await runtime.run(target.id, "Install");
    })();
  }

  function confirmRemoval() {
    if (!confirming || !instance || locked) return;
    const kind = confirming;
    confirming = null;
    act(() =>
      kind === "files"
        ? runtime.run(instance.id, "Remove")
        : runtime.deleteInstance(instance.id),
    );
  }

  function size(bytes: number): string {
    if (!bytes) return "—";
    const units = ["KB", "MB", "GB"];
    let value = bytes / 1024;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
      value /= 1024;
      unit += 1;
    }
    return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${units[unit]}`;
  }

  const task = (model: Model): string =>
    model.realtime ? "Streaming" : "Batch";

  function source(model: Model): string {
    if (!model.source) return `${entry.name} model manager`;
    return model.source.publisher || model.source.metadataOrigin || "Catalog";
  }

  function act(action: () => Promise<unknown>) {
    void action();
  }

  // The binary actually pinned for this machine, so "checksum pinned" is a
  // claim the panel can substantiate rather than a slogan.
  const artifact = $derived(
    (entry.source?.artifacts ?? []).find(
      (item) => !status?.backend || item.backend === status.backend,
    ) ?? entry.source?.artifacts?.[0],
  );
</script>

<div class="flex min-h-0 flex-1 flex-col">
  <div
    class="flex h-12 shrink-0 items-center justify-between gap-3 border-b border-hairline px-5"
  >
    <div class="flex min-w-0 items-center gap-2.5">
      <h3 class="truncate font-display text-[15px] font-semibold tracking-tight">
        {entry.name}
      </h3>
      <StatusBadge
        tone={running
          ? "success"
          : status?.state === "error"
            ? "danger"
            : status?.state === "starting"
              ? "accent"
              : "neutral"}
        dot>{view.label}</StatusBadge
      >
    </div>
    <div class="flex shrink-0 items-center gap-1.5">
      {#if installed}
        <Button
          variant="outline"
          size="xs"
          disabled={locked || busy || !running}
          onclick={() =>
            act(async () => {
              await runtime.run(instance!.id, "Stop");
              await runtime.run(instance!.id, "Start");
            })}>Restart</Button
        >
        <Button
          variant="outline"
          size="xs"
          disabled={locked || busy}
          onclick={() =>
            act(() => runtime.run(instance!.id, running ? "Stop" : "Start"))}
          >{running ? "Stop" : "Start"}</Button
        >
      {/if}
    </div>
  </div>

  <!-- The literals a runtime is actually identified by. Freehand does not
       report a pid or a port to the renderer, so this states what it knows. -->
  <div
    class="flex h-[38px] shrink-0 items-center gap-3 overflow-x-auto border-b border-hairline px-5"
  >
    {#each [status?.version || entry.version, status?.backend ? backendLabel(status.backend) : "", status?.selectedModel] as fact, index (index)}
      {#if fact}
        {#if index > 0}
          <span class="h-3 w-px shrink-0 bg-border" aria-hidden="true"></span>
        {/if}
        <span class="shrink-0 font-mono text-[10px] text-ink-quiet">{fact}</span>
      {/if}
    {/each}
    <span class="flex-1"></span>
    <button
      type="button"
      class="shrink-0 text-[11px] text-ink-quiet underline-offset-2 hover:text-accent-text hover:underline"
      onclick={onOpenConnections}
      >Appears to your chains as a built-in Connection</button
    >
  </div>

  <div class="min-h-0 flex-1 overflow-y-auto px-5">
    {#if !installed}
      <!-- Nothing is fetched until a binary is chosen and install is pressed:
           probing reads host capability only. -->
      <div class="flex flex-col gap-3 py-4">
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <p class="text-[13px] font-medium">
              {entry.supported ? "Not installed" : "Unavailable on this machine"}
            </p>
            <p class="mt-0.5 font-mono text-[10px] text-ink-quiet">
              {entry.unavailableReason ||
                (switchable
                  ? "per-backend binaries · choose one to install"
                  : "installs the pinned official release")}
            </p>
          </div>
          <Button
            size="xs"
            disabled={locked || probing || runtime.busy || !entry.supported}
            onclick={() => install()}
          >
            <DownloadIcon class="size-3" />
            {probing ? "Checking…" : recommendation ? "Re-check" : "Install"}
          </Button>
        </div>

        {#if probing}
          <p class="font-mono text-[10px] text-ink-quiet" role="status">
            Checking host binary options · no files are downloaded
          </p>
        {/if}

        {#if recommendation}
          <div class="rounded-lg border border-hairline p-3">
            <p class="text-[12.5px] font-medium">
              Recommended: {backendLabel(recommendation.recommendedBackend)}
            </p>
            <p class="mt-0.5 font-mono text-[10px] text-ink-quiet">
              {recommendation.os} · {recommendation.architecture}{recommendation.reason
                ? ` · ${recommendation.reason}`
                : ""}
            </p>
            <div class="mt-2.5 flex flex-wrap gap-1.5">
              {#each [{ backend: "auto", label: "Auto", supported: true, available: true, reason: "Use the recommended binary" }, ...(recommendation.options ?? [])] as option (option.backend)}
                <button
                  type="button"
                  class="h-6 rounded-md border px-2 text-[11px] transition-colors disabled:opacity-40 {binaryChoice ===
                  option.backend
                    ? 'border-accent-edge bg-accent-wash text-accent-text'
                    : 'border-border text-secondary-foreground hover:bg-subtle-fill-hover'}"
                  aria-pressed={binaryChoice === option.backend}
                  disabled={!option.supported || !option.available}
                  title={option.reason}
                  onclick={() => (binaryChoice = option.backend)}
                  >{option.backend === "auto"
                    ? "Auto"
                    : backendLabel(option.backend)}</button
                >
              {/each}
            </div>
            <div class="mt-3 flex items-center justify-between gap-3">
              <p class="font-mono text-[10px] text-ink-quiet">
                downloads on install · checksum pinned
              </p>
              <Button
                size="xs"
                disabled={locked || runtime.busy || !choiceAvailable}
                onclick={() => install(chosenBackend)}
                >Download and install</Button
              >
            </div>
          </div>
        {/if}

        {#if entry.models?.length}
          <div class="mt-1">
            <p
              class="mb-1 text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
            >
              Models it can run
            </p>
            {#each entry.models as model (model.id)}
              <div class="flex h-11 items-center gap-3 border-b border-hairline">
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-[12.5px]">{model.name}</span>
                  <span class="block truncate font-mono text-[10px] text-ink-quiet"
                    >{model.id}</span
                  >
                </span>
                <span class="shrink-0 font-mono text-[10px] text-ink-quiet"
                  >{size(model.sizeBytes)}</span
                >
                {#if model.recommended}
                  <span
                    class="shrink-0 rounded-sm border border-accent-edge px-1.5 py-0.5 text-[10px] text-accent-text"
                    >Recommended</span
                  >
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {:else}
    <div class="flex h-10 items-center justify-between gap-3">
      <span
        class="text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
        >Models</span
      >
      <div class="flex items-center gap-2.5">
        <span class="hidden text-[11px] text-ink-quiet min-[900px]:inline"
          >Browsing is metadata-only. Only Get fetches files.</span
        >
        <Button
          variant="outline"
          size="xs"
          disabled={locked || busy || !installed}
          onclick={() => act(() => runtime.run(instance!.id, "RefreshCatalog"))}
        >
          <RefreshCwIcon class="size-3" />
          Refresh catalog
        </Button>
      </div>
    </div>

    <div
      class="flex h-[26px] items-center border-b border-hairline text-[10px] font-semibold tracking-[0.07em] text-ink-quiet uppercase"
    >
      <span class="min-w-0 flex-1">Model</span>
      <span class="w-[92px] shrink-0">Task</span>
      <span class="w-[72px] shrink-0 text-right">Size</span>
      <span class="hidden w-[150px] shrink-0 pl-6 min-[1000px]:block">Source</span>
      <span class="w-[104px] shrink-0 pl-4">State</span>
      <span class="w-[84px] shrink-0 text-right">Action</span>
    </div>

    {#each models as model (model.id)}
      {@const selected = model.id === (instance?.model || status?.selectedModel)}
      {@const loaded = selected && running}
      <div
        class="flex h-12 items-center border-b border-hairline {selected
          ? 'bg-accent-wash'
          : ''}"
      >
        <span class="min-w-0 flex-1 pl-2">
          <span class="block truncate text-[12.5px] text-foreground"
            >{model.name}</span
          >
          <span class="block truncate font-mono text-[10px] text-ink-quiet"
            >{model.id}</span
          >
        </span>
        <span class="w-[92px] shrink-0 text-[11.5px] text-secondary-foreground"
          >{task(model)}</span
        >
        <span class="w-[72px] shrink-0 text-right font-mono text-[10px]"
          >{size(model.sizeBytes)}</span
        >
        <span
          class="hidden w-[150px] shrink-0 truncate pl-6 text-[11.5px] text-muted-foreground min-[1000px]:block"
          >{source(model)}</span
        >
        <span
          class="flex w-[104px] shrink-0 items-center gap-1.5 pl-4 text-[11.5px] {loaded
            ? 'text-success'
            : model.installed
              ? 'text-muted-foreground'
              : 'text-ink-quiet'}"
        >
          {#if loaded}
            <span class="size-[7px] rounded-full bg-success" aria-hidden="true"
            ></span>
          {/if}
          {loaded
            ? "Loaded"
            : selected
              ? "Selected"
              : model.installed
                ? "Downloaded"
                : "Not installed"}
        </span>
        <span class="flex w-[84px] shrink-0 items-center justify-end gap-1">
          {#if !model.installed}
            <Button
              variant="outline"
              size="xs"
              disabled={locked || busy || !installed}
              onclick={() => act(() => runtime.downloadModel(instance!.id, model.id))}
            >
              <DownloadIcon class="size-3" />
              Get
            </Button>
          {:else if !selected}
            <Button
              variant="outline"
              size="xs"
              disabled={locked || busy}
              onclick={() =>
                act(() =>
                  runtime.saveInstance({ ...instance!, model: model.id }),
                )}>Select</Button
            >
          {:else}
            <Button
              variant="ghost"
              size="xs"
              class="size-6 p-0"
              disabled={locked || busy}
              aria-label={`Remove ${model.name} files`}
              title="Remove downloaded files"
              onclick={() => act(() => runtime.removeModel(instance!.id, model.id))}
            >
              <Trash2Icon class="size-3.5" />
            </Button>
          {/if}
        </span>
      </div>
    {:else}
      <p class="py-6 text-center text-[13px] text-muted-foreground">
        {installed
          ? "No models in this runtime's catalog yet."
          : "Install this runtime to browse its models."}
      </p>
    {/each}

    {#each [{ open: preferencesOpen, toggle: () => (preferencesOpen = !preferencesOpen), label: "Runtime preferences", value: `${status?.realtime ? "streaming" : "batch"} · ${status?.backend ? backendLabel(status.backend) : "backend unset"}` }, { open: sourceOpen, toggle: () => (sourceOpen = !sourceOpen), label: "Binary download source", value: artifact ? `official release · ${artifact.backend} · checksum pinned` : "official release" }] as disclosure (disclosure.label)}
      <button
        type="button"
        class="flex h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
        aria-expanded={disclosure.open}
        onclick={disclosure.toggle}
      >
        <span class="flex items-center gap-2">
          <ChevronRightIcon
            class="size-3.5 text-muted-foreground transition-transform {disclosure.open
              ? 'rotate-90'
              : ''}"
            aria-hidden="true"
          />
          <span class="text-[12.5px] text-secondary-foreground"
            >{disclosure.label}</span
          >
        </span>
        <span class="truncate font-mono text-[10px] text-ink-quiet"
          >{disclosure.value}</span
        >
      </button>
      {#if disclosure.open && disclosure.label === "Binary download source" && entry.source}
        <dl
          class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 border-b border-hairline py-2.5 pl-7 font-mono text-[10px]"
        >
          {#each [["Repository", entry.source.repositoryURL], ["Release", entry.source.releaseURL], ["File", artifact?.filename ?? ""], ["Checksum", artifact?.sha256 ?? ""]] as pair (pair[0])}
            {#if pair[1]}
              <dt class="text-ink-quiet">{pair[0]}</dt>
              <dd class="truncate text-secondary-foreground">{pair[1]}</dd>
            {/if}
          {/each}
        </dl>
      {/if}
    {/each}

    <button
      type="button"
      class="flex h-[38px] w-full items-center justify-between gap-3 border-b border-hairline text-left"
      aria-expanded={manageOpen}
      onclick={() => (manageOpen = !manageOpen)}
    >
      <span class="flex items-center gap-2">
        <ChevronRightIcon
          class="size-3.5 text-muted-foreground transition-transform {manageOpen
            ? 'rotate-90'
            : ''}"
          aria-hidden="true"
        />
        <span class="text-[12.5px] text-secondary-foreground"
          >Manage runtime</span
        >
      </span>
      <span class="font-mono text-[10px] text-ink-quiet"
        >repair · remove · retry</span
      >
    </button>
    {#if manageOpen}
      <div class="flex flex-col gap-2 border-b border-hairline py-3">
        {#if problem}
          <p
            class="flex items-start gap-1.5 text-[11.5px] leading-snug text-destructive"
            role="alert"
          >
            <TriangleAlertIcon class="mt-0.5 size-3.5 shrink-0" />
            {problem}
          </p>
        {/if}
        <div class="flex flex-wrap items-center gap-1.5">
          {#if instance && runtime.canRetry(instance.id) && !busy}
            <Button
              variant="outline"
              size="xs"
              disabled={locked}
              onclick={() => act(() => runtime.retry(instance.id))}>Retry</Button
            >
          {/if}
          <Button
            variant="outline"
            size="xs"
            disabled={locked || busy || running}
            onclick={() => (confirming = "files")}
          >
            <Trash2Icon class="size-3" />
            Remove downloaded files
          </Button>
          <Button
            variant="outline"
            size="xs"
            class="text-destructive"
            disabled={locked || busy || running}
            onclick={() => (confirming = "instance")}>Delete runtime</Button
          >
        </div>
        {#if confirming}
          <!-- Removal is explicit and says exactly what it takes with it. -->
          <div
            class="rounded-lg border border-warning/30 bg-warning/[0.08] p-3"
            role="alertdialog"
            aria-label="Confirm removal"
          >
            <p class="text-[12px] leading-snug text-secondary-foreground">
              {confirming === "files"
                ? "Stops this runtime and deletes its binary and downloaded models. The instance and its saved Connections stay, so it can be repaired."
                : "Deletes this runtime instance. Saved Connections that point at it will need a new target."}
            </p>
            <div class="mt-2.5 flex gap-1.5">
              <Button size="xs" onclick={confirmRemoval}>
                {confirming === "files" ? "Remove files" : "Delete"}
              </Button>
              <Button variant="ghost" size="xs" onclick={() => (confirming = null)}
                >Cancel</Button
              >
            </div>
          </div>
        {/if}
      </div>
    {/if}
    {/if}
  </div>

  <div
    class="flex shrink-0 flex-col border-t border-hairline"
    class:h-44={!panelCollapsed}
  >
    <PanelTabs
      tabs={[{ id: "output", label: "Runtime output" }]}
      bind:active={panelTab}
      bind:collapsed={panelCollapsed}
      note="memory only · not written to disk"
    />
    {#if !panelCollapsed}
      <RuntimeOutputDrawer instanceID={instance?.id ?? ""} {running} />
    {/if}
  </div>
</div>
