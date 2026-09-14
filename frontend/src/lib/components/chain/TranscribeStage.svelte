<script lang="ts">
  import type { Snippet } from "svelte";
  import Stage, { type StageTone } from "./Stage.svelte";
  import StagePick from "./StagePick.svelte";

  let {
    ordinal = "02",
    label = "Transcribe",
    model,
    source = "",
    tone = "idle",
    state = "",
    live = "",
    partial = "",
    notice = "",
    extra,
    control,
    onOpen,
    panel,
  }: {
    ordinal?: string;
    label?: string;
    model: string;
    /** Runtime or endpoint behind the model. */
    source?: string;
    tone?: StageTone;
    /** Endpoint health in a word: Reachable, Running, Not checked. */
    state?: string;
    /** Text already committed by the streaming recogniser. */
    live?: string;
    /** The unstable tail the recogniser may still revise. */
    partial?: string;
    /** A transcribe-mode fallback worth explaining, such as streaming being
     *  unavailable for the selected profile. */
    notice?: string;
    /** Extra footer content: the file workflow puts its streaming switch here. */
    extra?: Snippet;
    /** Replaces the state lamp, for a stage that owns a mode control. */
    control?: Snippet;
    onOpen: () => void;
    /** The stage's own settings, shown in place. */
    panel?: Snippet;
  } = $props();

  const streaming = $derived(Boolean(live || partial));
</script>

<Stage {ordinal} {label} {tone} {control}>
  <StagePick
    value={model || "Not selected"}
    meta={source}
    label="Transcription model"
    {onOpen}
    {panel}
  />

  {#if streaming}
    <!-- Live text belongs to the stage that produces it, so the chain shows
         the recogniser working rather than only its finished output. -->
    <p
      class="h-[62px] overflow-hidden rounded-md bg-well px-2.5 py-2 text-xs leading-relaxed"
      aria-live="polite"
    >
      <span class="text-secondary-foreground">{live}</span
      ><span class="text-accent-text">{partial}</span>
    </p>
  {/if}

  {#if notice}
    <p class="px-1 text-[11px] leading-snug text-muted-foreground">{notice}</p>
  {/if}

  {#snippet footer()}
    {#if state}
      <span
        class="rounded-sm border border-border px-1.5 py-0.5 text-[11px] text-muted-foreground"
        >{state}</span
      >
    {:else}
      <span></span>
    {/if}
    {#if extra}{@render extra()}{/if}
  {/snippet}
</Stage>
