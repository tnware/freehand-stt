import { describe, expect, it, vi } from "vitest";
import { CancellablePromise } from "@wailsio/runtime";
import { Action, Purpose } from "$bindings/savedconnection";
import { ID } from "$bindings/compatibility";
import {
  AuthenticationMode,
  PostProcessingPreset,
  type Settings,
} from "$lib/state";
import {
  createEditor,
  settings,
  idle,
  serviceWithStatus,
  connectionResult,
} from "./session-fixtures";

function configured(): Settings {
  return {
    ...structuredClone(settings),
    savedConnections: {
      selected: { stt: "first", cleanup: "cleanup", speech: "speech" },
      entries: [
        {
          id: "first",
          name: "First",
          purpose: Purpose.Transcription,
          hasCredential: true,
          details: {
            compatibilityProfile: ID.Generic,
            baseURL: settings.baseURL,
            allowInsecureHTTP: false,
            authenticationMode: AuthenticationMode.AuthenticationModeAPIKey,
            model: settings.model,
            healthPath: "",
            headers: {},
            cleanupPreset: PostProcessingPreset.$zero,
          },
        },
      ],
    },
  };
}
const select = {
  action: Action.Select,
  purpose: Purpose.Transcription,
  id: "second",
  name: "",
  replacementID: "",
};

describe("saved connection editor", () => {
  it("blocks switching an unsaved draft", async () => {
    const SaveSettings = vi.fn(() => CancellablePromise.resolve(configured()));
    const { editor } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    editor.applySettingsSnapshot(configured());
    editor.draft!.model = "unsaved";
    expect(await editor.changeConnection(select)).toBe(false);
    expect(SaveSettings).not.toHaveBeenCalled();
  });
  it("saves a new connection from the draft with expected selections and clears its key draft", async () => {
    const next = configured();
    next.savedConnections.selected!.stt = "second";
    const SaveSettings = vi.fn(() => CancellablePromise.resolve(next));
    const { editor } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    editor.applySettingsSnapshot(configured());
    editor.draft!.baseURL = "https://second.example.test/v1";
    editor.apiKey = "draft-canary";
    expect(
      await editor.changeConnection({
        ...select,
        action: Action.Create,
        name: "Second",
      }),
    ).toBe(true);
    expect(SaveSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        expectedConnections: {
          stt: "first",
          cleanup: "cleanup",
          speech: "speech",
        },
        sttCredentialDraft: "draft-canary",
        settings: expect.objectContaining({
          baseURL: "https://second.example.test/v1",
        }),
      }),
    );
    expect(editor.apiKey).toBe("");
    expect(editor.applied?.savedConnections.selected?.stt).toBe("second");
  });
  it("keeps the applied connection when a switch fails", async () => {
    const { editor } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          SaveSettings: () =>
            CancellablePromise.reject(new Error("fixture failure")),
        },
      }),
    );
    editor.applySettingsSnapshot(configured());
    expect(await editor.changeConnection(select)).toBe(false);
    expect(editor.applied?.savedConnections.selected?.stt).toBe("first");
    expect(editor.saving).toBe(false);
  });
  it("ignores an old cleanup probe after another window selects a connection", async () => {
    let complete!: (v: typeof connectionResult) => void;
    const pending = new CancellablePromise<typeof connectionResult>(
      (resolve) => {
        complete = resolve;
      },
    );
    const { editor } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        connection: { TestPostProcessingConnection: () => pending },
      }),
    );
    editor.applySettingsSnapshot(configured());
    const probe = editor.testPostProcessingConnection();
    const next = configured();
    next.savedConnections.selected!.cleanup = "different";
    editor.applySettingsSnapshot(next);
    complete(connectionResult);
    await probe;
    expect(editor.processingConnection).toBeNull();
  });
});
