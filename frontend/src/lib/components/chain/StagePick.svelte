<script lang="ts">
  import type { Snippet } from "svelte";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import * as Popover from "$lib/components/ui/popover";

  let {
    value,
    meta = "",
    label,
    onOpen,
    panel,
    panelWidth = "w-[420px]",
    disabled = false,
  }: {
    /** What this stage is currently set to — the one fact the card leads with. */
    value: string;
    /** The literal underneath: an endpoint, a runtime, a file. Monospaced. */
    meta?: string;
    /** Accessible name, since the value alone does not say what it configures. */
    label: string;
    /** Used when the stage has no inline panel and must navigate instead. */
    onOpen?: () => void;
    /**
     * The stage's own settings. A stage that carries one configures itself in
     * place: that is what takes connection, model and cleanup off the settings
     * nav rather than merely linking to them from here.
     */
    panel?: Snippet;
    panelWidth?: string;
    disabled?: boolean;
  } = $props();

  let open = $state(false);
</script>

<div class="flex flex-col gap-0.5">
  {#if panel}
    <Popover.Root bind:open>
      <Popover.Trigger
        {disabled}
        aria-label={`${label}: ${value}`}
        class="flex w-full items-center justify-between gap-2 rounded-md px-1 py-0.5 text-left transition-colors hover:bg-subtle-fill-hover aria-expanded:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-60"
      >
        <span class="truncate text-[13px] font-medium">{value}</span>
        <ChevronDownIcon
          class="size-3.5 shrink-0 text-muted-foreground"
          aria-hidden="true"
        />
      </Popover.Trigger>
      <Popover.Content
        align="start"
        role="dialog"
        aria-label={label}
        class="max-h-[420px] overflow-y-auto {panelWidth} p-0"
      >
        {@render panel()}
      </Popover.Content>
    </Popover.Root>
  {:else}
    <button
      type="button"
      class="flex w-full items-center justify-between gap-2 rounded-md px-1 py-0.5 text-left transition-colors hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-60"
      {disabled}
      onclick={onOpen}
      aria-label={`${label}: ${value}`}
    >
      <span class="truncate text-[13px] font-medium">{value}</span>
      <ChevronDownIcon
        class="size-3.5 shrink-0 text-muted-foreground"
        aria-hidden="true"
      />
    </button>
  {/if}
  {#if meta}
    <p class="truncate px-1 font-mono text-[10px] text-ink-quiet" title={meta}>
      {meta}
    </p>
  {/if}
</div>
