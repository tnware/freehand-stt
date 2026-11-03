import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import { Action, Purpose } from "$bindings/savedconnection";
import { ID, Role } from "$bindings/compatibility";
import { ID as ModelID, type Profile } from "$bindings/modelprofile";
import { TTSPhase } from "$lib/state";
import { ProviderID, type InstanceStatus } from "$bindings/managedruntime";
import type { SettingsDTO } from "$bindings/settings";
import {
  Session,
  settings,
  idle,
  serviceWithStatus,
} from "$lib/stores/session-fixtures";
import {
  runtimeWorkflowConnection,
  useRuntimeConnection,
} from "./runtimeWorkflowUse";

function setup() {
  const current = structuredClone(settings);
  current.savedConnections = {
    selected: { voice: "manual", stt: "manual" },
    entries: [
      {
        id: "builtin-local",
        name: "Local speech",
        builtIn: true,
        uses: [Purpose.Voice, Purpose.Transcription],
        hasCredential: false,
        details: {
          managedInstanceID: "local",
          compatibilityProfile: ID.$zero,
          baseURL: "",
          authenticationMode: current.authenticationMode,
          allowInsecureHTTP: false,
          healthPath: "",
          headers: {},
        },
      },
      {
        id: "manual",
        name: "Manual speech",
        builtIn: false,
        uses: [Purpose.Voice, Purpose.Transcription],
        hasCredential: false,
        details: {
          compatibilityProfile: current.compatibilityProfile,
          baseURL: current.baseURL,
          authenticationMode: current.authenticationMode,
          allowInsecureHTTP: false,
          healthPath: "",
          headers: {},
        },
      },
    ],
  };
  const pending = CancellablePromise.withResolvers<SettingsDTO>();
  const SaveSettings = vi.fn(
    (_request: import("$bindings/settings").SaveSettingsRequest) =>
      pending.promise,
  );
  const services = serviceWithStatus(() => CancellablePromise.resolve(idle), {
    settings: { SaveSettings },
  });
  const session = new Session(services);
  session.editor.applySettingsSnapshot(current);
  const row: InstanceStatus = {
    instance: {
      id: "local",
      name: "Local speech",
      provider: ProviderID.NeMoSpeechCPP,
      model: "nemotron",
      autoStart: false,
    },
    activeModel: "nemotron",
    status: {
      supported: true,
      state: "running",
      enabled: false,
      realtime: true,
      selectedModel: "nemotron",
      backend: "cpu",
      version: "0.1.0",
      progress: -1,
      phase: "",
      error: "",
      operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
      acquisition: { phase: "", bytes: 0, totalBytes: 0 },
      models: [],
    },
  };
  session.runtime.instances = [row];
  const profile: Profile = {
    id: ModelID.Nemotron35,
    name: "Nemotron",
    description: "Fixture",
    reasoningOffRequired: false,
    capabilities: {
      nemoTranscriptionControls: true,
      realtime: true,
      serverLoadedModel: true,
      vllmTranscriptionEvents: false,
      cleanupOutputLimit: false,
      cleanupDisableReasoning: false,
      fileStreaming: false,
      typedTranscriptionEvents: false,
      legacyTranscriptionSegments: false,
      languageHint: true,
      voiceDiscovery: false,
      speechInstructions: false,
      speechLanguage: false,
      speechSpeed: false,
      transcriptionPrompt: false,
      transcriptionHotwords: false,
      transcriptionTemperature: false,
    },
  };
  session.runtime.providers = [
    {
      id: row.instance.provider,
      name: "NeMo",
      supported: true,
      backends: ["cpu"],
      version: "0.1.0",
      models: [
        {
          id: row.instance.model,
          name: "Nemotron",
          description: "",
          installed: true,
          recommended: true,
          realtime: true,
          profile: profile.id,
          sizeBytes: 0,
          contracts: [
            {
              role: Role.Transcription,
              compatibilityProfile: ID.NeMoSpeechV1,
              modelProfile: profile.id,
              behavior: profile,
            },
          ],
        },
      ],
    },
  ];
  return { current, session, SaveSettings, pending };
}

describe("explicit runtime workflow selection", () => {
  it.each([
    ["voice", Purpose.Voice],
    ["file", Purpose.Transcription],
  ] as const)(
    "selects only the %s connection through the existing transaction",
    async (workflow, purpose) => {
      const { current, session, SaveSettings, pending } = setup();
      try {
        const before = structuredClone(current);
        const using = useRuntimeConnection(session, "local", workflow);
        expect(SaveSettings).toHaveBeenCalledOnce();
        const request = SaveSettings.mock.calls[0][0];
        expect(request).toMatchObject({
          expectedConnections: { voice: "manual", stt: "manual" },
          connectionChange: {
            action: Action.Select,
            id: "builtin-local",
            purpose,
            name: "",
          },
          settings: before,
          clearConnectionCredential: false,
          connectionCredentialDraft: "",
        });
        expect(
          session.editor.applied!.savedConnections.selected![purpose],
        ).toBe("manual");
        expect(session.editor.saving).toBe(true);
        const saved = structuredClone(current);
        saved.savedConnections.selected![purpose] = "builtin-local";
        pending.resolve(saved);
        expect(await using).toBe(true);
        expect(
          session.editor.applied!.savedConnections.selected![purpose],
        ).toBe("builtin-local");
        expect(session.editor.applied!.rememberedModels).toEqual(
          before.rememberedModels,
        );
        expect(session.editor.applied!.voiceTranscription.realtime).toBe(
          before.voiceTranscription.realtime,
        );
      } finally {
        session.dispose();
      }
    },
  );

  it("returns without saving when this workflow already uses the runtime", async () => {
    const { current, session, SaveSettings } = setup();
    try {
      current.savedConnections.selected!.voice = "builtin-local";
      session.editor.applySettingsSnapshot(current);
      expect(await useRuntimeConnection(session, "local", "voice")).toBe(true);
      expect(SaveSettings).not.toHaveBeenCalled();
    } finally {
      session.dispose();
    }
  });

  it("keeps selection and manual settings when the transaction fails", async () => {
    const { current, session, SaveSettings, pending } = setup();
    try {
      const using = useRuntimeConnection(session, "local", "voice");
      pending.reject(new Error("Fixture selection failed"));
      expect(await using).toBe(false);
      expect(SaveSettings).toHaveBeenCalledOnce();
      expect(session.editor.applied).toMatchObject(current);
      expect(session.messages.error).toContain("Fixture selection failed");
    } finally {
      session.dispose();
    }
  });

  it.each([
    "dirty",
    "connection-draft",
    "saving",
    "quick-save",
    "work",
    "recovery",
    "starting",
    "busy",
    "speech-generating",
    "speech-submitting",
    "speech-previewing",
  ])("blocks %s without an unwanted settings save", async (guard) => {
    const { current, session, SaveSettings } = setup();
    try {
      if (guard === "dirty") session.editor.draft!.maxDurationSeconds++;
      if (guard === "connection-draft")
        session.editor.beginConnection(current.savedConnections.entries![1]);
      if (guard === "saving") session.editor.saving = true;
      if (guard === "quick-save")
        session.editor.quickSettingsPending = ["voice-transcription"];
      if (guard === "recovery")
        session.editor.applied!.configuration.recoveryRequired = true;
      if (guard === "starting")
        session.runtime.instances[0].status.state = "starting";
      if (guard === "busy")
        session.runtime.instances[0].status.phase = "catalog";
      if (guard === "speech-generating")
        session.speech.status.phase = TTSPhase.Generating;
      if (guard === "speech-submitting") session.speech.submitting = true;
      if (guard === "speech-previewing") session.speech.previewing = true;
      expect(
        await useRuntimeConnection(session, "local", "voice", guard === "work"),
      ).toBe(false);
      expect(SaveSettings).not.toHaveBeenCalled();
      expect(session.editor.applied!.savedConnections.selected!.voice).toBe(
        "manual",
      );
    } finally {
      session.dispose();
    }
  });

  it("does not offer unsupported roles, legacy aliases, or unknown runtime models", () => {
    const { session } = setup();
    try {
      expect(
        runtimeWorkflowConnection(session, "local", "tts"),
      ).toBeUndefined();
      session.editor.applied!.savedConnections.entries![0].builtIn = false;
      expect(
        runtimeWorkflowConnection(session, "local", "voice"),
      ).toBeUndefined();
      session.editor.applied!.savedConnections.entries![0].builtIn = true;
      session.runtime.instances[0].status.selectedModel =
        "different-selected-model";
      expect(
        runtimeWorkflowConnection(session, "local", "voice"),
      ).toBeUndefined();
      session.runtime.instances[0].status.selectedModel = "nemotron";
      session.runtime.instances[0].activeModel = "provider-api-alias";
      expect(runtimeWorkflowConnection(session, "local", "voice")?.id).toBe(
        "builtin-local",
      );
      session.runtime.instances[0].instance.model = "unknown";
      expect(
        runtimeWorkflowConnection(session, "local", "voice"),
      ).toBeUndefined();
    } finally {
      session.dispose();
    }
  });
});
