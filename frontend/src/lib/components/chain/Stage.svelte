<script lang="ts">
  import type { Snippet } from "svelte";

  export type StageTone = "idle" | "ok" | "busy" | "warn" | "bad" | "off";

  let {
    ordinal,
    label,
    tone = "idle",
    unused = false,
    control,
    children,
    footer,
  }: {
    /** Position in the chain. Carries the sequence when the window is too
     *  narrow for connectors, so it is content rather than decoration. */
    ordinal: string;
    label: string;
    tone?: StageTone;
    /** A stage this workflow does not use is dimmed in place, never removed:
     *  the four positions have to mean the same thing in every chain. */
    unused?: boolean;
    /** Replaces the state lamp when a stage owns a control, such as a switch. */
    control?: Snippet;
    children: Snippet;
    footer?: Snippet;
  } = $props();
</script>

<section
  class="stage flex min-w-0 flex-1 flex-col overflow-hidden rounded-lg border bg-card"
  class:unused
  data-tone={tone}
  aria-label={`Stage ${ordinal}: ${label}${unused ? " (not used in this workflow)" : ""}`}
>
  <header
    class="flex h-[30px] shrink-0 items-center justify-between gap-2 border-b border-hairline bg-subtle-fill-hover px-2.5"
  >
    <span class="flex min-w-0 items-center gap-2">
      <span class="figure shrink-0 font-mono text-[10px] text-ink-quiet"
        >{ordinal}</span
      >
      <h3
        class="truncate text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase"
      >
        {label}
      </h3>
    </span>
    {#if control}
      {@render control()}
    {:else}
      <span class="lamp size-[7px] shrink-0 rounded-full" aria-hidden="true"
      ></span>
    {/if}
  </header>

  <div class="flex min-h-0 flex-1 flex-col gap-2 p-3">
    {@render children()}
    {#if footer}
      <div class="flex-1"></div>
      <div class="flex items-center justify-between gap-2">
        {@render footer()}
      </div>
    {/if}
  </div>
</section>

<style>
  .stage {
    border-color: var(--hairline);
  }
  .stage.unused {
    background: transparent;
    border-style: dashed;
    border-color: var(--border);
  }
  .stage.unused :global(*) {
    color: var(--ink-disabled);
  }
  .lamp {
    background: var(--rest, var(--meter-rest));
  }
  .stage[data-tone="ok"] .lamp {
    background: var(--success);
  }
  .stage[data-tone="busy"] .lamp {
    background: var(--primary);
  }
  .stage[data-tone="warn"] .lamp {
    background: var(--warning);
  }
  .stage[data-tone="bad"] .lamp {
    background: var(--destructive);
  }
  .stage[data-tone="off"] .lamp {
    background: var(--meter-rest);
  }
  .stage[data-tone="warn"] {
    border-color: color-mix(in srgb, var(--warning) 30%, transparent);
  }
  .stage[data-tone="bad"] {
    border-color: color-mix(in srgb, var(--destructive) 34%, transparent);
  }
</style>
