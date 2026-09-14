<script lang="ts">
  import type {
    InstanceStatus,
    Model,
    ProviderDescriptor,
  } from "$bindings/managedruntime";
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import { backendLabel, runtimePresentation } from "$lib/utils/managedRuntime";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import TrashIcon from "@lucide/svelte/icons/trash-2";
  import LocalRuntimeModelCatalog from "./LocalRuntimeModelCatalog.svelte";

  let {
    runtime,
    row,
    entry,
    view,
    uid,
    locked,
    operating,
    disabled,
    switchable,
    hasDuplicates,
    act,
    onRemove,
    onRemoveModel,
  }: {
    runtime: ManagedRuntimeState;
    row: InstanceStatus;
    entry: ProviderDescriptor;
    view: ReturnType<typeof runtimePresentation>;
    uid: string;
    locked: boolean;
    operating: boolean;
    disabled: boolean;
    switchable: boolean;
    hasDuplicates: boolean;
    act: (action: () => Promise<unknown>) => void;
    onRemove: (kind: "files" | "instance") => void;
    onRemoveModel: (model: Model) => void;
  } = $props();
  const id = $derived(row.instance.id);
  const status = $derived(row.status);
  const running = $derived(status.state === "running");
</script>

<section class="space-y-3" aria-label="Runtime setup">
  <div class="flex flex-wrap items-center justify-between gap-2">
    <h4 class="text-sm font-semibold">Runtime binary</h4>
    <span class="text-xs text-secondary-foreground">
      {status.version || entry.version || "Not installed"}
      {#if status.backend && !switchable}
        · {backendLabel(status.backend)}{/if}
    </span>
  </div>
  {#if switchable && status.supported && status.backend}
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
            disabled={locked || running || status.backend === backend}
            onclick={() => {
              if (!running) act(() => runtime.installBackend(id, backend));
            }}>{backendLabel(backend)}</Button
          >
        {/each}
        {#if running}<span class="text-xs text-secondary-foreground"
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
        Switching keeps downloaded models and saved Connections. Existing
        installations are never changed automatically.
        {#if entry.backends?.includes("metal")}
          Metal uses the Apple Silicon GPU and shares memory with NeMo and other
          apps. Choose CPU to disable GPU offload.
        {:else if entry.backends?.includes("cuda")}
          CUDA 12.4 requires an NVIDIA GPU with compute capability 5.0+ and
          driver 551.78 or newer. It may share GPU memory with NeMo or other
          apps.
        {/if}
      </p>
    </details>
  {/if}
  {#if operating && !view.startup && (status.operation?.kind !== "download" || status.operation.model === row.instance.model)}<div
      class="mt-3"
      role="status"
    >
      <p class="text-xs text-secondary-foreground">
        {view.operationModel || view.activity}{view.percent !== null
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
      {view.operationModel ? `${view.operationModel}: ` : ""}{view.completion}
    </p>{/if}
</section>
<LocalRuntimeModelCatalog
  {runtime}
  {row}
  {entry}
  {view}
  {locked}
  {operating}
  {disabled}
  {act}
  {onRemoveModel}
/>
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
        <p>The API model identity differs from the selected catalog key.</p>
      {/if}
      <p>
        Stopping or removing files keeps saved Connections selected; there is no
        automatic fallback.
      </p>
    </div>
    <div class="flex items-center justify-between gap-3">
      <div>
        <label for={`${uid}-autostart`} class="text-sm font-medium"
          >Start when Freehand launches</label
        >
        <p class="mt-1 text-xs text-secondary-foreground">
          Uses this runtime’s selected model. Does not download missing files.
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
          onRemove("files");
        }}>Remove runtime files</Button
      >{#if hasDuplicates}<Button
          variant="ghost"
          size="sm"
          disabled={locked}
          onclick={() => {
            onRemove("instance");
          }}><TrashIcon class="size-3.5" />Delete duplicate entry</Button
        >{/if}
    </div>
  </div>
</details>
