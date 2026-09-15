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

// A tab can remount while its previous reader is still closing. Coordinate
// controls by backend target so an old disable cannot overtake the new enable.
const controls = new WeakMap<
  ProcessOutputService,
  Map<string, Promise<void>>
>();

export class ProcessOutputState {
  instanceID = $state("");
  accepted = $state(false);
  chunks = $state<OutputChunk[]>([]);
  cursor = $state(0);
  error = $state("");
  busy = $state(false);
  visible = $state(true);
  truncated = $state(false);
  // Parser/viewport epoch, including resets coalesced into a single UI update.
  revision = $state(0);
  #generation = 0;
  #reading = false;
  #disposed = false;
  constructor(private readonly service: ProcessOutputService) {}

  #control(instanceID: string, action: () => Promise<void>) {
    let targets = controls.get(this.service);
    if (!targets) {
      targets = new Map();
      controls.set(this.service, targets);
    }
    const pending = (targets.get(instanceID) ?? Promise.resolve()).then(action);
    const settled = pending.catch(() => {});
    targets.set(instanceID, settled);
    void settled.then(() => {
      if (targets.get(instanceID) === settled) targets.delete(instanceID);
    });
    return pending;
  }

  select(instanceID: string) {
    const previous = this.instanceID;
    this.#generation++;
    this.revision++;
    this.instanceID = instanceID;
    this.accepted = false;
    this.busy = false;
    this.chunks = [];
    this.cursor = 0;
    this.error = "";
    this.truncated = false;
    return previous
      ? this.#control(previous, () =>
          this.service.DisableProcessOutput({ instanceID: previous }),
        ).catch(() => {})
      : Promise.resolve();
  }
  /** Opening an embedded output tab starts its bounded reader immediately. */
  async open(instanceID: string) {
    if (this.#disposed) return;
    const selection = this.select(instanceID);
    const generation = this.#generation;
    await selection;
    if (
      !instanceID ||
      !this.visible ||
      generation !== this.#generation ||
      this.#disposed
    )
      return;
    await this.show();
    if (generation === this.#generation) await this.poll();
  }
  async show() {
    if (!this.instanceID || this.accepted || this.busy || this.#disposed)
      return;
    const generation = this.#generation;
    const instanceID = this.instanceID;
    this.busy = true;
    this.error = "";
    try {
      await this.#control(instanceID, async () => {
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
    this.revision++;
    this.chunks = [];
    this.cursor = 0;
    this.truncated = false;
    this.busy = true;
    this.error = "";
    try {
      const instanceID = this.instanceID;
      await this.#control(instanceID, () =>
        this.service.ClearProcessOutput({ instanceID }),
      );
    } catch {
      if (generation === this.#generation) {
        this.error =
          "Could not clear process output. Close and reopen to try again.";
        this.accepted = false;
        const instanceID = this.instanceID;
        void this.#control(instanceID, () =>
          this.service.DisableProcessOutput({ instanceID }),
        ).catch(() => {});
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
        void this.select(this.instanceID);
        this.error = "Output reading stopped. Reopen the reader to continue.";
        return;
      }
      if (snapshot.truncated) this.revision++;
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
