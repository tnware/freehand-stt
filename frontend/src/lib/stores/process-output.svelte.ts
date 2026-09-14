import type * as Manager from "$bindings/managedruntime/manager";
import type { OutputChunk } from "$bindings/managedruntime/models";

type Methods = Pick<
  typeof Manager,
  | "EnableProcessOutput"
  | "DisableProcessOutput"
  | "ReadProcessOutput"
  | "ClearProcessOutput"
>;
export type ProcessOutputService = {
  [K in keyof Methods]: (
    ...args: Parameters<Methods[K]>
  ) => Promise<Awaited<ReturnType<Methods[K]>>>;
};

export class ProcessOutputState {
  instanceID = $state("");
  accepted = $state(false);
  chunks = $state<OutputChunk[]>([]);
  cursor = $state(0);
  error = $state("");
  busy = $state(false);
  visible = $state(true);
  truncated = $state(false);
  #generation = 0;
  #reading = false;
  #disposed = false;
  #controls: Promise<void> = Promise.resolve();
  constructor(private readonly service: ProcessOutputService) {}

  #control(action: () => Promise<void>) {
    const pending = this.#controls.then(action);
    this.#controls = pending.catch(() => {});
    return pending;
  }

  select(instanceID: string) {
    const previous = this.instanceID;
    this.#generation++;
    this.instanceID = instanceID;
    this.accepted = false;
    this.busy = false;
    this.chunks = [];
    this.cursor = 0;
    this.error = "";
    this.truncated = false;
    return previous
      ? this.#control(() =>
          this.service.DisableProcessOutput({ instanceID: previous }),
        ).catch(() => {})
      : Promise.resolve();
  }
  async show() {
    if (!this.instanceID || this.accepted || this.busy || this.#disposed)
      return;
    const generation = this.#generation;
    const instanceID = this.instanceID;
    this.busy = true;
    this.error = "";
    try {
      await this.#control(async () => {
        if (generation !== this.#generation || this.#disposed) return;
        await this.service.EnableProcessOutput({ instanceID });
        if (generation === this.#generation && !this.#disposed)
          this.accepted = true;
        else await this.service.DisableProcessOutput({ instanceID });
      });
    } catch {
      if (generation === this.#generation)
        this.error = "Could not enable process output.";
    } finally {
      if (generation === this.#generation) this.busy = false;
    }
  }
  async clear() {
    if (!this.accepted || this.busy || this.#disposed) return;
    const generation = ++this.#generation;
    this.chunks = [];
    this.cursor = 0;
    this.truncated = false;
    this.busy = true;
    this.error = "";
    try {
      const instanceID = this.instanceID;
      await this.#control(() =>
        this.service.ClearProcessOutput({ instanceID }),
      );
    } catch {
      if (generation === this.#generation) {
        this.error =
          "Could not clear process output. Close and reopen to try again.";
        this.accepted = false;
        void this.service
          .DisableProcessOutput({ instanceID: this.instanceID })
          .catch(() => {});
      }
    } finally {
      if (generation === this.#generation) this.busy = false;
    }
  }
  dispose() {
    this.select("");
    this.#disposed = true;
  }
  async poll() {
    if (
      !this.accepted ||
      !this.visible ||
      this.busy ||
      this.#reading ||
      this.#disposed
    )
      return;
    const generation = this.#generation;
    this.#reading = true;
    try {
      const snapshot = await this.service.ReadProcessOutput({
        instanceID: this.instanceID,
        after: this.cursor,
      });
      if (generation !== this.#generation || this.#disposed) return;
      if (!snapshot.enabled) {
        this.select("");
        return;
      }
      const chunks = [
        ...(snapshot.truncated ? [] : this.chunks),
        ...(snapshot.chunks ?? []).filter(
          (chunk) => chunk.sequence > this.cursor,
        ),
      ];
      const encoder = new TextEncoder();
      let bytes = 0;
      let start = chunks.length;
      while (start > 0 && chunks.length - start < 1024) {
        const size = encoder.encode(chunks[start - 1].text).length;
        if (bytes + size > 256 * 1024) break;
        bytes += size;
        start--;
      }
      this.chunks = chunks.slice(start);
      this.truncated ||= start > 0;
      this.cursor = snapshot.next;
      this.truncated ||= snapshot.truncated;
      this.error = "";
    } catch {
      if (generation === this.#generation)
        this.error = "Could not read process output.";
    } finally {
      this.#reading = false;
    }
  }
}
