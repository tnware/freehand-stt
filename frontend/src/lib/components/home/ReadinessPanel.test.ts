import { describe, expect, it } from "vitest";
import { render } from "svelte/server";
import {
  ProviderID,
  type InstanceStatus,
  type ProviderDescriptor,
} from "$bindings/managedruntime";
import { Role, ID as BackendID } from "$bindings/compatibility";
import { ID } from "$bindings/modelprofile";
import { Purpose } from "$bindings/savedconnection";
import { settings } from "$lib/stores/session-fixtures";
import { appReadiness } from "$lib/utils/readiness";
import { localSpeechSetup } from "$lib/utils/localSpeechSetup";
import ReadinessPanel from "./ReadinessPanel.svelte";
import LocalRuntimeSetupChoice from "../settings/LocalRuntimeSetupChoice.svelte";

const provider: ProviderDescriptor = {
  id: ProviderID.NeMoSpeechCPP,
  name: "NeMo-Speech.cpp",
  supported: true,
  version: "0.1.0",
  backends: ["cpu"],
  models: [
    {
      id: "model",
      name: "Selected local model",
      description: "Fixture",
      sizeBytes: 0,
      installed: false,
      recommended: true,
      realtime: false,
      profile: ID.Generic,
      contracts: [
        {
          role: Role.Transcription,
          compatibilityProfile: BackendID.NeMoSpeechV1,
          modelProfile: ID.Generic,
          behavior: settings.modelProfiles.transcription![0],
        },
      ],
    },
  ],
};
const instance: InstanceStatus = {
  instance: {
    id: "existing",
    name: "Existing speech",
    provider: provider.id,
    model: "model",
    autoStart: false,
  },
  activeModel: "",
  status: {
    supported: true,
    state: "installed",
    enabled: false,
    selectedModel: "model",
    realtime: false,
    backend: "cpu",
    version: "0.1.0",
    progress: -1,
    phase: "",
    error: "",
    operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
    acquisition: { phase: "", bytes: 0, totalBytes: 0 },
    models: provider.models,
  },
};
function view(
  config = structuredClone(settings),
  row = structuredClone(instance),
) {
  config.setupCompleted = false;
  const readiness = appReadiness(
    config,
    null,
    [{ id: "default", name: "Microphone", default: true }],
    false,
    "voice",
    row,
  );
  return render(ReadinessPanel, {
    props: {
      readiness,
      providers: [provider],
      instances: [row],
      onManageRuntime: () => {},
      onChooseServer: () => {},
      onTestConnection: () => {},
      onComplete: () => {},
      onDismiss: () => {},
      onOpenSettings: () => {},
    },
  }).body;
}

describe("focused local setup continuation", () => {
  it("resumes an installed model download without repeating the recommendation hero or selecting anything", () => {
    const config = structuredClone(settings);
    config.savedConnections = { entries: [], selected: {} };
    config.voiceTranscription.baseURL = "";
    config.voiceTranscription.model = "";
    const before = structuredClone(config);
    const html = view(config);
    expect(html).toContain('aria-label="Continue local speech setup"');
    expect(html).toContain("Download selected model");
    expect(html).toContain("Selected local model");
    expect(html).not.toContain('aria-label="Local speech setup"');
    expect(config).toEqual({ ...before, setupCompleted: false });
  });

  it("keeps a selected remote connection needing a model ahead of an unrelated local installation", () => {
    const config = structuredClone(settings);
    config.voiceTranscription.baseURL = "https://remote.example.test/v1";
    config.voiceTranscription.model = "";
    config.savedConnections = {
      selected: { voice: "remote" },
      entries: [
        {
          id: "remote",
          name: "Chosen remote server",
          builtIn: false,
          hasCredential: false,
          uses: [Purpose.Voice],
          details: config.voiceTranscription,
        },
      ],
    };
    const html = view(config);
    expect(html).toContain('aria-label="Current transcription connection"');
    expect(html).toContain("Chosen remote server");
    expect(html).toContain("Choose a model to continue.");
    expect(html).not.toContain('aria-label="Continue local speech setup"');
    expect(html).not.toContain('aria-label="Local speech setup"');
  });

  it("keeps a selected managed model visible while setup is incomplete", () => {
    const config = structuredClone(settings);
    config.voiceTranscription.managedInstanceID = "existing";
    const html = view(config);
    expect(html).toContain('aria-label="Current transcription connection"');
    expect(html).toContain("Download the selected model to continue.");
    expect(html).toContain("Continue local setup");
    expect(html).not.toContain('aria-label="Local speech setup"');
  });

  it("never resumes unsupported providers or invents an installed model", () => {
    const result = localSpeechSetup(
      [{ ...provider, supported: false }],
      [instance],
    );
    expect(result.existing).toBeNull();
    expect(result.availability).toBe("unavailable");
    const config = structuredClone(settings);
    config.voiceTranscription.managedInstanceID = "existing";
    const html = view(config, {
      ...instance,
      status: { ...instance.status, supported: false },
    });
    expect(html).toContain("Unavailable on this machine");
    expect(html).not.toContain("Install this runtime to continue.");
  });
});

describe("empty cleanup and speech local routes", () => {
  it.each([
    [Role.PostProcessing, "cleanup"],
    [Role.Speech, "speech"],
  ] as const)("offers only a qualified %s local route", (role, label) => {
    const entry = structuredClone(provider);
    entry.models![0].contracts![0].role = role;
    const html = render(LocalRuntimeSetupChoice, {
      props: { providers: [entry], role, label, onOpen: () => {} },
    }).body;
    expect(html).toContain(`Set up local ${label}`);
    expect(html.replace(/\s+/g, " ")).toContain(
      "does not download models or enable this feature",
    );
    const unavailable = render(LocalRuntimeSetupChoice, {
      props: {
        providers: [{ ...entry, supported: false }],
        role,
        label,
        onOpen: () => {},
      },
    }).body;
    expect(unavailable).toContain("unavailable on this platform");
    expect(unavailable).not.toContain(`Set up local ${label}`);
  });
});
