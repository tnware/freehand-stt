import type {
  Instance,
  InstanceStatus,
  Status,
  ProviderDescriptor,
} from "$bindings/managedruntime";
import type { ManagedRuntimeService } from "$lib/stores/managed-runtime.svelte";
import { ID, type Profile } from "$bindings/modelprofile";
import { ProviderID } from "$bindings/managedruntime";
import { ID as CompatibilityID, Role } from "$bindings/compatibility";

// Small UI-only contract fixture; production catalog metadata comes from Go.
const nemotron: Profile = {
  id: ID.Nemotron35,
  name: "Nemotron 3.5",
  description: "Streaming transcription",
  reasoningOffRequired: false,
  realtimeLanguageHint: true,
  languages: [
    { code: "auto", label: "Automatic detection" },
    { code: "en-US", label: "English (United States)" },
    { code: "fr-FR", label: "French (France)" },
  ],
  capabilities: {
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
const parakeet: Profile = {
  ...nemotron,
  id: ID.ParakeetTDT,
  name: "Parakeet v3",
  realtimeLanguageHint: false,
  languages: [{ code: "auto", label: "Automatic detection" }],
  capabilities: {
    ...nemotron.capabilities,
    realtime: false,
    languageHint: false,
  },
};

/** UI boundary fixture only; no native installation or inference acceptance. */
export function createRuntimeFixture(
  supported: boolean,
  preferencesChanged: (instances: Instance[]) => void,
  ready = true,
  providerID = ProviderID.NeMoSpeechCPP,
) {
  let models = [
    {
      id: "nemotron-3.5",
      name: "Nemotron 3.5 Streaming",
      description: "Multilingual speech recognition with realtime dictation.",
      sizeBytes: 0,
      installed: false,
      recommended: true,
      realtime: true,
      profile: "nemotron-3.5-streaming",
      behavior: nemotron,
      contracts: [Role.Transcription, Role.Realtime].map((role) => ({
        role,
        compatibilityProfile: CompatibilityID.NeMoSpeechV1,
        modelProfile: ID.Nemotron35,
        behavior: nemotron,
      })),
    },
    {
      id: "parakeet-tdt",
      name: "Parakeet TDT v3",
      description: "Multilingual completed speech recognition.",
      sizeBytes: 0,
      installed: false,
      recommended: false,
      realtime: false,
      profile: "parakeet-tdt-v3",
      behavior: parakeet,
      contracts: [
        {
          role: Role.Transcription,
          compatibilityProfile: CompatibilityID.NeMoSpeechV1,
          modelProfile: ID.ParakeetTDT,
          behavior: parakeet,
        },
      ],
    },
  ];
  const ggml = providerID !== ProviderID.NeMoSpeechCPP;
  if (ggml) {
    const llama = providerID === ProviderID.LlamaCPP;
    const behavior = {
      ...parakeet,
      id: llama ? ID.S1Mini : ID.Generic,
      name: llama ? "S1-mini" : "Whisper Small",
    };
    models = [
      {
        ...models[0],
        id: llama ? "s1-mini" : "whisper-small",
        name: behavior.name,
        description: "Downloaded GGML model for browser interaction tests.",
        realtime: false,
        profile: behavior.id,
        behavior,
        contracts: [
          {
            role: llama ? Role.PostProcessing : Role.Transcription,
            compatibilityProfile: llama
              ? CompatibilityID.LlamaCPP
              : CompatibilityID.WhisperCPP,
            modelProfile: behavior.id,
            behavior,
          },
        ],
      },
    ];
  }
  const providers: ProviderDescriptor[] = [
    {
      id: providerID,
      name:
        providerID === ProviderID.LlamaCPP
          ? "llama.cpp"
          : providerID === ProviderID.WhisperCPP
            ? "whisper.cpp"
            : "NeMo-Speech.cpp",
      version: "0.1.0",
      supported,
      models,
    },
  ];
  const initial = (instance: Instance): InstanceStatus => ({
    instance,
    activeModel: "",
    status: {
      acquisition: { phase: "", bytes: 0, totalBytes: 0 },
      operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
      supported,
      state: "not_installed",
      enabled: false,
      realtime: false,
      selectedModel: instance.model,
      backend: "",
      version: "",
      progress: -1,
      phase: "",
      error: "",
      models: structuredClone(models),
    },
  });
  let rows: InstanceStatus[] = ready
    ? [
        initial({
          id: ggml ? `${providerID}-default` : "nemo-default",
          name: "Local speech",
          provider: providers[0].id,
          model: models[0].id,
          autoStart: false,
        }),
      ]
    : [];
  if (ready)
    rows[0] = {
      ...rows[0],
      activeModel: models[0].id,
      status: {
        ...rows[0].status,
        state: "running",
        backend: "cpu",
        version: "0.1.0",
        models: models.map((m) => ({ ...m, installed: true })),
      },
    };
  const preferences = () =>
    preferencesChanged(structuredClone(rows.map((r) => r.instance)));
  preferences();
  const calls: string[] = [];
  const downloading = new Map<string, string>();
  let operationID = 0;
  let publish: (row: InstanceStatus) => void = () => {};
  const get = (id: string) => {
    const row = rows.find((r) => r.instance.id === id);
    if (!row) throw new Error("Unknown fixture instance");
    return row;
  };
  const change = (id: string, patch: Partial<Status>) => {
    const row = get(id);
    row.status = { ...row.status, ...patch };
    row.activeModel = row.status.state === "running" ? row.instance.model : "";
    publish(structuredClone(row));
  };
  const service: ManagedRuntimeService = {
    GetInstances: async () => structuredClone(rows),
    GetProviders: async () => structuredClone(providers),
    GetBinaryOptions: async ({ provider }) => {
      calls.push(`GetBinaryOptions:${provider}`);
      return {
        provider,
        os: "windows",
        architecture: "amd64",
        supported: true,
        recommendedBackend: "cuda",
        reason: "Compatible NVIDIA GPU detected.",
        options: ["cpu", "cuda"].map((backend) => ({
          backend,
          supported: true,
          available: true,
          reason: "Available",
        })),
      };
    },
    OpenProcessOutput: async ({ instanceID }) => {
      calls.push(`OpenProcessOutput:${instanceID}`);
    },
    SetInstance: async (instance) => {
      calls.push(`SetInstance:${instance.id}`);
      const old = rows.find((r) => r.instance.id === instance.id);
      const row = old ?? initial(instance);
      row.status.selectedModel = instance.model;
      if (old && old.instance.model !== instance.model) {
        row.status.state = "stopped";
        row.activeModel = "";
      }
      row.instance = structuredClone(instance);
      if (!old) rows.push(row);
      preferences();
      publish(structuredClone(row));
    },
    DeleteInstance: async ({ instanceID }) => {
      calls.push(`DeleteInstance:${instanceID}`);
      rows = rows.filter((r) => r.instance.id !== instanceID);
      preferences();
    },
    Install: async ({ instanceID }) => {
      calls.push(`Install:${instanceID}`);
      change(instanceID, {
        state: "installed",
        backend: "cpu",
        version: "0.1.0",
      });
    },
    InstallBackend: async ({ instanceID, backend }) => {
      calls.push(`InstallBackend:${instanceID}:${backend}`);
      if (get(instanceID).status.state === "running")
        throw new Error("Stop the runtime before changing its binary");
      change(instanceID, { state: "installed", backend });
    },
    Start: async ({ instanceID }) => {
      calls.push(`Start:${instanceID}`);
      change(instanceID, { state: "running", phase: "", progress: -1 });
    },
    Stop: async ({ instanceID }) => {
      calls.push(`Stop:${instanceID}`);
      change(instanceID, { state: "stopped", phase: "", progress: -1 });
    },
    Cancel: async ({ instanceID }) => {
      calls.push(`Cancel:${instanceID}`);
      downloading.delete(instanceID);
      change(instanceID, {
        state: "installed",
        phase: "",
        progress: -1,
        operation: {
          ...get(instanceID).status.operation,
          outcome: "cancelled",
        },
      });
    },
    RefreshCatalog: async ({ instanceID }) => {
      calls.push(`RefreshCatalog:${instanceID}`);
    },
    Remove: async ({ instanceID }) => {
      calls.push(`Remove:${instanceID}`);
      change(instanceID, {
        state: "not_installed",
        backend: "",
        version: "",
        models:
          get(instanceID).status.models?.map((m) => ({
            ...m,
            installed: false,
          })) ?? [],
      });
    },
    DownloadModel: async ({ instanceID, model }) => {
      calls.push(`DownloadModel:${instanceID}:${model}`);
      downloading.set(instanceID, model);
      change(instanceID, {
        state: "installing",
        phase: "download",
        progress: 0,
        operation: {
          id: ++operationID,
          kind: "download",
          model,
          outcome: "running",
          error: "",
        },
        acquisition: {
          phase: "downloading",
          bytes: 0,
          totalBytes: 1_000_000_000,
        },
      });
    },
    RemoveModel: async ({ instanceID, model }) => {
      calls.push(`RemoveModel:${instanceID}:${model}`);
      change(instanceID, {
        state: "installed",
        models:
          get(instanceID).status.models?.map((m) =>
            m.id === model ? { ...m, installed: false } : m,
          ) ?? [],
      });
    },
  };
  const control = {
    calls,
    change,
    snapshot: () => structuredClone(rows),
    finishDownload: (instanceID: string) => {
      const model = downloading.get(instanceID);
      if (!model) throw new Error("No pending fixture download");
      change(instanceID, {
        state: "installed",
        operation: {
          ...get(instanceID).status.operation,
          outcome: "succeeded",
        },
        phase: "",
        progress: -1,
        models:
          get(instanceID).status.models?.map((m) =>
            m.id === model ? { ...m, installed: true } : m,
          ) ?? [],
      });
      downloading.delete(instanceID);
    },
  };
  return {
    service,
    providers,
    control,
    subscribe: (callback: (row: InstanceStatus) => void) => {
      publish = callback;
    },
  };
}
declare global {
  interface Window {
    testRuntime: ReturnType<typeof createRuntimeFixture>["control"];
  }
}
