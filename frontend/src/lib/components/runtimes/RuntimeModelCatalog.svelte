<script lang="ts">
  import { Role } from "$bindings/compatibility";
  import type {
    InstanceStatus,
    Model,
    ProviderDescriptor,
  } from "$bindings/managedruntime";
  import BoxIcon from "@lucide/svelte/icons/box";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import SearchIcon from "@lucide/svelte/icons/search";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import XIcon from "@lucide/svelte/icons/x";
  import { Button } from "$lib/components/ui/button";
  import { Input } from "$lib/components/ui/input";
  import ModelDownloadSource from "$lib/components/settings/ModelDownloadSource.svelte";
  import {
    isSpeechRuntimeModel,
    modelSize,
    runtimePresentation,
  } from "$lib/utils/managedRuntime";
  import {
    runtimeModelGroups,
    runtimeVariantLabel,
    type RuntimeModelFamily,
    type RuntimeModelGroup,
  } from "$lib/utils/runtimeCatalog";

  let {
    models,
    entry,
    row,
    installed,
    locked,
    busy,
    cancelling,
    view,
    onRefresh,
    onGet,
    onSelect,
    onRemove,
    onCancel,
    onDisableSpeech,
  }: {
    models: Model[];
    entry: ProviderDescriptor;
    row: InstanceStatus | undefined;
    installed: boolean;
    locked: boolean;
    busy: boolean;
    cancelling: boolean;
    view: ReturnType<typeof runtimePresentation>;
    onRefresh: () => void;
    onGet: (model: Model) => void;
    onSelect: (model: Model) => void;
    onRemove: (model: Model) => void;
    onCancel: () => void;
    onDisableSpeech: () => void;
  } = $props();

  const uid = $props.id();
  let filter = $state("");
  let filterInput: HTMLInputElement | null = $state(null);
  let expandedFamilies = $state<Record<string, boolean>>({});
  let expandedGroups = $state<Record<string, boolean>>({});
  let searchGroups = $state<{ query: string; groups: Record<string, boolean> }>(
    { query: "", groups: {} },
  );
  let searchExpansion = $state<{
    query: string;
    families: Record<string, boolean>;
  }>({ query: "", families: {} });
  let expandedModels = $state<Record<string, boolean>>({});
  const largeCatalog = $derived(models.length > 6);
  const query = $derived(largeCatalog ? filter.trim().toLowerCase() : "");
  const groups = $derived(runtimeModelGroups(entry.id, models));
  const families = $derived(groups.flatMap((group) => group.families));
  const initialFamily = $derived(
    families.find((family) =>
      family.models.some((model) => model.id === row?.instance.model),
    )?.id ??
      families.find((family) =>
        family.models.some((model) => model.recommended),
      )?.id ??
      families[0]?.id,
  );
  const visibleGroups = $derived(
    groups
      .map((group) => ({
        ...group,
        families: group.families
          .map((family) => ({
            ...family,
            models: query
              ? family.models.filter(
                  (model) =>
                    model.name.toLowerCase().includes(query) ||
                    model.id.toLowerCase().includes(query),
                )
              : family.models,
          }))
          .filter((family) => family.models.length),
      }))
      .filter((group) => group.families.length),
  );
  const running = $derived(row?.status.state === "running");
  const controlsLocked = $derived(locked || running);
  const catalogID = `${uid}-models`;
  const familyID = (id: string) => `${uid}-family-${id}`;
  const groupID = (id: string) => `${uid}-group-${id}`;
  const isGroupOpen = (group: RuntimeModelGroup) =>
    !group.label ||
    (query
      ? ((searchGroups.query === query
          ? searchGroups.groups[group.id]
          : undefined) ?? true)
      : (expandedGroups[group.id] ?? true));
  function toggleGroup(group: RuntimeModelGroup) {
    const open = !isGroupOpen(group);
    if (query)
      searchGroups = {
        query,
        groups: {
          ...(searchGroups.query === query ? searchGroups.groups : {}),
          [group.id]: open,
        },
      };
    else expandedGroups[group.id] = open;
  }
  const modelID = (id: string) => `${uid}-model-${encodeURIComponent(id)}`;
  const isFamilyOpen = (family: RuntimeModelFamily) =>
    !family.label ||
    (query
      ? ((searchExpansion.query === query
          ? searchExpansion.families[family.id]
          : undefined) ?? true)
      : (expandedFamilies[family.id] ??
        (!largeCatalog || family.id === initialFamily)));
  function toggleFamily(family: RuntimeModelFamily) {
    const open = !isFamilyOpen(family);
    if (query)
      searchExpansion = {
        query,
        families: {
          ...(searchExpansion.query === query ? searchExpansion.families : {}),
          [family.id]: open,
        },
      };
    else expandedFamilies[family.id] = open;
  }
  const task = (model: Model): string =>
    isSpeechRuntimeModel(model)
      ? "Speech"
      : model.contracts?.some(
            (contract) => contract.role === Role.PostProcessing,
          )
        ? "Cleanup"
        : model.realtime
          ? "Streaming"
          : "Batch";
  function clearFilter() {
    filter = "";
    filterInput?.focus();
  }
</script>

<div class="flex min-h-10 flex-wrap items-center justify-between gap-2 py-2">
  <h3 class="content-section-title flex items-center gap-2">
    <BoxIcon class="content-section-icon" aria-hidden="true" />Models
  </h3>
  <div class="flex min-w-0 flex-wrap items-center gap-2">
    {#if largeCatalog}
      <div class="relative w-44 max-w-full">
        <SearchIcon
          class="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          type="search"
          aria-label="Filter models"
          aria-controls={catalogID}
          placeholder="Filter name or ID"
          bind:value={filter}
          bind:ref={filterInput}
          class="h-7 pl-7 pr-7 text-xs [&::-webkit-search-cancel-button]:hidden"
        />
        {#if filter}<Button
            variant="ghost"
            size="icon-xs"
            class="absolute right-0.5 top-1/2 -translate-y-1/2"
            aria-label="Clear model filter"
            title="Clear model filter"
            onclick={clearFilter}
            ><XIcon class="size-3.5" aria-hidden="true" /></Button
          >{/if}
      </div>
    {/if}
    <Button
      variant="outline"
      size="xs"
      disabled={locked}
      title={!installed ? "Install this runtime first" : undefined}
      onclick={onRefresh}
      ><RefreshCwIcon class="size-3" aria-hidden="true" />Refresh catalog</Button
    >
  </div>
</div>
<p class="content-meta pb-2">
  {!installed
    ? "Install the runtime to download or select models. Browsing is metadata-only."
    : running
      ? "Stop to download, change or remove models."
      : "Browsing is metadata-only. Only Get fetches model files."}
</p>
{#if entry.models?.some(isSpeechRuntimeModel)}
  <p class="content-meta pb-2">
    Choose one transcription model and optionally a speech model. Both load in
    this runtime; Start and Stop affect both.
  </p>
{/if}
<div
  id={catalogID}
  class="model-catalog min-w-0"
  role="region"
  aria-label="Model catalog"
>
  {#each visibleGroups as group (group.id)}
    {@const count = group.families.reduce(
      (count, family) => count + family.models.length,
      0,
    )}
    {#if group.label}
      <button
        type="button"
        class="content-disclosure flex min-h-9 w-full items-center gap-2 border-b border-hairline bg-well px-1 text-left"
        aria-label={`${group.label} models`}
        aria-expanded={isGroupOpen(group)}
        aria-controls={groupID(group.id)}
        onclick={() => toggleGroup(group)}
      >
        <ChevronRightIcon
          class="size-3.5 shrink-0 text-muted-foreground {isGroupOpen(group)
            ? 'rotate-90'
            : ''}"
          aria-hidden="true"
        />
        <span>{group.label}</span>
        <span class="ml-auto text-xs font-normal text-muted-foreground"
          >{count} {count === 1 ? "model" : "models"}</span
        >
      </button>
    {/if}
    <div id={groupID(group.id)} hidden={!isGroupOpen(group)}>
      {#each group.families as family (family.id)}
        {#if family.label}
          <button
            type="button"
            class="content-disclosure flex min-h-8 w-full items-center gap-2 border-b border-hairline bg-well px-1 text-left"
            aria-label={`${family.label} models`}
            aria-expanded={isFamilyOpen(family)}
            aria-controls={familyID(family.id)}
            onclick={() => toggleFamily(family)}
          >
            <ChevronRightIcon
              class="size-3.5 shrink-0 text-muted-foreground {isFamilyOpen(
                family,
              )
                ? 'rotate-90'
                : ''}"
              aria-hidden="true"
            />
            <span>{family.label}</span>
            <span class="ml-auto text-xs font-normal text-muted-foreground"
              >{family.models.length}
              {family.models.length === 1 ? "variant" : "variants"}</span
            >
          </button>
        {/if}
        <div id={familyID(family.id)} hidden={!isFamilyOpen(family)}>
          {#each family.models as model (model.id)}
            {@const speechModel = isSpeechRuntimeModel(model)}
            {@const selected =
              model.id ===
              (speechModel ? row?.instance.speechModel : row?.instance.model)}
            {@const loaded = selected && running}
            {@const source =
              entry.models?.find((qualified) => qualified.id === model.id)
                ?.source ?? model.source}
            {@const downloading =
              row?.status.operation?.kind === "download" &&
              row.status.operation.model === model.id}
            {@const detailsOpen = expandedModels[model.id] ?? false}
            {@const variant = runtimeVariantLabel(entry.id, model.id)}
            <article aria-label={model.name} class="border-b border-hairline">
              <div class="model-row {selected ? 'bg-accent-wash' : ''}">
                <button
                  type="button"
                  class="model-identity"
                  aria-label={`Model details: ${model.name}`}
                  title={`${model.name} (${model.id})`}
                  aria-describedby={`${modelID(model.id)}-description`}
                  aria-expanded={detailsOpen}
                  aria-controls={modelID(model.id)}
                  onclick={() => {
                    expandedModels[model.id] = !detailsOpen;
                  }}
                >
                  <ChevronRightIcon
                    class="size-3.5 shrink-0 text-muted-foreground {detailsOpen
                      ? 'rotate-90'
                      : ''}"
                    aria-hidden="true"
                  />
                  <span class="min-w-0 flex-1">
                    <span
                      class="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5"
                    >
                      <span class="content-value min-w-0 break-words"
                        >{model.name}</span
                      >
                      {#if model.recommended}<span
                          class="shrink-0 text-[10px] font-medium text-muted-foreground"
                          >Recommended</span
                        >{/if}
                    </span>
                    <span
                      id={`${modelID(model.id)}-description`}
                      class="mt-0.5 block break-words text-xs font-normal leading-relaxed text-muted-foreground"
                    >
                      {model.description || variant || task(model)}
                    </span>
                  </span>
                </button>
                <span
                  class="model-size text-right text-xs text-secondary-foreground"
                >
                  <span class="block text-muted-foreground">{task(model)}</span>
                  <span class="font-mono" title={modelSize(model.sizeBytes)}
                    >{model.sizeBytes ? modelSize(model.sizeBytes) : "—"}</span
                  >
                </span>
                <span
                  class="model-status text-xs {loaded
                    ? 'text-success'
                    : 'text-ink-quiet'}"
                >
                  {loaded
                    ? "Loaded"
                    : selected
                      ? "Selected"
                      : model.installed
                        ? "Downloaded"
                        : "Not installed"}
                </span>
                <span class="model-actions flex items-center justify-end gap-1">
                  {#if downloading && busy}
                    <Button
                      variant="outline"
                      size="xs"
                      disabled={cancelling}
                      onclick={onCancel}>Cancel</Button
                    >
                  {:else}
                    {#if !model.installed}<Button
                        variant="outline"
                        size="xs"
                        disabled={controlsLocked}
                        title={!installed
                          ? "Install this runtime first"
                          : undefined}
                        onclick={() => onGet(model)}
                        ><DownloadIcon
                          class="size-3"
                          aria-hidden="true"
                        />Get</Button
                      >{/if}
                    {#if !selected}<Button
                        variant="outline"
                        size="xs"
                        disabled={controlsLocked}
                        title={!installed
                          ? "Install this runtime first"
                          : undefined}
                        onclick={() => onSelect(model)}>Select</Button
                      >{/if}
                    {#if selected && speechModel}<Button
                        variant="outline"
                        size="xs"
                        disabled={controlsLocked}
                        title="Deselect this runtime in Text to speech first. Disabling keeps downloaded files."
                        onclick={onDisableSpeech}>Disable</Button
                      >{/if}
                    {#if model.installed}<Button
                        variant="ghost"
                        size="xs"
                        class="size-6 p-0"
                        disabled={controlsLocked}
                        aria-label={`Delete ${model.name}`}
                        title="Remove downloaded files"
                        onclick={() => onRemove(model)}
                        ><Trash2Icon
                          class="size-3.5"
                          aria-hidden="true"
                        /></Button
                      >{/if}
                  {/if}
                </span>
              </div>
              {#if downloading}<div
                  class="space-y-1 px-2 py-2 text-xs text-secondary-foreground"
                  role="status"
                >
                  {#if busy}
                    {#if view.percent !== null}<progress
                        class="h-1.5 w-full accent-primary"
                        max="100"
                        value={view.percent}
                        aria-label={`${model.name} download progress`}
                      ></progress>{/if}
                    <p>
                      {view.activity}{view.transferred
                        ? ` · ${view.transferred}`
                        : ""}
                    </p>
                  {:else if view.completion}<p>{view.completion}</p>{/if}
                </div>{/if}
              <div
                id={modelID(model.id)}
                hidden={!detailsOpen}
                class="space-y-2 border-t border-hairline bg-well py-2 pl-6 pr-2 text-xs text-muted-foreground"
              >
                {#if detailsOpen}
                  <p>
                    Model ID: <span
                      class="break-all font-mono text-secondary-foreground"
                      >{model.id}</span
                    >
                  </p>
                  {#if source}<ModelDownloadSource {source} />{/if}
                {/if}
              </div>
            </article>
          {/each}
        </div>
      {/each}
    </div>
  {:else}
    <p class="py-6 text-center text-[13px] text-muted-foreground" role="status">
      {query
        ? "No models match your filter."
        : "No qualified models in this runtime's catalog yet."}
    </p>
  {/each}
</div>

<style>
  .model-catalog {
    container-type: inline-size;
  }
  .model-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 56px 82px 104px;
    min-height: 44px;
    align-items: center;
    gap: 8px;
  }
  .model-identity {
    display: flex;
    min-width: 0;
    min-height: 44px;
    align-items: center;
    gap: 6px;
    padding: 4px 2px;
    text-align: left;
  }
  .model-identity:hover {
    background: var(--subtle-fill-hover);
  }
  .model-identity:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: -2px;
  }
  @container (max-width: 620px) {
    .model-row {
      grid-template-columns: minmax(0, 1fr) auto 104px;
      row-gap: 2px;
      padding-block: 4px;
    }
    .model-identity {
      grid-column: 1 / 3;
    }
    .model-actions {
      grid-column: 3;
      grid-row: 1 / 3;
    }
    .model-size {
      display: flex;
      gap: 8px;
      padding-left: 22px;
      text-align: left;
    }
    .model-status {
      text-align: right;
    }
  }
  @container (max-width: 400px) {
    .model-row {
      grid-template-columns: minmax(0, 1fr) auto;
      row-gap: 4px;
      padding-block: 6px;
    }
    .model-identity {
      grid-column: 1 / -1;
    }
    .model-actions {
      grid-column: 2;
      grid-row: 2 / 4;
      max-width: 104px;
      flex-wrap: wrap;
    }
    .model-size {
      grid-column: 1;
      grid-row: 2;
      flex-wrap: wrap;
    }
    .model-status {
      grid-column: 1;
      grid-row: 3;
      padding-left: 22px;
      text-align: left;
    }
  }
</style>
