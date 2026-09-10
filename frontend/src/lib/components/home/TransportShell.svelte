<script lang="ts">
  import type { Snippet } from "svelte";
  import type { RailPhase } from "$lib/utils/status";

  let {
    control,
    stage,
    readout,
    summary,
    actionsVisible = true,
    rail = "hidden",
    railPercent,
    tall = false,
    stageGrid = false,
    busy = false,
    state,
  }: {
    /** The 116px cell: record, mode glyph, or outcome mark. */
    control: Snippet;
    /** The elastic middle. A meter while capturing, the text itself afterwards. */
    stage: Snippet;
    /** The 236px cell: the clock, the readouts, one action. */
    readout: Snippet;
    /** A compact clock/status group next to the recording control. */
    summary?: Snippet;
    actionsVisible?: boolean;
    rail?: RailPhase;
    /**
     * Determinate progress, 0-100. A file upload knows its own length, so it
     * gets a real percentage where a live request only gets the marching rail.
     */
    railPercent?: number;
    /** Text to speech needs room to compose; every other mode is one bar tall. */
    tall?: boolean;
    /** Measurement rules suit waveform/progress stages, but not text editors. */
    stageGrid?: boolean;
    busy?: boolean;
    /** Mirrored to a data attribute so a state can be inspected in the DOM. */
    state?: string;
  } = $props();
</script>

<!--
  File transport uses three columns. Voice supplies a summary to group the
  clock and status beside the record control. Each layout keeps a fixed height
  across recording, processing and recovery states.
-->
<section
  class="transport relative shrink-0 border-b border-hairline bg-background"
  class:tall
  class:recording-strip={!!summary}
  data-state={state}
  aria-busy={busy}
>
  <div class="cell control">
    {@render control()}
  </div>

  {#if summary}
    <div class="cell summary">{@render summary()}</div>
  {/if}

  <div class:stage-grid={stageGrid} class="cell stage">
    {@render stage()}
  </div>

  <div class="cell readout" class:actions-hidden={summary && !actionsVisible}>
    {@render readout()}
  </div>

  <div class="rail" data-phase={rail} data-determinate={railPercent !== undefined || undefined}>
    <span
      class="fill"
      style={railPercent === undefined
        ? undefined
        : `width: ${Math.max(0, Math.min(100, railPercent))}%`}
    ></span>
  </div>
</section>

<style>
  .transport {
    display: grid;
    grid-template-columns: 7.25rem minmax(0, 1fr) 14.75rem;
    grid-template-areas: "control stage readout";
    height: 8.25rem;
  }
  .transport.tall {
    height: 11.5rem;
  }

  .cell {
    display: flex;
    min-width: 0;
  }
  .control {
    grid-area: control;
    align-items: center;
    justify-content: center;
  }
  .stage {
    grid-area: stage;
    flex-direction: column;
    justify-content: center;
    padding: 0 1.125rem;
  }
  .stage-grid {
    /* Vertical rules behind the meter, so amplitude is read against something
       rather than floating in an empty band. */
    background-image: repeating-linear-gradient(
      to right,
      var(--hairline) 0 1px,
      transparent 1px 1.75rem
    );
  }
  .readout {
    grid-area: readout;
    flex-direction: column;
    justify-content: center;
    gap: 0.625rem;
    padding: 0 1.125rem;
  }

  /*
   * Below roughly 700px the three cells cannot all keep their share, and the
   * meter is what loses: a two-pixel-wide meter says nothing, and an outcome
   * stage squeezed to one word per line says less. So the readout drops to its
   * own full-width row and the bar grows downward instead. The meter halves in
   * height so that row stays compact rather than becoming a dead band, and the
   * clock stops being a headline once it shares a line. The cells keep both
   * their identity and their order, so a state change still moves nothing.
   */
  @container (max-width: 699px) {
    .transport,
    .transport.tall {
      grid-template-columns: 5.5rem minmax(0, 1fr);
      grid-template-areas:
        "control stage"
        "readout readout";
      height: auto;
      --meter-height: 2.375rem;
    }
    /* Voice and file controls contain different copy, but occupy one stable
       transport frame. Text to speech remains intentionally taller because
       its primary control is an editor rather than a one-line transport. */
    .transport:not(.tall) {
      grid-template-rows: minmax(0, 1fr) 3rem;
      height: 9rem;
    }
    .transport.tall {
      grid-template-rows: minmax(0, 1fr) 3.75rem;
      height: 13.25rem;
    }
    .control {
      padding: 0.75rem 0;
    }
    .stage {
      padding: 0.75rem 0.875rem;
    }
    .readout {
      min-height: 3rem;
      padding: 0 0.875rem;
      border-top: 1px solid var(--hairline);
      border-left: none;
    }
    /*
     * One bar tall: the clock, the state and the action share the row. Text to
     * speech keeps a column, because its readout is a block of endpoint facts
     * rather than three glance values.
     */
    .transport:not(.tall) .readout {
      flex-direction: row;
      align-items: center;
      justify-content: space-between;
      gap: 0.75rem;
    }
    .transport:not(.tall) .readout :global(.figure:first-child) {
      font-size: 1.5rem;
    }
    .transport.tall .readout {
      padding: 0.375rem 0.75rem;
    }
  }

  /* Voice groups the record control, clock and status as one unit. The
     waveform uses the remaining width and actions never change strip height. */
  .recording-strip {
    grid-template-columns: 3rem 7.5rem minmax(0, 1fr) auto;
    grid-template-areas: "control summary stage readout";
    align-items: center;
    column-gap: 1rem;
    height: 6.5rem;
    padding: 0 1.5rem;
    --meter-height: 2.25rem;
  }
  .recording-strip .control,
  .recording-strip .stage,
  .recording-strip .readout {
    padding: 0;
    border: 0;
    min-height: 0;
  }
  .summary {
    grid-area: summary;
    flex-direction: column;
    gap: 0.5rem;
  }
  .recording-strip .actions-hidden {
    display: none;
  }
  .recording-strip .readout {
    max-width: 12rem;
  }
  @container (max-width: 699px) {
    .recording-strip:not(.tall) {
      grid-template-columns: 3rem 6.25rem minmax(0, 1fr) auto;
      grid-template-areas: "control summary stage readout";
      grid-template-rows: minmax(0, 1fr);
      column-gap: 0.75rem;
      height: 6.5rem;
      padding: 0 1rem;
      --meter-height: 2rem;
    }
    .recording-strip .stage {
      align-self: center;
    }
  }

  .recording-strip .rail[data-phase="done"] {
    height: 1px;
    opacity: 0.55;
  }

  .rail {
    position: absolute;
    inset-inline: 0;
    bottom: 0;
    height: 2px;
    overflow: hidden;
    background-color: var(--hairline);
  }
  .rail[data-phase="hidden"] {
    background-color: transparent;
  }
  .fill {
    display: block;
    height: 100%;
    width: 0;
    background-color: var(--primary);
    transition:
      width 420ms ease,
      background-color 260ms ease;
  }
  /* Indeterminate: the endpoint does not report progress, so the rail says
     "in flight" rather than inventing a percentage. */
  .rail[data-phase="working"] .fill {
    width: 36%;
    animation: march 1.35s cubic-bezier(0.65, 0, 0.35, 1) infinite;
  }
  /* A measured upload reports its own share, so it must not also march. */
  .rail[data-determinate] .fill {
    animation: none;
  }
  @keyframes march {
    from {
      margin-left: -36%;
    }
    to {
      margin-left: 100%;
    }
  }
  .rail[data-phase="done"] .fill {
    width: 100%;
    background-color: var(--success);
  }
  .rail[data-phase="error"] .fill {
    width: 100%;
    background-color: var(--destructive);
  }

  @media (prefers-reduced-motion: reduce) {
    .fill {
      transition: none;
    }
    .rail[data-phase="working"]:not([data-determinate]) .fill {
      animation: none;
      width: 100%;
      opacity: 0.5;
    }
  }
</style>
