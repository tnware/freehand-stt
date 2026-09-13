import type { PermissionStatus } from "$bindings/input";

export type PermissionKind = "microphone" | "accessibility" | "keyboard";
export const NATIVE_PERMISSION_SERVICES = Symbol("native-permission-services");
export interface NativePermissionServices {
  NativePermissions(): Promise<PermissionStatus>;
  RequestPermission(kind: string): Promise<PermissionStatus>;
  OpenPermissionSettings(kind: string): Promise<void>;
}

/** Passive checks never prompt. Teardown invalidates all pending completions. */
export class NativePermissionState {
  status = $state<PermissionStatus | null>(null);
  busy = $state(false);
  error = $state("");
  private disposed = false;
  private requestPending = false;
  constructor(private readonly services: NativePermissionServices) {}

  dispose() {
    this.disposed = true;
    this.busy = false;
  }

  private async run(
    action: () => Promise<PermissionStatus | void>,
    rereadOnFailure = false,
  ) {
    if (this.busy || this.disposed) return;
    this.busy = true;
    this.error = "";
    let timer: ReturnType<typeof setTimeout> | undefined;
    let active = true;
    try {
      const status = await Promise.race([
        action().catch((error: unknown) => {
          if (!rereadOnFailure || !active || this.disposed) throw error;
          // Wails drops the snapshot accompanying a rejected request. Read it
          // passively, within the same deadline and busy interval; never prompt.
          this.error = "Could not check or update permissions. Try again.";
          return this.services.NativePermissions();
        }),
        new Promise<never>((_, reject) => {
          timer = setTimeout(
            () => reject(new Error("permission-timeout")),
            15000,
          );
        }),
      ]);
      if (!this.disposed && status) this.status = status;
    } catch {
      if (!this.disposed)
        this.error = "Could not check or update permissions. Try again.";
    } finally {
      active = false;
      clearTimeout(timer);
      if (!this.disposed) this.busy = false;
    }
  }
  refresh() {
    return this.run(() => this.services.NativePermissions());
  }
  async request(kind: PermissionKind) {
    // A renderer deadline does not cancel the native prompt. Do not stack
    // further requests until the actual bridge call settles.
    if (this.requestPending) return;
    return this.run(async () => {
      this.requestPending = true;
      try {
        return await this.services.RequestPermission(kind);
      } finally {
        this.requestPending = false;
      }
    }, true);
  }
  openSettings(kind: PermissionKind) {
    return this.run(() => this.services.OpenPermissionSettings(kind));
  }
}
