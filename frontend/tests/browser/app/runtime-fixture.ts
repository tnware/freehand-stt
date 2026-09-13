import type { Preferences, Status } from "$bindings/managedruntime";
import type { ManagedRuntimeService } from "$lib/stores/managed-runtime.svelte";
import { ID, type Profile } from "$bindings/modelprofile";

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

/** Deterministic UI fixture only: no network, installation, or inference. */
export function createRuntimeFixture(
  supported: boolean,
  preferencesChanged: (p: Preferences) => void,
  ready = false,
) {
  let status: Status = {
    supported,
    state: "not_installed",
    enabled: false,
    selectedModel: "nemotron-3.5",
    realtime: true,
    backend: "",
    version: "",
    progress: -1,
    phase: "",
    error: "",
    models: [
      {
        id: "nemotron-3.5",
        name: "Nemotron 3.5",
        description: "Live streaming speech recognition",
        sizeBytes: 1_500_000_000,
        installed: false,
        recommended: true,
        realtime: true,
        profile: "nemotron-3.5-streaming",
        behavior: nemotron,
      },
      {
        id: "parakeet-tdt",
        name: "Parakeet v3",
        description: "Completed multilingual transcription",
        sizeBytes: 1_200_000_000,
        installed: false,
        recommended: false,
        realtime: false,
        profile: "parakeet-tdt-v3",
        behavior: parakeet,
      },
    ],
  };
  if (ready) {
    status = {
      ...status,
      enabled: true,
      state: "running",
      backend: "cuda",
      version: "0.1.0",
      models:
        status.models?.map((model) => ({ ...model, installed: true })) ?? [],
    };
    preferencesChanged({
      enabled: true,
      model: status.selectedModel,
      realtime: true,
    });
  }
  let downloading = "";
  const calls: string[] = [];
  const snapshot = () => structuredClone(status);
  let publish: (status: Status) => void = () => {};
  const emit = () => publish(snapshot());
  const change = (patch: Partial<Status>) => {
    status = { ...status, ...patch };
    emit();
  };
  const service: ManagedRuntimeService = {
    GetStatus: async () => snapshot(),
    SetPreferences: async (p) => {
      calls.push("SetPreferences");
      const stops = !p.enabled || p.model !== status.selectedModel;
      change({
        enabled: p.enabled,
        selectedModel: p.model,
        realtime: p.realtime,
        state: stops && status.version ? "stopped" : status.state,
      });
      preferencesChanged(p);
    },
    RefreshCatalog: async () => {
      calls.push("RefreshCatalog");
    },
    Install: async () => {
      calls.push("Install");
      change({ state: "installed", version: "v0.1.0", backend: "cpu" });
    },
    Start: async () => {
      calls.push("Start");
      change({ state: "running", phase: "", progress: -1 });
    },
    Stop: async () => {
      calls.push("Stop");
      change({ state: "stopped", phase: "", progress: -1 });
    },
    Cancel: async () => {
      calls.push("Cancel");
      downloading = "";
      change({ state: "installed", phase: "", progress: -1 });
    },
    Remove: async () => {
      calls.push("Remove");
      change({
        state: "not_installed",
        version: "",
        backend: "",
        models: (status.models ?? []).map((m) => ({ ...m, installed: false })),
      });
    },
    DownloadModel: async (id) => {
      calls.push(`DownloadModel:${id}`);
      downloading = id;
      change({ state: "installing", phase: "Downloading model", progress: -1 });
    },
    RemoveModel: async (id) => {
      calls.push(`RemoveModel:${id}`);
      change({
        state: id === status.selectedModel ? "installed" : status.state,
        models: (status.models ?? []).map((m) =>
          m.id === id ? { ...m, installed: false } : m,
        ),
      });
    },
  };
  const control = {
    calls,
    change,
    finishDownload: () => {
      if (!downloading) throw new Error("No pending fixture download");
      change({
        models: (status.models ?? []).map((m) =>
          m.id === downloading ? { ...m, installed: true } : m,
        ),
        state: "installed",
        phase: "",
        progress: -1,
      });
      downloading = "";
    },
  };
  return {
    service,
    control,
    subscribe: (callback: (status: Status) => void) => {
      publish = callback;
    },
  };
}

declare global {
  interface Window {
    testRuntime: ReturnType<typeof createRuntimeFixture>["control"];
  }
}
