<script lang="ts">
  import { onDestroy } from "svelte";
  import { Clipboard } from "@wailsio/runtime";
  import * as Manager from "$bindings/managedruntime/manager";
  import type {
    InstanceStatus,
    Model,
    ProviderDescriptor,
  } from "$bindings/managedruntime";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import { Button } from "$lib/components/ui/button";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import ProcessOutputTerminal from "$lib/components/ProcessOutputTerminal.svelte";
  import { ProcessOutputState } from "$lib/stores/process-output.svelte";
  import { backendLabel, runtimePresentation } from "$lib/utils/managedRuntime";
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";

  let {
    runtime,
    row,
    entry,
    locked = false,
    onOpenConnections,
  }: {
    runtime: ManagedRuntimeState;
    row: InstanceStatus | undefined;
    entry: ProviderDescriptor;
    locked?: boolean;
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

  const output = new ProcessOutputState(Manager);
  let following = $state(true);
  let preferencesOpen = $state(false);
  let sourceOpen = $state(false);

  // Output is per instance and bounded to memory, so it follows the selection
  // and is released the moment the pane goes away.
  $effect(() => {
    const id = running ? (instance?.id ?? "") : "";
    if (output.instanceID !== id) void output.select(id);
  });
  $effect(() => {
    if (!output.accepted) return;
    const timer = setInterval(() => void output.poll(), 1000);
    return () => clearInterval(timer);
  });
  onDestroy(() => output.dispose());

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
          disabled={!running}
          onclick={() => void output.select(instance?.id ?? "")}
        >
          View output
        </Button>
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
  </div>

  <div class="flex h-44 shrink-0 flex-col border-t border-hairline">
    <div
      class="flex h-7 shrink-0 items-center justify-between border-b border-hairline px-3"
    >
      <span
        class="text-[10px] font-semibold tracking-[0.08em] text-secondary-foreground uppercase"
        >Runtime output</span
      >
      <span class="font-mono text-[10px] text-ink-quiet"
        >memory only · not written to disk</span
      >
    </div>
    <div class="relative min-h-0 flex-1">
      {#if !running}
        <p
          class="flex h-full items-center justify-center px-4 text-center text-[12px] text-muted-foreground"
        >
          Output is available while this runtime is running.
        </p>
      {:else}
        <ProcessOutputTerminal
          chunks={output.chunks}
          revision={output.revision}
          enabled={output.accepted}
          busy={output.busy}
          bind:following
          onclear={() => void output.clear()}
          oncopy={Clipboard.SetText}
          describedby={!output.accepted ? "runtime-output-consent" : undefined}
        >
          {#if !output.accepted}
            <div
              class="absolute inset-x-0 top-0 z-10 flex flex-wrap items-center justify-between gap-3 border-b border-warning/25 bg-card px-4 py-3"
            >
              <p
                id="runtime-output-consent"
                class="text-[12px] leading-relaxed text-secondary-foreground"
              >
                <span class="block font-semibold text-foreground"
                  >Show sensitive output</span
                >
                Output may include transcripts, prompts and file paths.
              </p>
              <Button
                size="xs"
                class="shrink-0"
                disabled={!output.instanceID || output.busy}
                onclick={() => void output.show()}>Show output</Button
              >
            </div>
          {/if}
        </ProcessOutputTerminal>
      {/if}
    </div>
  </div>
</div>
