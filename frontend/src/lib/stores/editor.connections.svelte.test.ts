import { describe, expect, it, vi } from "vitest";
import { CancellablePromise } from "@wailsio/runtime";
import { Action, Purpose } from "$bindings/savedconnection";
import { ID } from "$bindings/compatibility";
import { AuthenticationMode, PostProcessingPreset, type Settings } from "$lib/state";
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
          uses: [Purpose.Transcription],
          hasCredential: true,
          details: {
            compatibilityProfile: ID.Generic,
            baseURL: settings.baseURL,
            allowInsecureHTTP: false,
            authenticationMode: AuthenticationMode.AuthenticationModeAPIKey,
            healthPath: "",
            headers: {},
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
};

describe("saved connection editor", () => {
  it("only marks changed connection fields or credential intent as dirty", () => {
    const { editor } = createEditor(serviceWithStatus(() => CancellablePromise.resolve(idle)));
    editor.applySettingsSnapshot(configured());
    editor.beginConnection(editor.applied!.savedConnections.entries![0]);
    expect(editor.dirty).toBe(false);
    const form = editor.connectionDraft!;
    form.name = "Renamed";
    expect(editor.connectionDirty).toBe(true);
    form.name = "First";
    expect(editor.connectionDirty).toBe(false);
    form.uses.push(Purpose.Speech);
    expect(editor.connectionDirty).toBe(true);
    form.uses.pop();
    form.details.baseURL += "/changed";
    expect(editor.connectionDirty).toBe(true);
    form.details.baseURL = settings.baseURL;
    expect(editor.connectionDirty).toBe(false);
    form.credentialDraft = "transient-canary";
    expect(editor.dirty).toBe(true);
    form.credentialDraft = "";
    form.clearCredential = true;
    expect(editor.dirty).toBe(true);
    form.clearCredential = false;
    expect(editor.dirty).toBe(false);
    editor.cancelConnectionEdit();
    editor.beginConnection();
    expect(editor.dirty).toBe(false);
    editor.connectionDraft!.name = "New";
    expect(editor.dirty).toBe(true);
    editor.cancelConnectionEdit();
    expect(editor.connectionDraft).toBeNull();
    expect(editor.dirty).toBe(false);
  });
  it("preserves an open unchanged connection against another window's snapshot", () => {
    const { editor } = createEditor(serviceWithStatus(() => CancellablePromise.resolve(idle)));
    editor.applySettingsSnapshot(configured());
    editor.beginConnection(editor.applied!.savedConnections.entries![0]);
    const changed = configured();
    changed.savedConnections.entries![0].name = "Changed elsewhere";
    expect(editor.applySettingsSnapshot(changed)).toBe(false);
    expect(editor.connectionDraft!.name).toBe("First");
  });
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
  it("saves an explicit connection draft without activating it or altering the runtime draft", async () => {
    const next = configured();
    const SaveSettings = vi.fn(() => CancellablePromise.resolve(next));
    const { editor } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    editor.applySettingsSnapshot(configured());
    editor.beginConnection();
    editor.connectionDraft!.name = "Second";
    editor.connectionDraft!.details.baseURL = "https://second.example.test/v1";
    editor.connectionDraft!.credentialDraft = "draft-canary";
    expect(editor.dirty).toBe(true);
    expect(editor.runtimeDirty).toBe(false);
    expect(await editor.saveConnection()).toBe(true);
    expect(SaveSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        connectionChange: expect.objectContaining({
          action: Action.Create,
          name: "Second",
          details: expect.objectContaining({
            baseURL: "https://second.example.test/v1",
          }),
        }),
        connectionCredentialDraft: "draft-canary",
      }),
    );
    expect(editor.connectionDraft).toBeNull();
    expect(editor.applied?.savedConnections.selected?.stt).toBe("first");
  });
  it("clears a connection credential draft when Settings is hidden", () => {
    const { editor } = createEditor(serviceWithStatus(() => CancellablePromise.resolve(idle)));
    editor.applySettingsSnapshot(configured());
    editor.beginConnection();
    editor.connectionDraft!.credentialDraft = "draft-canary";
    editor.clearCredentialDraft();
    expect(editor.connectionDraft).toBeNull();
    expect(editor.dirty).toBe(false);
  });

  it("keeps the applied connection when a switch fails", async () => {
    const { editor } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          SaveSettings: () => CancellablePromise.reject(new Error("fixture failure")),
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
    const pending = new CancellablePromise<typeof connectionResult>((resolve) => {
      complete = resolve;
    });
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

describe("task connection creation", () => {
  it.each([Purpose.Transcription, Purpose.Cleanup, Purpose.Speech])(
    "preselects %s and requests one atomic save",
    async (purpose) => {
      const next = configured();
      const SaveSettings = vi.fn(() => CancellablePromise.resolve(next));
      const { editor } = createEditor(
        serviceWithStatus(() => CancellablePromise.resolve(idle), { settings: { SaveSettings } }),
      );
      editor.applySettingsSnapshot(configured());
      editor.beginConnection(undefined, purpose);
      expect(editor.connectionDraft!.uses).toEqual([purpose]);
      editor.connectionDraft!.name = "New task server";
      editor.connectionDraft!.details.baseURL = "https://new.example.test/v1";
      expect(await editor.saveConnection(purpose)).toBe(true);
      expect(SaveSettings).toHaveBeenCalledTimes(1);
      expect(SaveSettings).toHaveBeenCalledWith(
        expect.objectContaining({
          connectionChange: expect.objectContaining({
            action: Action.Create,
            activateFor: purpose,
            uses: [purpose],
          }),
        }),
      );
      expect(editor.connectionDraft).toBeNull();
    },
  );
  it("retains a failed new-connection draft and the active selection for retry", async () => {
    const { editor } = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          SaveSettings: () => CancellablePromise.reject(new Error("fixture save failure")),
        },
      }),
    );
    editor.applySettingsSnapshot(configured());
    editor.beginConnection(undefined, Purpose.Speech);
    editor.connectionDraft!.name = "New speech server";
    expect(await editor.saveConnection(Purpose.Speech)).toBe(false);
    expect(editor.connectionDraft!.name).toBe("New speech server");
    expect(editor.applied!.savedConnections.selected!.speech).toBe("speech");
    editor.cancelConnectionEdit();
    expect(editor.connectionDraft).toBeNull();
  });
});

it("adopts the latest external selection only after discarding a connection draft", () => {
  const { editor } = createEditor(serviceWithStatus(() => CancellablePromise.resolve(idle)));
  editor.applySettingsSnapshot(configured());
  editor.beginConnection(undefined, Purpose.Transcription);
  editor.connectionDraft!.name = "Unfinished server";
  editor.connectionDraft!.credentialDraft = "transient-canary";
  const changed = configured();
  changed.savedConnections.selected!.stt = "other-window";
  changed.model = "other-model";
  expect(editor.applySettingsSnapshot(changed)).toBe(false);
  expect(editor.connectionDraft!.name).toBe("Unfinished server");
  expect(editor.applied!.savedConnections.selected!.stt).toBe("first");
  editor.cancelConnectionEdit();
  expect(editor.connectionDraft).toBeNull();
  expect(editor.applied!.savedConnections.selected!.stt).toBe("other-window");
  expect(editor.draft!.model).toBe("other-model");
  expect(editor.dirty).toBe(false);
});

describe("connection card checks", () => {
  function setup() {
    const config = configured();
    config.savedConnections.entries!.push({
      ...config.savedConnections.entries![0],
      id: "second",
      name: "Second",
    });
    const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
    vi.spyOn(services.settings, "SaveSettings");
    services.connection.TestSavedConnection = vi.fn(() =>
      CancellablePromise.resolve(connectionResult),
    );
    const { editor } = createEditor(services);
    editor.applySettingsSnapshot(config);
    return { editor, services, config };
  }
  it("retains individual results without changing the active connection", async () => {
    const { editor, services } = setup();
    await editor.testSavedConnection("first");
    await editor.testSavedConnection("second");
    expect(Object.keys(editor.savedConnectionChecks)).toEqual(["first", "second"]);
    expect(editor.applied!.savedConnections.selected?.stt).toBe("first");
    expect(services.settings.SaveSettings).not.toHaveBeenCalled();
  });
  it("invalidates card checks while preserving a draft against an external snapshot", async () => {
    const { editor, config } = setup();
    await editor.testSavedConnection("first");
    editor.draft!.model = "unfinished-model";
    expect(editor.applySettingsSnapshot(config)).toBe(false);
    expect(editor.savedConnectionChecks).toEqual({});
    expect(editor.draft!.model).toBe("unfinished-model");
  });
  it("invalidates checks and late completions on a confirmed settings snapshot", async () => {
    const { editor, services, config } = setup();
    await editor.testSavedConnection("first");
    const response = CancellablePromise.withResolvers<typeof connectionResult>();
    services.connection.TestSavedConnection = vi.fn(() => response.promise);
    const pending = editor.testSavedConnection("second");
    expect(editor.savedConnectionCheckingID).toBe("second");
    // Identical public settings can still accompany a credential rotation.
    editor.applySettingsSnapshot(config);
    response.resolve(connectionResult);
    await pending;
    expect(editor.savedConnectionChecks).toEqual({});
    expect(editor.managedConnectionTesting).toBe(false);
    expect(editor.savedConnectionCheckingID).toBe("");
  });
  it("keeps a failed retry on its own card and rejects unknown connection ids", async () => {
    const { editor, services } = setup();
    await editor.testSavedConnection("first");
    services.connection.TestSavedConnection = vi.fn(() =>
      CancellablePromise.reject(new Error("failure")),
    );
    await editor.testSavedConnection("second");
    expect(editor.savedConnectionChecks.first).toEqual(connectionResult);
    expect(editor.savedConnectionCheckErrors.second).toContain("Try again");
    await editor.testSavedConnection("missing");
    expect(services.connection.TestSavedConnection).toHaveBeenCalledTimes(1);
  });
});
