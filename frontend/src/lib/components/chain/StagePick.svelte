<script lang="ts">
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";

  let {
    value,
    meta = "",
    label,
    onOpen,
    disabled = false,
  }: {
    /** What this stage is currently set to — the one fact the card leads with. */
    value: string;
    /** The literal underneath: an endpoint, a runtime, a file. Monospaced. */
    meta?: string;
    /** Accessible name, since the value alone does not say what it configures. */
    label: string;
    onOpen: () => void;
    disabled?: boolean;
  } = $props();
</script>

<!-- Each stage owns its own setting. Opening it goes to the one place that
     configures this step, rather than to a settings window's table of
     contents. -->
<div class="flex flex-col gap-0.5">
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
  {#if meta}
    <p class="truncate px-1 font-mono text-[10px] text-ink-quiet" title={meta}>
      {meta}
    </p>
  {/if}
</div>
