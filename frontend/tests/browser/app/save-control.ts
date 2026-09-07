import { CancellablePromise } from "@wailsio/runtime";
import type { SaveSettingsRequest, SettingsDTO } from "$bindings/settings";

export interface SaveControl {
  waitForStart: (after: number) => Promise<number>;
  complete: (
    id: number,
    outcome: "success" | "failure" | "invalid-duration",
  ) => void;
}

declare global {
  interface Window {
    testSaves: SaveControl;
  }
}

// The only test controls are at the asynchronous SaveSettings service boundary.
export function controlledSaves(
  result: (request: SaveSettingsRequest) => SettingsDTO,
) {
  let sequence = 0;
  let pending:
    | {
        id: number;
        request: SaveSettingsRequest;
        resolve: (value: SettingsDTO) => void;
        reject: (reason: Error) => void;
      }
    | undefined;
  const started: Array<(id: number) => void> = [];
  const control: SaveControl = {
    waitForStart: (after) =>
      pending && pending.id > after
        ? Promise.resolve(pending.id)
        : new Promise((resolve) => started.push(resolve)),
    complete: (id, outcome) => {
      if (!pending || pending.id !== id)
        throw new Error("No matching pending save");
      const save = pending;
      pending = undefined;
      if (outcome === "invalid-duration") {
        const message = "Enter a recording limit from 1 to 262 seconds.";
        const error = new Error(message, {
          cause: {
            kind: "settings_validation",
            field: "maxDurationSeconds",
            message,
          },
        });
        error.name = "RuntimeError";
        save.reject(error);
      } else if (outcome === "failure")
        save.reject(new Error("Fixture save failed. Try again."));
      else save.resolve(result(save.request));
    },
  };
  return {
    control,
    save: (request: SaveSettingsRequest) =>
      new CancellablePromise<SettingsDTO>((resolve, reject) => {
        if (pending) throw new Error("A save is already pending");
        pending = { id: ++sequence, request, resolve, reject };
        for (const notify of started.splice(0)) notify(sequence);
      }),
  };
}
