import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it } from "vitest";
import {
  settings,
  idle,
  createEditor,
  serviceWithStatus,
} from "./session-fixtures";

const message = "Enter a recording limit from 1 to 262 seconds.";
function rejectedSave() {
  const error = new Error(message, {
    cause: {
      kind: "settings_validation",
      field: "maxDurationSeconds",
      message,
    },
  });
  error.name = "RuntimeError";
  return CancellablePromise.reject(error);
}

describe("field-addressable settings failures", () => {
  it("keeps applied settings and every draft while identifying the invalid control", async () => {
    const { editor, messages } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings: rejectedSave },
      }),
    );
    editor.applySettingsSnapshot(settings);
    editor.draft!.maxDurationSeconds = 0;
    editor.draft!.transcriptionTimeoutSeconds = 75;
    expect(await editor.save()).toBe(false);
    expect(editor.validationIssue).toEqual({
      field: "maxDurationSeconds",
      section: "audio",
      control: "max-duration",
      message,
    });
    expect(messages.error).toBe("");
    expect(editor.applied!.maxDurationSeconds).toBe(
      settings.maxDurationSeconds,
    );
    expect(editor.draft!.maxDurationSeconds).toBe(0);
    expect(editor.draft!.transcriptionTimeoutSeconds).toBe(75);
    editor.discardSettingsDraft();
    expect(editor.validationIssue).toBeNull();
  });
});
