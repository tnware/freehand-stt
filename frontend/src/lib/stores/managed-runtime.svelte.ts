import type {
  Instance,
  InstanceStatus,
  ProviderDescriptor,
} from "$bindings/managedruntime";
import type * as Manager from "$bindings/managedruntime/manager";

export type ManagedRuntimeService = {
  [
    K in
      | "GetInstances"
      | "GetProviders"
      | "SetInstance"
      | "DeleteInstance"
      | "Install"
      | "Remove"
      | "Start"
      | "Stop"
      | "Cancel"
      | "RefreshCatalog"
      | "DownloadModel"
      | "RemoveModel"
  ]: (
    ...args: Parameters<(typeof Manager)[K]>
  ) => Promise<Awaited<ReturnType<(typeof Manager)[K]>>>;
};
type Operation = "Install" | "Remove" | "Start" | "Stop" | "RefreshCatalog";

export class ManagedRuntimeState {
  instances = $state<InstanceStatus[]>([]);
  providers = $state<ProviderDescriptor[]>([]);
  loading = $state(false);
  error = $state("");
  #pending = $state<Record<string, string>>({});
  #errors = $state<Record<string, string>>({});
  #retries = $state<Record<string, (() => Promise<boolean>) | undefined>>({});
  #revision = 0;
  #events = new Map<string, number>();
  #deleted = new Set<string>();
  #loadID = 0;
  #disposed = false;
  constructor(
    private readonly service?: ManagedRuntimeService,
    private readonly canMutate: () => boolean = () => true,
    private readonly preferencesChanged: () => unknown = () => {},
  ) {}
  statusFor(id: string) {
    return this.instances.find((row) => row.instance.id === id);
  }
  pendingFor(id: string) {
    return this.#pending[id] ?? "";
  }
  errorFor(id: string) {
    return this.#errors[id] ?? "";
  }
  isBusy(id: string) {
    const status = this.statusFor(id)?.status;
    return (
      !!this.pendingFor(id) ||
      !!status?.phase ||
      ["installing", "starting", "stopping"].includes(status?.state ?? "")
    );
  }
  get busy() {
    return (
      Object.values(this.#pending).some(Boolean) ||
      this.instances.some((row) => this.isBusy(row.instance.id))
    );
  }
  canRetry(id: string) {
    return !!this.#retries[id];
  }
  applyStatus(row: InstanceStatus) {
    const id = row.instance.id;
    if (this.#disposed || !id || this.#deleted.has(id)) return;
    this.#events.set(id, ++this.#revision);
    this.instances = [
      ...this.instances.filter((item) => item.instance.id !== id),
      row,
    ];
  }
  async load(target?: string): Promise<boolean> {
    if (!this.service || this.#disposed) return false;
    const revision = this.#revision,
      request = ++this.#loadID;
    this.loading = true;
    try {
      const [rows, providers] = await Promise.all([
        this.service.GetInstances(),
        this.service.GetProviders(),
      ]);
      if (this.#disposed || request !== this.#loadID) return false;
      const ids = new Set((rows ?? []).map((row) => row.instance.id));
      const merged = (rows ?? []).map((row) => {
        const id = row.instance.id,
          current = this.statusFor(id);
        this.#deleted.delete(id);
        return current &&
          ((this.#events.get(id) ?? 0) > revision || (target && target !== id))
          ? current
          : row;
      });
      for (const current of this.instances) {
        const id = current.instance.id;
        if (!ids.has(id)) {
          if ((this.#events.get(id) ?? 0) > revision) merged.push(current);
          else this.#deleted.add(id);
        }
      }
      this.instances = merged;
      this.providers = providers ?? [];
      this.error = "";
      return true;
    } catch {
      if (!this.#disposed && request === this.#loadID)
        this.error = "Could not read runtime inventory. Refresh to try again.";
      return false;
    } finally {
      if (!this.#disposed && request === this.#loadID) this.loading = false;
    }
  }
  async #perform(
    id: string,
    label: string,
    action: () => Promise<void>,
    inventory = false,
    cancel = false,
  ): Promise<boolean> {
    if (
      this.#disposed ||
      !this.service ||
      (!cancel && this.isBusy(id)) ||
      (cancel && this.pendingFor(id) === "Cancelling")
    )
      return false;
    if (!cancel && !this.canMutate()) {
      this.#errors[id] =
        "Save or discard your settings edits before changing this runtime.";
      return false;
    }
    if (!inventory && !this.statusFor(id)?.status.supported) return false;
    this.#pending[id] = label;
    this.#errors[id] = "";
    // Destructive retries require a fresh UI confirmation instead of an opaque retry.
    const retryable = !["Remove", "Remove model", "Delete instance"].includes(
      label,
    );
    this.#retries[id] = undefined;
    try {
      await action();
      if (this.#disposed) return false;
      if (inventory) await this.preferencesChanged();
      return await this.load(inventory ? undefined : id);
    } catch {
      if (!this.#disposed) {
        this.#errors[id] =
          label === "Delete instance"
            ? "Could not delete runtime. Remove or reassign its Connections, finish active work, and try again."
            : "The runtime operation did not finish. Review its status and try again.";
        if (retryable)
          this.#retries[id] = () =>
            this.#perform(id, label, action, inventory, cancel);
        await this.load(id);
      }
      return false;
    } finally {
      if (!this.#disposed && this.#pending[id] === label)
        this.#pending[id] = "";
    }
  }
  run(id: string, operation: Operation) {
    return this.#perform(id, operation, () =>
      this.service![operation]({ instanceID: id }),
    );
  }
  downloadModel(id: string, model: string) {
    return this.#perform(id, "Download model", () =>
      this.service!.DownloadModel({ instanceID: id, model }),
    );
  }
  removeModel(id: string, model: string) {
    return this.#perform(id, "Remove model", () =>
      this.service!.RemoveModel({ instanceID: id, model }),
    );
  }
  cancel(id: string) {
    return this.#perform(
      id,
      "Cancelling",
      () => this.service!.Cancel({ instanceID: id }),
      false,
      true,
    );
  }
  saveInstance(instance: Instance) {
    return this.#perform(
      instance.id,
      "Save instance",
      () => this.service!.SetInstance(instance),
      true,
    );
  }
  deleteInstance(id: string) {
    return this.#perform(
      id,
      "Delete instance",
      async () => {
        await this.service!.DeleteInstance({ instanceID: id });
        if (!this.#disposed) {
          this.#deleted.add(id);
          this.#events.set(id, ++this.#revision);
          this.instances = this.instances.filter(
            (row) => row.instance.id !== id,
          );
        }
      },
      true,
    );
  }
  retry(id: string) {
    return this.#retries[id]?.() ?? Promise.resolve(false);
  }
  dispose() {
    this.#disposed = true;
    ++this.#loadID;
    this.#retries = {};
  }
}
