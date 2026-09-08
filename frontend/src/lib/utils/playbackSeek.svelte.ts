import type { SeekRequest } from "$bindings/tts";
import type { TTSStatus } from "$lib/state";

// Give the slider valid values before rendering: Bits otherwise rounds native
// progress updates through onValueChange, which would create a false drag draft.
export function playbackSteps(duration: number): number[] {
  const steps = Array.from({ length: Math.ceil(duration / 100) }, (_, i) => i * 100);
  return [...steps, duration];
}

export function playbackSliderPosition(position: number, duration: number): number {
  const value = Math.max(0, Math.min(position, duration));
  const regular = Math.min(Math.round(value / 100) * 100, Math.floor(duration / 100) * 100);
  return duration - value < Math.abs(regular - value) ? duration : regular;
}

/** Local drag preview; Go owns admission and the actual playback position. */
export class PlaybackSeek {
  draft = $state<SeekRequest | null>(null);
  pending = $state(false);
  position(status: TTSStatus) {
    return this.draft?.generation === status.generation && status.canSeek
      ? this.draft.positionMilliseconds
      : status.positionMilliseconds;
  }
  change(status: TTSStatus, position: number) {
    if (!status.canSeek || this.pending || !Number.isFinite(position)) return;
    this.draft = {
      generation: status.generation,
      positionMilliseconds: Math.round(
        Math.max(0, Math.min(position, status.durationMilliseconds)),
      ),
    };
  }
  async commit(status: TTSStatus, send: (request: SeekRequest) => Promise<void>) {
    const request = this.draft;
    if (this.pending) return;
    if (!request || request.generation !== status.generation || !status.canSeek) {
      this.draft = null;
      return;
    }
    this.pending = true;
    try {
      await send({ ...request });
    } finally {
      this.draft = null;
      this.pending = false;
    }
  }
}
