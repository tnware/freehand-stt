<script lang="ts">
  import { Combobox } from "bits-ui";
  import { Button } from "$lib/components/ui/button";
  import * as Menu from "$lib/components/ui/dropdown-menu";
  import ChevronsUpDownIcon from "@lucide/svelte/icons/chevrons-up-down";
  import CheckIcon from "@lucide/svelte/icons/check";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  let {
    id,
    value,
    models = [],
    savedModels = [],
    draftModels = [],
    serverLoaded = false,
    busy = false,
    onDiscover,
    onChoose,
    onForget,
    compact = false,
    immediate = false,
  }: {
    id: string;
    value: string;
    models?: string[];
    savedModels?: string[];
    draftModels?: string[];
    serverLoaded?: boolean;
    busy?: boolean;
    onDiscover: () => void;
    onChoose: (model: string) => boolean | Promise<boolean>;
    compact?: boolean;
    immediate?: boolean;
    onForget?: () => void;
  } = $props();
  let open = $state(false),
    query = $state("");
  const allModels = $derived([
    ...new Set([...draftModels, ...savedModels, ...models, ...(value ? [value] : [])]),
  ]);
  const matches = $derived(
    allModels.filter((model) => model.toLowerCase().includes(query.trim().toLowerCase())),
  );
  const custom = $derived(query.trim() && !allModels.includes(query.trim()) ? query.trim() : "");
  const choices = $derived(
    [...matches, ...(custom ? [custom] : [])].map((model) => ({ value: model, label: model })),
  );
  async function choose(model: string) {
    if (model && (await onChoose(model))) {
      open = false;
      query = "";
    }
  }
</script>

<div class={compact ? "space-y-2" : "space-y-2 px-5 py-4"}>
  <div class="flex items-center justify-between gap-3">
    <label for={id} class="text-sm font-medium">Model</label>
    <div class="flex items-center gap-1">
      <Button variant="ghost" size="sm" disabled={busy} onclick={onDiscover}
        ><RefreshCwIcon class={busy ? "size-3.5 animate-spin" : "size-3.5"} />{serverLoaded
          ? "Check server"
          : "Refresh models"}</Button
      >
      {#if onForget && savedModels.includes(value) && !serverLoaded}<Menu.Root
          ><Menu.Trigger
            aria-label="Model actions"
            class="inline-flex size-8 items-center justify-center rounded-md text-muted-foreground hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            disabled={busy}><EllipsisIcon class="size-4" /></Menu.Trigger
          ><Menu.Content align="end"
            ><Menu.Item onclick={onForget}>Forget saved settings for this model</Menu.Item
            ></Menu.Content
          ></Menu.Root
        >{/if}
    </div>
  </div>
  {#if serverLoaded}<p {id} class="text-sm text-muted-foreground">Server-loaded model</p>
  {:else}<Combobox.Root
      type="single"
      {value}
      inputValue={open ? query : value}
      bind:open
      items={choices}
      onValueChange={choose}
      onOpenChange={(next) => {
        if (!next) query = "";
      }}
      allowDeselect={false}
      disabled={busy}
    >
      <div class="relative">
        <Combobox.Input
          {id}
          aria-label="Choose model"
          aria-describedby={`${id}-help`}
          placeholder="Search or enter a model ID…"
          class="h-10 w-full rounded-md border border-input bg-background px-3 pr-10 font-mono text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
          spellcheck={false}
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
          {#snippet child({ props })}<input {...props} value={open ? query : value} />{/snippet}
        </Combobox.Input>
        <Combobox.Trigger
          aria-label="Show models"
          class="absolute inset-y-0 right-0 flex w-10 items-center justify-center text-muted-foreground"
          ><ChevronsUpDownIcon class="size-4" /></Combobox.Trigger
        >
      </div>
      <Combobox.Portal
        ><Combobox.Content
          sideOffset={4}
          class="z-50 max-h-[min(20rem,var(--bits-combobox-content-available-height))] w-[var(--bits-combobox-anchor-width)] min-w-64 max-w-[calc(100vw-24px)] overflow-y-auto rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md"
        >
          {#key query}{#each choices as choice (choice.value)}
              <Combobox.Item
                value={choice.value}
                label={choice.label}
                class="flex cursor-default items-center gap-3 rounded-sm px-3 py-2.5 text-sm outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground"
              >
                <span class="min-w-0 flex-1 break-all font-mono"
                  >{custom === choice.value ? `Use “${choice.value}”` : choice.value}</span
                >
                <span class="shrink-0 text-[11px] text-muted-foreground"
                  >{draftModels.includes(choice.value)
                    ? "Edited"
                    : savedModels.includes(choice.value)
                      ? "Saved"
                      : custom === choice.value
                        ? "New"
                        : "Server"}</span
                >
                {#if value === choice.value}<CheckIcon class="size-3.5 shrink-0" />{/if}
              </Combobox.Item>
            {:else}<p class="px-3 py-3 text-xs text-muted-foreground">
                Enter a model ID, or refresh the server’s model list.
              </p>{/each}{/key}
        </Combobox.Content></Combobox.Portal
      >
    </Combobox.Root>{/if}
  <p
    id={`${id}-help`}
    class={compact ? "sr-only" : "text-xs leading-relaxed text-muted-foreground"}
  >
    {serverLoaded
      ? "Options apply to the model loaded by this server."
      : immediate
        ? "Each model keeps its own options. Changes apply immediately."
        : "Each model keeps its own options. Save applies all your edits."}
  </p>
</div>
