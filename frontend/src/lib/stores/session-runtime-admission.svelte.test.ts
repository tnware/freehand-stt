import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import { ProviderID, type InstanceStatus } from "$bindings/managedruntime";
import { Purpose } from "$bindings/savedconnection";
import type { ConnectionResult, Settings } from "$lib/state";
import type { ManagedRuntimeService } from "./managed-runtime.svelte";
import {
  Session,
  connectionResult,
  idle,
  serviceWithStatus,
  settings,
} from "./session-fixtures";

function runtimeBoundary() {
  const row: InstanceStatus = {
    instance: {
      id: "local",
      name: "Local speech",
      provider: ProviderID.NeMoSpeechCPP,
      model: "nemotron-3.5",
      autoStart: false,
    },
    activeModel: "",
    status: {
      acquisition: { phase: "", bytes: 0, totalBytes: 0 },
      operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
      supported: true,
      state: "installed",
      enabled: false,
      realtime: false,
      selectedModel: "nemotron-3.5",
      backend: "cpu",
      version: "0.1.0",
      progress: -1,
      phase: "",
      error: "",
      models: [],
    },
  };
  const Start = vi.fn(async () => {
    row.status.state = "running";
    row.activeModel = row.instance.model;
  });
  const Stop = vi.fn(async () => {
    row.status.state = "stopped";
    row.activeModel = "";
  });
  const unexpected = async () => {
    throw new Error("Unexpected runtime operation in admission test");
  };
  const service: ManagedRuntimeService = {
    GetInstances: async () => [structuredClone(row)],
    GetProviders: async () => [],
    GetBinaryOptions: unexpected,
    SetInstance: unexpected,
    DeleteInstance: unexpected,
    Install: unexpected,
    InstallBackend: unexpected,
    Remove: unexpected,
    Start,
    Stop,
    Restart: unexpected,
    Cancel: unexpected,
    RefreshCatalog: unexpected,
    DownloadModel: unexpected,
    RemoveModel: unexpected,
  };
  return { service, Start, Stop };
}

describe("Session runtime mutation admission", () => {
  it("preserves feature edits begun while a runtime preference save is pending", async () => {
    const pending = CancellablePromise.withResolvers<void>();
    const runtime = runtimeBoundary();
    runtime.service.SetInstance = vi.fn(() => pending.promise);
    const updated = structuredClone(settings);
    updated.managedRuntimes = [
      {
        id: "local",
        name: "Local speech",
        provider: ProviderID.NeMoSpeechCPP,
        model: "nemotron-3.5",
        autoStart: true,
      },
    ];
    const session = new Session({
      ...serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          GetSettings: vi
            .fn()
            .mockImplementationOnce(() => CancellablePromise.resolve(settings))
            .mockImplementationOnce(() => CancellablePromise.resolve(updated)),
        },
      }),
      runtime: runtime.service,
    });
    let saving: Promise<boolean> | undefined;
    try {
      await session.load();
      saving = session.runtime.saveInstance(updated.managedRuntimes[0]);
      expect(runtime.service.SetInstance).toHaveBeenCalledOnce();
      session.editor.draft!.fileTranscriptionTimeoutSeconds = 75;
      pending.resolve();
      expect(await saving).toBe(true);
      expect(session.editor.draft!.fileTranscriptionTimeoutSeconds).toBe(75);
      expect(session.editor.dirty).toBe(true);
      session.editor.discardSettingsDraft();
      expect(session.editor.applied).toMatchObject(updated);
      expect(session.editor.dirty).toBe(false);
    } finally {
      pending.resolve();
      await saving;
      session.dispose();
    }
  });

  it.each(["file", "cleanup", "speech", "saved connection"] as const)(
    "permits Start and Stop during a pending %s metadata probe",
    async (feature) => {
      const metadata = CancellablePromise.withResolvers<ConnectionResult>();
      const runtime = runtimeBoundary();
      const current = structuredClone(settings);
      current.savedConnections.entries = [
        {
          id: "remote",
          name: "Independent remote endpoint",
          builtIn: false,
          uses: [Purpose.Transcription],
          hasCredential: false,
          details: {
            baseURL: current.baseURL,
            compatibilityProfile: current.compatibilityProfile,
            authenticationMode: current.authenticationMode,
            allowInsecureHTTP: false,
            healthPath: "",
            headers: {},
          },
        },
      ];
      const session = new Session({
        ...serviceWithStatus(() => CancellablePromise.resolve(idle), {
          settings: { GetSettings: () => CancellablePromise.resolve(current) },
          connection: {
            TestConnection: () => metadata.promise,
            TestPostProcessingConnection: () => metadata.promise,
            TestTextToSpeechConnection: () => metadata.promise,
            TestSavedConnection: () => metadata.promise,
          },
        }),
        runtime: runtime.service,
      });
      let checking: Promise<void> | undefined;
      try {
        await session.load();
        checking =
          feature === "file"
            ? session.editor.testConnection()
            : feature === "cleanup"
              ? session.editor.testPostProcessingConnection()
              : feature === "speech"
                ? session.editor.testTextToSpeechConnection()
                : session.editor.testSavedConnection("remote");
        expect(session.editor.busy).toBe(true);
        expect(session.editor.dirty).toBe(false);

        expect(await session.runtime.run("local", "Start")).toBe(true);
        expect(runtime.Start).toHaveBeenCalledExactlyOnceWith({
          instanceID: "local",
        });
        expect(session.runtime.statusFor("local")?.status.state).toBe(
          "running",
        );
        expect(await session.runtime.run("local", "Stop")).toBe(true);
        expect(runtime.Stop).toHaveBeenCalledExactlyOnceWith({
          instanceID: "local",
        });
        expect(session.runtime.errorFor("local")).toBe("");
        // Runtime completion must not cancel or finish the independent probe.
        expect(session.editor.busy).toBe(true);
      } finally {
        metadata.resolve(connectionResult);
        await checking;
        session.dispose();
      }
    },
  );

  it("still blocks runtime changes while settings or credential drafts are dirty", async () => {
    const runtime = runtimeBoundary();
    const session = new Session({
      ...serviceWithStatus(() => CancellablePromise.resolve(idle)),
      runtime: runtime.service,
    });
    try {
      await session.load();
      session.editor.draft!.language = "fr";
      expect(await session.runtime.run("local", "Start")).toBe(false);
      session.editor.discardSettingsDraft();
      session.editor.apiKey = "fixture-transient-credential";
      expect(await session.runtime.run("local", "Start")).toBe(false);
      expect(runtime.Start).not.toHaveBeenCalled();
      session.editor.clearCredentialDraft();
      expect(await session.runtime.run("local", "Start")).toBe(true);
      expect(runtime.Start).toHaveBeenCalledOnce();
    } finally {
      session.dispose();
    }
  });

  it("waits for a pending quick save even when the visible draft is still clean", async () => {
    const saved = CancellablePromise.withResolvers<Settings>();
    const SaveSettings = vi.fn(() => saved.promise);
    const runtime = runtimeBoundary();
    const session = new Session({
      ...serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
      runtime: runtime.service,
    });
    let saving: Promise<boolean> | undefined;
    try {
      await session.load();
      saving = session.editor.updateQuickSettings(
        { language: "fr" },
        "stt-language",
      );
      await vi.waitFor(() => expect(SaveSettings).toHaveBeenCalledOnce());
      expect(session.editor.dirty).toBe(false);
      expect(session.editor.quickSettingsPending).toEqual(["stt-language"]);
      expect(await session.runtime.run("local", "Start")).toBe(false);
      expect(runtime.Start).not.toHaveBeenCalled();
      saved.resolve({ ...settings, language: "fr" });
      expect(await saving).toBe(true);
      expect(await session.runtime.run("local", "Start")).toBe(true);
      expect(runtime.Start).toHaveBeenCalledOnce();
    } finally {
      saved.resolve(settings);
      await saving;
      session.dispose();
    }
  });
});
