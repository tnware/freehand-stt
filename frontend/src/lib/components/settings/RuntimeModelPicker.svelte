<script lang="ts">
  import * as Picker from "$lib/components/ui/combobox";
  import FieldHelp from "./FieldHelp.svelte";
  import { modelSources } from "$lib/utils/modelSources";
  import { Combobox } from "bits-ui";
  import PickerRefreshButton from "./PickerRefreshButton.svelte";
  import * as Menu from "$lib/components/ui/dropdown-menu";
  import CheckIcon from "@lucide/svelte/icons/check";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  import EraserIcon from "@lucide/svelte/icons/eraser";
  let {
    id,
    value,
    models = [],
    savedModels = [],
    draftModels = [],
    serverLoaded = false,
    busy = false,
    onDiscover,
    onEnter,
    metadataStatus = "idle",
    onChoose,
    onForget,
    compact = false,
    sidebar = false,
    immediate = false,
    disabled = false,
    profileName = "",
    showProfileName = true,
  }: {
    id: string;
    value: string;
    models?: string[];
    savedModels?: string[];
    draftModels?: string[];
    serverLoaded?: boolean;
    busy?: boolean;
    onDiscover: () => void;
    onEnter?: () => void;
    metadataStatus?: "idle" | "loading" | "ready" | "empty" | "failed";
    onChoose: (model: string) => boolean | Promise<boolean>;
    disabled?: boolean;
    profileName?: string;
    showProfileName?: boolean;
    compact?: boolean;
    sidebar?: boolean;
    immediate?: boolean;
    onForget?: () => void;
  } = $props();
  const help = $derived(
    serverLoaded
      ? "Options apply to the model loaded by this server."
      : immediate
        ? "Each model keeps its own options. Changes apply immediately."
        : "Each model keeps its own options. Save applies all your edits.",
  );
  let open = $state(false),
    query = $state("");
  const allModels = $derived([
    ...new Set([
      ...draftModels,
      ...savedModels,
      ...models,
      ...(value ? [value] : []),
    ]),
  ]);
  const matches = $derived(
    allModels.filter((model) =>
      model.toLowerCase().includes(query.trim().toLowerCase()),
    ),
  );
  const custom = $derived(
    query.trim() && !allModels.includes(query.trim()) ? query.trim() : "",
  );
  const choices = $derived(
    [...matches, ...(custom ? [custom] : [])].map((model) => ({
      value: model,
      label: model,
    })),
  );
  let choosing = $state(false);
  const locked = $derived(disabled || choosing);
  const hasStatus = $derived(
    busy || ["loading", "failed", "empty"].includes(metadataStatus),
  );
  async function choose(model: string) {
    if (!model || locked) return;
    choosing = true;
    try {
      if (await onChoose(model)) {
        open = false;
        query = "";
      }
    } finally {
      choosing = false;
    }
  }
</script>

<div class={compact ? "space-y-2" : "space-y-2 px-5 py-4"}>
  <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-2">
    <div class="flex items-center gap-1">
      <label for={id} class="content-value">Model</label><FieldHelp
        label="About model settings"
        text={help}
      />
    </div>
    <div class="flex items-center gap-1">
      <PickerRefreshButton
        label={serverLoaded ? "Check server" : "Refresh models"}
        {sidebar}
        busy={busy || metadataStatus === "loading"}
        disabled={locked}
        onclick={onDiscover}
      />
      {#if onForget && savedModels.includes(value) && !serverLoaded}<Menu.Root
          ><Menu.Trigger
            aria-label="Model actions"
            class="inline-flex size-7 items-center justify-center rounded-sm text-muted-foreground hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring"
            disabled={locked}><EllipsisIcon class="size-4" /></Menu.Trigger
          ><Menu.Content align="end" class="w-60 max-w-[calc(100vw-24px)]"
            ><Menu.Item onclick={onForget} class="gap-2.5 px-3 py-2.5">
              <EraserIcon class="size-4 text-muted-foreground" />
              <span>Forget saved settings</span>
            </Menu.Item></Menu.Content
          ></Menu.Root
        >{/if}
    </div>
  </div>
  {#if serverLoaded}<p {id} class="text-sm text-muted-foreground">
      Server-loaded model
    </p>
  {:else}<Combobox.Root
      type="single"
      bind:value={() => value, (next) => void choose(next)}
      inputValue={open ? query : value}
      bind:open
      items={choices}
      onOpenChange={(next) => {
        if (next) onEnter?.();
        else query = "";
      }}
      allowDeselect={false}
      disabled={locked}
    >
      <div class="relative">
        <Combobox.Input
          data-slot="combobox-input"
          {id}
          aria-label="Choose model"
          aria-describedby={`${id}-help${hasStatus ? ` ${id}-status` : ""}`}
          placeholder="Search or enter a model ID…"
          class="h-8 w-full min-w-0 rounded-md border border-input bg-well px-3 pr-9 font-mono text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
          spellcheck={false}
          onclick={() => {
            if (!open) {
              open = true;
              onEnter?.();
            }
          }}
          oninput={(e) => {
            query = e.currentTarget.value;
            open = true;
          }}
          onkeydown={(e) => {
            if (
              e.key === "Enter" &&
              query.trim() &&
              !e.currentTarget.getAttribute("aria-activedescendant")
            ) {
              e.preventDefault();
              choose(query.trim());
            }
          }}
        >
          {#snippet child({ props })}<input
              {...props}
              value={open ? query : value}
            />{/snippet}
        </Combobox.Input>
        <Picker.Trigger aria-label="Show models" />
      </div>
      <Picker.Content>
        {#key query}{#each choices as choice (choice.value)}
            <Picker.Item value={choice.value} label={choice.label}>
              <span class="min-w-0 flex-1 break-all font-mono"
                >{custom === choice.value
                  ? `Use “${choice.value}”`
                  : choice.value}</span
              >
              <span class="shrink-0 text-xs text-muted-foreground">
                {modelSources(
                  choice.value,
                  models,
                  savedModels,
                  draftModels,
                )}</span
              >
              {#if value === choice.value}<CheckIcon
                  class="size-3.5 shrink-0"
                />{/if}
            </Picker.Item>
          {:else}<p class="px-3 py-3 text-xs text-muted-foreground">
              Enter a model ID, or refresh the server’s model list.
            </p>{/each}{/key}
      </Picker.Content>
    </Combobox.Root>{/if}
  {#if busy || metadataStatus === "loading"}
    <p id={`${id}-status`} role="status" class="text-xs text-muted-foreground">
      Loading model list…
    </p>
  {:else if metadataStatus === "failed"}
    <p id={`${id}-status`} role="status" class="text-xs text-warning">
      Could not load the model list. Refresh or reopen to retry; you can still
      enter a model ID.
    </p>
  {:else if metadataStatus === "empty"}
    <p id={`${id}-status`} role="status" class="text-xs text-muted-foreground">
      No model IDs were reported. You can enter a model ID manually.
    </p>
  {/if}
  {#if !sidebar && ((showProfileName && profileName) || (value && !serverLoaded))}
    <p
      class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground"
    >
      {#if showProfileName && profileName}<span
          >Profile <span class="font-medium text-foreground">{profileName}</span
          ></span
        >{/if}
      {#if value && !serverLoaded}<span
          >{modelSources(value, models, savedModels, draftModels)}</span
        >{/if}
    </p>
  {/if}
  <p id={`${id}-help`} class="sr-only">{help}</p>
</div>
