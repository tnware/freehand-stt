import { State, type Status } from "$lib/state";

type ClockStatus = Pick<Status, "state" | "generation" | "startedAt">;

/** Recording duration only; never interpret Go's zero time as a recording start. */
export class CaptureClock {
  seconds = $state(0);
  #generation: number | undefined;
  #start: number | null = null;
  #recording = false;

  update(status: ClockStatus, now: number) {
    if (status.generation !== this.#generation || status.state === State.Idle) {
      this.#generation = status.generation;
      this.#start = null;
      this.#recording = false;
      this.seconds = 0;
    }
    if (status.state === State.Recording) {
      const start = Date.parse(status.startedAt ?? "");
      if (Number.isFinite(start) && start > 0 && start <= now) this.#start = start;
      this.#recording = true;
    } else if (!this.#recording) {
      return;
    } else {
      // Transcribing/failure snapshots may omit StartedAt or serialize year 1.
      // Freeze using the last valid capture timestamp from this generation.
      this.#recording = false;
    }
    if (this.#start !== null && Number.isFinite(now)) {
      this.seconds = Math.max(0, Math.floor((now - this.#start) / 1000));
    }
  }
}
