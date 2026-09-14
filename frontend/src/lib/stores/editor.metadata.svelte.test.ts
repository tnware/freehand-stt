import { describe, expect, it, vi } from "vitest";
import { CancellablePromise } from "@wailsio/runtime";
import { Purpose } from "$bindings/savedconnection";
import { modelFor } from "$lib/utils/modelSettings";
import {
  createEditor,
  settings,
  idle,
  serviceWithStatus,
  connectionResult,
} from "./session-fixtures";

const roles = [
  Purpose.Voice,
  Purpose.Transcription,
  Purpose.Cleanup,
  Purpose.Speech,
];
function fixture() {
  const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
  const probe = vi.fn(() => CancellablePromise.resolve(connectionResult));
  services.connection.TestSavedConnection = probe;
  services.connection.TestConnection = probe;
  services.connection.TestPostProcessingConnection = probe;
  services.connection.TestTextToSpeechConnection = probe;
  const { editor } = createEditor(services);
  const snapshot = structuredClone(settings);
  snapshot.savedConnections.selected = {
    voice: "a",
    stt: "a",
    cleanup: "a",
    speech: "a",
  };
  editor.applySettingsSnapshot(snapshot);
  return { editor, services, probe, snapshot };
}

describe("shared metadata lifecycle", () => {
  it.each(roles)(
    "rejects late %s metadata after a dirty same-ID external snapshot without losing drafts",
    async (role) => {
      const { editor, probe, snapshot } = fixture();
      let finish!: (value: typeof connectionResult) => void;
      probe.mockImplementationOnce(() => new CancellablePromise(resolve => { finish = resolve; }));
      const pending = editor.ensureConnectionMetadata(role);
      editor.draft!.language = "ja";
      editor.apiKey = "uncommitted-stt";
      editor.processingAPIKey = "uncommitted-cleanup";
      editor.ttsAPIKey = "uncommitted-speech";
      const confirmed = structuredClone(snapshot); // credential rotation is opaque
      confirmed.language = "fr";
      expect(editor.applySettingsSnapshot(confirmed)).toBe(false);
      expect(editor.applied!.language).toBe(snapshot.language);
      expect(editor.draft!.language).toBe("ja");
      expect(editor.dirty).toBe(true);
      finish({ ...connectionResult, modelIDs: ["stale-model"] });
      await pending;
      expect(editor.connectionMetadataResult(role)).toBeNull();
      expect(editor.connectionMetadataStatus(role)).toBe("idle");
      await editor.ensureConnectionMetadata(role);
      expect(probe).toHaveBeenCalledTimes(2);
      expect(editor.connectionMetadataResult(role)?.modelIDs).toEqual(connectionResult.modelIDs);
      if (role === Purpose.Voice) expect(probe.mock.calls[1]).toEqual(["a"]);
      else expect(probe.mock.calls[1]).toEqual([expect.objectContaining({ credentialDraft: "" })]);
      expect(editor.apiKey).toBe("uncommitted-stt");
      expect(editor.processingAPIKey).toBe("uncommitted-cleanup");
      expect(editor.ttsAPIKey).toBe("uncommitted-speech");
      expect(editor.draft!.language).toBe("ja");
      editor.discardSettingsDraft();
      expect(editor.applied!.language).toBe("fr");
      expect(editor.draft!.language).toBe("fr");
      expect(editor.dirty).toBe(false);
    },
  );
  it.each(roles)("keeps mutation guards for %s model drafts", (role) => {
    const { editor } = fixture();
    for (const flag of ["saving", "setupCompleting", "managedConnectionTesting", "configurationRetrying", "configurationResetting"] as const) {
      editor[flag] = true;
      expect(editor.chooseModel(role, "blocked/model"), flag).toBe(false);
      editor[flag] = false;
    }
    editor.quickSettingsPending = ["voice-transcription"];
    expect(editor.chooseModel(role, "blocked/model")).toBe(false);
  });
  it.each(
    roles.flatMap(
      (role) =>
        [
          { role, outcome: "ready" },
          { role, outcome: "failed" },
        ] as const,
    ),
  )(
    "preserves manual $role model choice when delayed discovery becomes $outcome",
    async ({ role, outcome }) => {
      const { editor, probe } = fixture();
      const deferred =
        CancellablePromise.withResolvers<typeof connectionResult>();
      probe.mockImplementationOnce(() => deferred.promise);
      const appliedModel = modelFor(editor.applied!, role);
      const pending = editor.ensureConnectionMetadata(role);
      expect(editor.connectionMetadataStatus(role)).toBe("loading");
      expect(editor.chooseModel(role, "manual/pending-model")).toBe(true);
      expect(modelFor(editor.draft!, role)).toBe("manual/pending-model");
      if (outcome === "ready") deferred.resolve(connectionResult);
      else deferred.reject(new Error("offline"));
      await pending;
      expect(editor.connectionMetadataStatus(role)).toBe(outcome);
      expect(modelFor(editor.draft!, role)).toBe("manual/pending-model");
      expect(modelFor(editor.applied!, role)).toBe(appliedModel);
      expect(editor.dirty).toBe(true);
      expect(editor.connectionMetadataResult(role)?.modelIDs ?? []).toEqual(
        outcome === "ready" ? connectionResult.modelIDs : [],
      );
    },
  );
  it.each([Purpose.Transcription, Purpose.Cleanup, Purpose.Speech])(
    "does not attribute successful draft %s metadata to the applied inventory",
    async (role) => {
      const { editor } = fixture();
      if (role === Purpose.Transcription) {
        editor.draft!.baseURL = "https://draft.example/v1";
        await editor.testConnection();
      } else if (role === Purpose.Cleanup) {
        editor.draft!.postProcessing.baseURL = "https://draft.example/v1";
        await editor.testPostProcessingConnection();
      } else {
        editor.draft!.textToSpeech.baseURL = "https://draft.example/v1";
        await editor.testTextToSpeechConnection();
      }
      expect(editor.connectionResultStale(role, editor.applied)).toBe(true);
      expect(editor.connectionMetadataResult(role)).toBeNull();
      expect(editor.connectionMetadataStatus(role)).toBe("idle");
      await editor.ensureConnectionMetadata(role);
      expect(editor.connectionMetadataResult(role)?.modelIDs).toEqual(connectionResult.modelIDs);
      expect(editor.connectionMetadataStatus(role)).toBe("ready");
    },
  );
  it.each(roles)(
    "does not keep a successful %s inventory after a failed refresh",
    async (role) => {
      const { editor, probe } = fixture();
      editor.applied!.textToSpeech.enabled = true;
      await editor.ensureConnectionMetadata(role);
      probe.mockImplementationOnce(() =>
        CancellablePromise.reject(new Error("offline")),
      );
      await editor.testAppliedConnection(role);
      expect(editor.connectionMetadataStatus(role)).toBe("failed");
      expect(editor.connectionMetadataResult(role)).toBeNull();
    },
  );
  it.each(roles)(
    "retries failed %s metadata only on deliberate reentry",
    async (role) => {
      const { editor, probe } = fixture();
      probe.mockImplementationOnce(() =>
        CancellablePromise.reject(new Error("offline")),
      );
      await editor.ensureConnectionMetadata(role);
      expect(editor.connectionMetadataStatus(role)).toBe("failed");
      await editor.ensureConnectionMetadata(role);
      expect(probe).toHaveBeenCalledTimes(1);
      await editor.ensureConnectionMetadata(role, true);
      expect(probe).toHaveBeenCalledTimes(2);
      expect(editor.connectionMetadataStatus(role)).toBe("ready");
      await editor.ensureConnectionMetadata(role, true);
      expect(probe).toHaveBeenCalledTimes(2);
    },
  );
  it.each(roles)(
    "invalidates same-ID confirmed %s snapshots and rejects A→B→A late completion",
    async (role) => {
      const { editor, probe, snapshot } = fixture();
      let finish!: (value: typeof connectionResult) => void;
      probe.mockImplementationOnce(
        () =>
          new CancellablePromise((resolve) => {
            finish = resolve;
          }),
      );
      const old = editor.ensureConnectionMetadata(role);
      const b = structuredClone(snapshot);
      b.savedConnections.selected![role] = "b";
      editor.applySettingsSnapshot(b);
      editor.applySettingsSnapshot(snapshot);
      await editor.ensureConnectionMetadata(role);
      expect(probe).toHaveBeenCalledTimes(2);
      finish({ ...connectionResult, modelIDs: ["stale-model"] });
      await old;
      const result =
        role === Purpose.Voice
          ? editor.currentVoiceConnection
          : role === Purpose.Transcription
            ? editor.connection
            : role === Purpose.Cleanup
              ? editor.processingConnection
              : editor.ttsConnection;
      expect(result?.modelIDs).toEqual(connectionResult.modelIDs);
      editor.applySettingsSnapshot(structuredClone(snapshot)); // opaque credential rotation
      await editor.ensureConnectionMetadata(role);
      expect(probe).toHaveBeenCalledTimes(3);
    },
  );
  it.each(roles)(
    "discovers %s on first entry despite task/secret drafts, deduping repeats",
    async (role) => {
      const { editor, probe } = fixture();
      editor.draft!.language = "ja";
      editor.apiKey = "uncommitted";
      await Promise.all([
        editor.ensureConnectionMetadata(role),
        editor.ensureConnectionMetadata(role),
      ]);
      await editor.ensureConnectionMetadata(role);
      expect(probe).toHaveBeenCalledTimes(1);
      if (role !== Purpose.Voice)
        expect(probe.mock.calls[0]).toEqual([
          expect.objectContaining({ credentialDraft: "" }),
        ]);
    },
  );
});
