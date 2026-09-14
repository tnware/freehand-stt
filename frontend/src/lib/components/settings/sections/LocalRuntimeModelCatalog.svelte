<script lang="ts">
  import type {
    InstanceStatus,
    Model,
    ProviderDescriptor,
  } from "$bindings/managedruntime";
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import { modelSize, runtimePresentation } from "$lib/utils/managedRuntime";
  import ModelDownloadSource from "$lib/components/settings/ModelDownloadSource.svelte";
  import { Button } from "$lib/components/ui/button";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import CheckIcon from "@lucide/svelte/icons/check";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RefreshIcon from "@lucide/svelte/icons/refresh-cw";
  import TrashIcon from "@lucide/svelte/icons/trash-2";

  let {
    runtime,
    row,
    entry,
    view,
    locked,
    operating,
    disabled,
    act,
    onRemoveModel,
  }: {
    runtime: ManagedRuntimeState;
    row: InstanceStatus;
    entry: ProviderDescriptor;
    view: ReturnType<typeof runtimePresentation>;
    locked: boolean;
    operating: boolean;
    disabled: boolean;
    act: (action: () => Promise<unknown>) => void;
    onRemoveModel: (model: Model) => void;
  } = $props();
  const id = $derived(row.instance.id);
  const status = $derived(row.status);
  const running = $derived(status.state === "running");
  const models = $derived(
    (status.models?.length ? status.models : (entry.models ?? [])).filter(
      (model) => entry.models?.some((qualified) => qualified.id === model.id),
    ),
  );
</script>

<section
  class="space-y-3 border-t border-hairline pt-4"
  aria-label="Model catalog"
>
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h4 class="text-sm font-semibold">Models</h4>
      <p class="mt-1 text-xs text-secondary-foreground">
        Browsing is metadata-only. Only Download fetches model files.
      </p>
    </div>
    <Button
      variant="outline"
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
      entry.models?.find((candidate) => candidate.id === model.id)?.source ??
      model.source}
    {@const downloading =
      status.operation?.kind === "download" &&
      status.operation.model === model.id}
    <article
      class={`rounded-lg border p-3 ${model.id === row.instance.model ? "border-accent-edge bg-accent-wash" : "border-hairline bg-background"}`}
      aria-label={model.name}
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0 flex-[1_1_12rem]">
          <h5 class="text-sm font-semibold" title={model.description}>
            {model.name}
          </h5>
          <p class="mt-1 text-xs text-secondary-foreground">
            {modelSize(model.sizeBytes)}{#if model.recommended}
              · Recommended{/if}
            {#if model.installed}
              · Downloaded{/if}
          </p>
        </div>
        <div class="ml-auto flex shrink-0 items-center gap-2">
          {#if model.id === row.instance.model}
            <StatusBadge tone="accent"
              ><span class="inline-flex items-center gap-1.5"
                ><CheckIcon class="size-3.5" />Selected</span
              ></StatusBadge
            >
          {:else}
            <Button
              variant="soft"
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
              disabled={disabled || runtime.pendingFor(id) === "Cancelling"}
              onclick={() => void runtime.cancel(id)}>Cancel</Button
            >{:else if model.installed}<Button
              variant="ghost"
              size="icon-sm"
              aria-label={`Delete ${model.name}`}
              title="Delete model"
              disabled={locked || running}
              onclick={() => {
                onRemoveModel(model);
              }}><TrashIcon class="size-3.5" /></Button
            >{:else}<Button
              variant="soft"
              size="sm"
              disabled={locked || running || !view.installed}
              onclick={() => act(() => runtime.downloadModel(id, model.id))}
              ><DownloadIcon class="size-3.5" />Download</Button
            >{/if}
        </div>
      </div>
      {#if downloading}
        <div class="mt-2 space-y-1" role="status">
          {#if operating}
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
            <p class="text-xs text-secondary-foreground">
              {runtime.pendingFor(id) || view.activity}{view.percent !== null
                ? ` · ${view.percent}%`
                : ""}
            </p>
            {#if view.transferred}<p
                class="text-xs tabular-nums text-secondary-foreground"
              >
                {view.transferred}
              </p>{/if}
          {:else if view.completion}
            <p class="text-sm">{view.completion}</p>
            {#if status.operation.error}<p class="text-xs text-destructive">
                {status.operation.error}
              </p>{/if}
          {/if}
        </div>
      {/if}
      {#if source}<div class="mt-2">
          <ModelDownloadSource {source} description={model.description} />
        </div>{:else}<p
          class="mt-2 text-xs leading-relaxed text-secondary-foreground"
        >
          {model.description}
        </p>{/if}
    </article>
  {/each}
  {#if !models.length}<p class="text-sm text-secondary-foreground">
      Install the runtime, then refresh its qualified model catalog.
    </p>{/if}
</section>
