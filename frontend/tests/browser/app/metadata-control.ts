import { CancellablePromise } from "@wailsio/runtime";
import type { Purpose } from "$bindings/savedconnection";
import type { ConnectionResult } from "$lib/state";
import { connectionResult } from "$lib/stores/session-fixtures-data";

export function controlledMetadata() {
  const calls: Purpose[] = [];
  const pending = new Map<
    Purpose,
    {
      resolve: (result: ConnectionResult) => void;
      reject: (cause: Error) => void;
    }
  >();
  return {
    request: (purpose: Purpose) =>
      new CancellablePromise<ConnectionResult>((resolve, reject) => {
        if (pending.has(purpose))
          throw new Error("Metadata request already pending");
        calls.push(purpose);
        pending.set(purpose, { resolve, reject });
      }),
    control: {
      calls,
      complete: (purpose: Purpose, success: boolean) => {
        const request = pending.get(purpose);
        if (!request) throw new Error("No pending metadata request");
        pending.delete(purpose);
        if (success)
          request.resolve({
            ...connectionResult,
            modelIDs: ["fixture/discovered"],
          });
        else request.reject(new Error("Fixture metadata failed"));
      },
    },
  };
}

declare global {
  interface Window {
    testMetadata: ReturnType<typeof controlledMetadata>["control"];
  }
}
