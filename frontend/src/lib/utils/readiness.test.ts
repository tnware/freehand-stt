import { ID } from "$bindings/compatibility";
import { ID as ModelProfileID } from "$bindings/modelprofile";
import { describe, expect, it } from "vitest";
import {
  AuthenticationMode,
  AppearanceMode,
  ConnectionErrorKind,
  ConnectionProbe,
  ModelPresence,
  OverlayAnchor,
  OverlayLayout,
  OverlayMotion,
  OverlaySurface,
  OverlayVisibility,
  OverlayVisualizer,
  PostProcessingPreset,
  VADMode,
  type ConnectionResult,
  type Device,
  type Settings,
} from "$lib/state";
import type { Status as ManagedStatus } from "$bindings/managedruntime";
import { appReadiness, readinessVisible } from "$lib/utils/readiness";

it("uses a running managed runtime for fresh Voice setup without touching manual endpoints", () => {
  const cfg = settings();
  cfg.voiceTranscription.baseURL = "";
  cfg.voiceTranscription.model = "";
  cfg.savedConnections = { entries: [], selected: {} };
  const runtime: ManagedStatus = {
    supported: true,
    state: "running",
    enabled: true,
    selectedModel: "nemotron-3.5",
    realtime: true,
    backend: "vulkan",
    version: "0.1.0",
    progress: 1,
    phase: "Ready",
    error: "",
    models: [],
  };
  const ready = appReadiness(cfg, null, devices, false, "voice", runtime);
  expect(ready.canComplete).toBe(true);
  expect(ready.canTestConnection).toBe(false);
  expect(ready.steps.find((s) => s.id === "server")?.settingsSection).toBe(
    "local-runtime",
  );
  expect(cfg.voiceTranscription.baseURL).toBe("");
  expect(
    appReadiness(cfg, connection(), devices, false, "voice", {
      ...runtime,
      state: "stopped",
    }).canComplete,
  ).toBe(false);
  expect(
    appReadiness(cfg, connection(), devices, false, "voice", {
      ...runtime,
      supported: false,
    }).canComplete,
  ).toBe(false);
  expect(
    appReadiness(cfg, null, devices, false, "file", runtime).canTestConnection,
  ).toBe(false);
  expect(
    appReadiness(cfg, null, devices, false, "file", runtime).recoveryNeeded,
  ).toBe(false);
});

const devices: Device[] = [
  { id: "mic-1", name: "Desk microphone", default: true },
];

const settings = (overrides: Partial<Settings> = {}): Settings => ({
  managedRuntime: { enabled: false, model: "nemotron-3.5", realtime: true },
  vocabulary: { terms: "", voice: false, files: false, boost: 3 },
  savedConnections: {
    entries: [
      {
        id: "voice",
        name: "Voice",
        uses: [],
        hasCredential: overrides.credentialConfigured ?? true,
        details: {
          compatibilityProfile: ID.Generic,
          baseURL: "https://example.test/v1",
          allowInsecureHTTP: false,
          authenticationMode: AuthenticationMode.AuthenticationModeAPIKey,
          healthPath: "",
          headers: {},
        },
      },
    ],
    selected: { voice: "voice" },
  },
  transcriptionOptions: {
    prompt: "",
    temperatureOverride: false,
    temperature: 0,
  },
  compatibilityProfile: ID.Generic,
  compatibilityProfiles: {
    transcription: [],
    postProcessing: [],
    speech: [],
    realtime: [],
  },
  rememberedModels: { entries: [], defaults: {} },
  modelProfiles: {
    voiceTranscription: [],
    transcription: [],
    postProcessing: [],
    speech: [],
    realtime: [],
  },
  modelProfile: ModelProfileID.Generic,
  transcriptionLanguages: [],
  realtimeLanguages: [],
  voiceTranscription: {
    realtime: false,
    healthPath: "",
    headers: {},
    timeoutSeconds: 120,
    transcriptionOptions: {
      prompt: "",
      temperatureOverride: false,
      temperature: 0,
    },
    compatibilityProfile: overrides.compatibilityProfile ?? ID.Generic,
    modelProfile: ModelProfileID.Generic,
    baseURL: overrides.baseURL ?? "https://example.test/v1",
    allowInsecureHTTP: false,
    authenticationMode:
      overrides.authenticationMode ??
      AuthenticationMode.AuthenticationModeAPIKey,
    model: overrides.model ?? "speech/stt",
    language: "auto",
    captions: true,
  },
  baseURL: "https://example.test/v1",
  allowInsecureHTTP: false,
  authenticationMode: AuthenticationMode.AuthenticationModeAPIKey,
  model: "speech/stt",
  headers: {},
  toggleShortcut: "Ctrl+Shift+Space",
  showShortcut: "Ctrl+Shift+D",
  maxDurationSeconds: 120,
  transcriptionTimeoutSeconds: 120,
  fileTranscriptionTimeoutSeconds: 21600,
  autoInsert: true,
  platform: "windows",
  startWithWindows: false,
  showWindowOnLaunch: true,
  checkForUpdates: true,
  setupCompleted: false,
  useMica: false,
  appearanceMode: AppearanceMode.AppearanceModeSystem,
  overlayEnabled: true,
  overlaySizePercent: 100,
  overlayOpacityPercent: 100,
  overlayTopOffset: 18,
  overlayGlowPercent: 100,
  overlayLayout: OverlayLayout.OverlayLayoutCapsule,
  overlayAnchor: OverlayAnchor.OverlayAnchorTopCenter,
  overlayVisibility: OverlayVisibility.OverlayVisibilityAll,
  overlayMotion: OverlayMotion.OverlayMotionSystem,
  overlaySurface: OverlaySurface.OverlaySurfaceGlass,
  overlayVisualizer: OverlayVisualizer.OverlayVisualizerBars,
  historyEnabled: false,
  vadEnabled: true,
  vadMode: VADMode.VADModeAggressive,
  vadActivitySilenceMilliseconds: 400,
  silenceTrimming: false,
  speechPaddingMilliseconds: 300,
  autoStopEnabled: false,
  autoStopSilenceMilliseconds: 2000,
  autoStopMinimumSpeechMilliseconds: 300,
  silenceSplitting: false,
  segmentSeconds: 90,
  segmentSilenceMilliseconds: 700,
  postProcessing: {
    generationOptions: {
      limitOutputTokens: false,
      maxOutputTokens: 0,
      disableReasoning: false,
    },
    compatibilityProfile: ID.Generic,
    enabled: false,
    baseURL: "http://127.0.0.1:8080/v1",
    allowInsecureHTTP: true,
    model: "",
    preset: PostProcessingPreset.PostProcessingPresetGeneric,
    systemPrompt: "Clean the transcript.",
    styling: "semi-casual",
    structure: "prose",
    context: "general",
    timeoutSeconds: 120,
  },
  textToSpeech: {
    options: { language: "", instructions: "" },
    modelProfile: ModelProfileID.Generic,
    compatibilityProfile: ID.Generic,
    enabled: false,
    baseURL: "",
    allowInsecureHTTP: false,
    authenticationMode: AuthenticationMode.AuthenticationModeNone,
    model: "",
    voice: "",
    speed: 1,
    timeoutSeconds: 180,
  },
  configuration: {
    recoveryRequired: false,
  },
  credentialConfigured: true,
  postProcessingCredentialConfigured: false,
  textToSpeechCredentialConfigured: false,
  holdAvailable: true,
  holdAvailabilityReason: "",
  micaActive: false,
  appearanceModeActive: AppearanceMode.AppearanceModeSystem,
  ...overrides,
});

const connection = (
  errorKind = ConnectionErrorKind.$zero,
): ConnectionResult => ({
  reachable: errorKind === ConnectionErrorKind.$zero,
  probe: ConnectionProbe.ConnectionProbeModels,
  requestedURL: "https://example.test/v1/models",
  httpStatus: errorKind === ConnectionErrorKind.$zero ? 200 : 503,
  latencyMilliseconds: 18,
  errorKind,
  checkedAt: "2026-08-31T12:00:00Z",
  modelPresence: ModelPresence.ModelPresenceListed,
  modelIDs: ["speech/stt"],
});

describe("app readiness", () => {
  it("blocks first-run completion for a reachable server with an invalid model response", () => {
    const value = {
      ...connection(ConnectionErrorKind.ConnectionErrorResponse),
      reachable: true,
      httpStatus: 200,
      modelPresence: ModelPresence.ModelPresenceUnavailable,
      modelIDs: [],
    };
    const readiness = appReadiness(settings(), value, devices, false);
    expect(readiness.canComplete).toBe(false);
    expect(
      readiness.steps.find((step) => step.id === "connection")?.detail,
    ).toContain("Invalid response");
  });

  it("states what the successful setup check verified", () => {
    const readiness = appReadiness(settings(), connection(), devices, false);
    expect(
      readiness.steps.find((step) => step.id === "connection")?.detail,
    ).toBe("Model list received in 18 ms. No model was invoked.");
  });

  it("requires one successful metadata check before initial setup can finish", () => {
    const before = appReadiness(settings(), null, devices, false);
    expect(before.show).toBe(true);
    expect(before.canComplete).toBe(false);
    expect(before.steps.find((step) => step.id === "connection")?.status).toBe(
      "pending",
    );

    const after = appReadiness(settings(), connection(), devices, false);
    expect(after.canComplete).toBe(true);
    expect(after.completedCount).toBe(after.steps.length);
  });

  it.each(["", "   "])(
    "allows initial setup with an unassigned toggle shortcut (%j)",
    (toggleShortcut) => {
      const current = settings({ toggleShortcut });
      const readiness = appReadiness(current, connection(), devices, false);

      expect(readiness.canComplete).toBe(true);
      expect(readiness.recoveryNeeded).toBe(false);
      expect(readiness.completedCount).toBe(readiness.steps.length);
      expect(
        readiness.steps.find((step) => step.id === "shortcut"),
      ).toMatchObject({
        status: "complete",
        blocking: false,
        detail:
          "Not assigned. Use Start recording in Freehand, or record a shortcut in Settings.",
        settingsSection: "shortcuts",
      });
      expect(current.toggleShortcut).toBe(toggleShortcut);
    },
  );

  it("does not require recovery after clearing the toggle shortcut", () => {
    const readiness = appReadiness(
      settings({ setupCompleted: true, toggleShortcut: "" }),
      null,
      devices,
      false,
    );
    expect(readiness.show).toBe(false);
    expect(readiness.recoveryNeeded).toBe(false);
    expect(readinessVisible(readiness, "")).toBe(false);
  });

  it("does not nag a completed setup merely because this session has not probed the server", () => {
    const readiness = appReadiness(
      settings({ setupCompleted: true }),
      null,
      devices,
      false,
    );
    expect(readiness.show).toBe(false);
  });

  it("returns for an established user after a failed explicit connection check", () => {
    const readiness = appReadiness(
      settings({ setupCompleted: true }),
      connection(ConnectionErrorKind.ConnectionErrorHTTP),
      devices,
      false,
    );
    expect(readiness.show).toBe(true);
    expect(readiness.recoveryNeeded).toBe(true);
    expect(
      readiness.steps.find((step) => step.id === "connection")?.status,
    ).toBe("attention");
  });

  it("lets an established user dismiss one recovery state without bypassing first-run setup", () => {
    const recovery = appReadiness(
      settings({ setupCompleted: true }),
      connection(ConnectionErrorKind.ConnectionErrorHTTP),
      devices,
      false,
    );
    expect(readinessVisible(recovery, "")).toBe(true);
    expect(readinessVisible(recovery, recovery.recoveryKey)).toBe(false);

    const firstRun = appReadiness(settings(), null, devices, false);
    expect(readinessVisible(firstRun, firstRun.recoveryKey)).toBe(true);
  });

  it("shows recovery again when the underlying problem changes", () => {
    const first = appReadiness(
      settings({ setupCompleted: true }),
      connection(ConnectionErrorKind.ConnectionErrorHTTP),
      devices,
      false,
    );
    const changed = appReadiness(
      settings({ setupCompleted: true, microphoneID: "missing-mic" }),
      connection(ConnectionErrorKind.ConnectionErrorHTTP),
      devices,
      false,
    );
    expect(changed.recoveryKey).not.toBe(first.recoveryKey);
    expect(readinessVisible(changed, first.recoveryKey)).toBe(true);
  });

  it("requires a credential only for API-key authentication", () => {
    expect(
      appReadiness(
        settings({ credentialConfigured: false }),
        null,
        devices,
        false,
      ).canTestConnection,
    ).toBe(false);
    expect(
      appReadiness(
        settings({
          authenticationMode: AuthenticationMode.AuthenticationModeNone,
          credentialConfigured: false,
        }),
        null,
        devices,
        false,
      ).canTestConnection,
    ).toBe(true);
  });

  it("surfaces an unavailable selected microphone after setup", () => {
    const readiness = appReadiness(
      settings({ setupCompleted: true, microphoneID: "missing-mic" }),
      null,
      devices,
      false,
    );
    expect(readiness.show).toBe(true);
    expect(
      readiness.steps.find((step) => step.id === "microphone")?.status,
    ).toBe("attention");
  });
});

it("accepts a catalog-declared server-loaded model without a client model ID", () => {
  const cfg = settings({ model: "", compatibilityProfile: ID.WhisperCPP });
  cfg.compatibilityProfiles.transcription = [
    {
      id: ID.WhisperCPP,
      label: "whisper.cpp",
      description: "Native server",
      available: true,
      capabilities: {
        speechInstructions: false,
        speechLanguage: false,
        realtime: false,
        voiceDiscovery: false,
        serverLoadedModel: true,
        vllmTranscriptionEvents: false,
        cleanupOutputLimit: false,
        cleanupDisableReasoning: false,
        fileStreaming: false,
        typedTranscriptionEvents: false,
        legacyTranscriptionSegments: false,
        languageHint: true,
        speechSpeed: false,
        transcriptionPrompt: true,
        transcriptionHotwords: false,
        transcriptionTemperature: true,
      },
    },
  ];
  const native = appReadiness(cfg, null, devices, false);
  expect(native.steps.find((step) => step.id === "server")?.status).toBe(
    "complete",
  );
  cfg.voiceTranscription.compatibilityProfile = ID.Generic;
  expect(
    appReadiness(cfg, null, devices, false).steps.find(
      (step) => step.id === "server",
    )?.status,
  ).toBe("attention");
});

describe("task-specific prerequisites", () => {
  it("allows file transcription before dictation setup without microphone or shortcut", () => {
    const v = settings({
      setupCompleted: false,
      toggleShortcut: "",
      credentialConfigured: true,
    });
    const file = appReadiness(v, null, [], false, "file");
    expect(file.initialSetup).toBe(false);
    expect(file.steps.map((s) => s.id)).not.toContain("microphone");
    expect(file.steps.map((s) => s.id)).not.toContain("shortcut");
    expect(file.recoveryNeeded).toBe(false);
    const voice = appReadiness(v, null, [], false);
    expect(voice.initialSetup).toBe(true);
    expect(voice.recoveryNeeded).toBe(true);
  });
  it("still requires file transcription endpoint and credentials", () => {
    const file = appReadiness(
      settings({ baseURL: "", credentialConfigured: false }),
      null,
      [],
      false,
      "file",
    );
    expect(file.recoveryNeeded).toBe(true);
    expect(file.steps.filter((s) => s.blocking).map((s) => s.id)).toEqual([
      "server",
      "credential",
    ]);
  });
});
