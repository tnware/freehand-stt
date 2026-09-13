import type { Status } from "$bindings/managedruntime";
import type * as Service from "$bindings/managedruntime/service";

/** Promise-returning generated operations, injectable without Wails promises in fixtures. */
export type ManagedRuntimeService = {
  [
    K in
      | "GetStatus"
      | "Install"
      | "Remove"
      | "Start"
      | "Stop"
      | "Cancel"
      | "RefreshCatalog"
      | "DownloadModel"
      | "RemoveModel"
      | "SetPreferences"
  ]: (
    ...args: Parameters<(typeof Service)[K]>
  ) => Promise<Awaited<ReturnType<(typeof Service)[K]>>>;
};

export class ManagedRuntimeState {
  status = $state<Status | null>(null);
  pending = $state("");
  loading = $state(false);
  error = $state("");
  #revision = 0;
  #disposed = false;
  #retry: (() => Promise<boolean>) | null = null;
  constructor(
    private readonly service?: ManagedRuntimeService,
    private readonly canMutate: () => boolean = () => true,
    private readonly preferencesChanged: () => unknown = () => {},
  ) {}
  get busy() {
    return (
      !!this.pending ||
      !!this.status?.phase ||
      ["installing", "starting", "stopping"].includes(this.status?.state ?? "")
    );
  }
  get canRetry() {
    return this.#retry !== null;
  }
  applyStatus(status: Status) {
    if (this.#disposed) return;
    this.#revision++;
    this.status = status;
  }
  async load() {
    if (!this.service || this.#disposed) return;
    const revision = this.#revision;
    this.loading = true;
    try {
      const status = await this.service.GetStatus();
      if (!this.#disposed && revision === this.#revision)
        this.applyStatus(status);
    } catch {
      if (!this.#disposed)
        this.error = "Could not read local runtime status. Try refreshing.";
    } finally {
      this.loading = false;
    }
  }
  async #perform(
    label: string,
    action: () => Promise<void>,
    preferences = false,
    cancel = false,
  ): Promise<boolean> {
    if (
      this.#disposed ||
      !this.service ||
      !this.status?.supported ||
      (!cancel && this.busy)
    )
      return false;
    if (!cancel && !this.canMutate()) {
      this.error =
        "Save or discard your settings edits before changing the local runtime.";
      return false;
    }
    this.pending = label;
    this.error = "";
    this.#retry = () => this.#perform(label, action, preferences, cancel);
    try {
      await action();
      await this.load();
      if (preferences && !this.#disposed) await this.preferencesChanged();
      return !this.error;
    } catch {
      if (!this.#disposed)
        this.error =
          "The local runtime operation did not finish. Retry, or review its status below.";
      await this.load();
      return false;
    } finally {
      this.pending = "";
    }
  }
  run(operation: "Install" | "Remove" | "Start" | "Stop" | "RefreshCatalog") {
    return this.#perform(
      operation,
      () => this.service![operation](),
      operation === "Remove",
    );
  }
  downloadModel(id: string) {
    return this.#perform("Download model", () =>
      this.service!.DownloadModel(id),
    );
  }
  removeModel(id: string) {
    return this.#perform("Remove model", () => this.service!.RemoveModel(id));
  }
  cancel() {
    return this.#perform(
      "Cancelling",
      () => this.service!.Cancel(),
      false,
      true,
    );
  }
  retry() {
    return this.#retry?.() ?? Promise.resolve(false);
  }
  enable() {
    return this.setPreferences({
      enabled: true,
      model: "nemotron-3.5",
      realtime: true,
    });
  }
  disable() {
    return this.setPreferences({
      enabled: false,
      model: this.status?.selectedModel ?? "nemotron-3.5",
      realtime: this.status?.realtime ?? true,
    });
  }
  setPreferences(
    preferences: Parameters<ManagedRuntimeService["SetPreferences"]>[0],
  ) {
    return this.#perform(
      "Applying preferences",
      () => this.service!.SetPreferences(preferences),
      true,
    );
  }
  useModel(id: string) {
    const model = this.status?.models?.find(
      (model) => model.id === id && model.installed,
    );
    if (!model || !this.status) return Promise.resolve(false);
    return this.setPreferences({
      enabled: this.status.enabled,
      model: id,
      realtime: model.realtime && this.status.realtime,
    });
  }
  dispose() {
    this.#disposed = true;
    this.#revision++;
    this.#retry = null;
  }
}
