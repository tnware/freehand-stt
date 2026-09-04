/**
 * Capture amplitude for the main screen meter.
 *
 * Go sends one number per tick and the history is kept here, because a whole
 * history per message would be dozens of numbers per tick to say what one
 * number already says. Each entry is one amplitude reading, so the meter is a
 * level histogram rather than the waveform itself.
 *
 * Sixty readings at the sender's 30 Hz is about two seconds of history.
 */
export class Levels {
  /** Oldest first, so the meter scrolls left to right. */
  history = $state<number[]>([]);

  readonly #size: number;

  constructor(size = 60) {
    this.#size = size;
    this.history = new Array<number>(size).fill(0);
  }

  get size(): number {
    return this.#size;
  }

  push(level: number) {
    const clamped = Math.min(1, Math.max(0, Number.isFinite(level) ? level : 0));
    this.history = [...this.history.slice(1), clamped];
  }

  reset() {
    this.history = new Array<number>(this.#size).fill(0);
  }
}

export const levels = new Levels();
