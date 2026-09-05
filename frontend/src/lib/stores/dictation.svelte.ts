import { RecordingMode, State, type Status } from "$lib/state";
import type { SessionMessages } from "./messages.svelte";
import type * as DictationBindings from "$bindings/dictation/service";
export type DictationStateService = Pick<
  typeof DictationBindings,
  | "CurrentStatus"
  | "StartRecording"
  | "StopRecording"
  | "Cancel"
  | "CopyPending"
>;

const IDLE: Status = {
  state: State.Idle,
  generation: 0,
  canCancel: false,
  canCopy: false,
};

/** Projects live dictation and issues its explicit commands. */
export class DictationState {
  readonly #service: DictationStateService;
  readonly #messages: SessionMessages;
  constructor(service: DictationStateService, messages: SessionMessages) {
    this.#service = service;
    this.#messages = messages;
  }
  status = $state<Status>(IDLE);
  #statusRevision = 0;

  applyStatus(status: Status) {
    this.#statusRevision++;
    this.status = status;
  }

  async toggleRecording() {
    this.#messages.clear();
    try {
      if (this.status.state === State.Recording)
        await this.#service.StopRecording();
      else await this.#service.StartRecording(RecordingMode.RecordingToggle);
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async cancel() {
    this.#messages.clear();
    try {
      await this.#service.Cancel();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async copyPending(): Promise<boolean> {
    this.#messages.clear();
    try {
      await this.#service.CopyPending();
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    }
  }

  async load() {
    const revision = this.#statusRevision;
    const status = await this.#service.CurrentStatus();
    if (revision === this.#statusRevision) this.status = status;
  }
}
