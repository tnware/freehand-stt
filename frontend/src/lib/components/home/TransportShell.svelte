<script lang="ts">
  import type { Snippet } from "svelte";
  import type { RailPhase } from "$lib/utils/status";

  let {
    control,
    stage,
    readout,
    rail = "hidden",
    railPercent,
    tall = false,
    stageGrid = true,
    busy = false,
    state,
  }: {
    /** The 116px cell: record, mode glyph, or outcome mark. */
    control: Snippet;
    /** The elastic middle. A meter while capturing, the text itself afterwards. */
    stage: Snippet;
    /** The 236px cell: the clock, the readouts, one action. */
    readout: Snippet;
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
  Every input mode and every dictation state shares this frame: a 116px control
  cell, an elastic stage, and a 236px readout cell. Because the geometry never
  changes, starting a recording, switching tabs or failing a request moves
  nothing on screen; only the contents of the three cells change.
-->
<section
  class="transport relative shrink-0 border-b border-card-stroke bg-card shadow-lift"
  class:tall
  data-state={state}
  aria-busy={busy}
>
  <div class="cell control">
    {@render control()}
  </div>

  <div class:stage-grid={stageGrid} class="cell stage">
    {@render stage()}
  </div>

  <div class="cell readout">
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
    border-right: 1px solid var(--hairline);
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
    border-left: 1px solid var(--hairline);
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
      grid-template-columns: 4.5rem minmax(0, 1fr);
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
