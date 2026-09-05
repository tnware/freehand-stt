import { ID } from "$bindings/compatibility";
import { CancellablePromise } from "@wailsio/runtime";
import { vi } from "vitest";
import {
  HistoryOutcome,
  HistoryProcessingStatus,
  HistoryResponseMode,
  HistorySource,
  InsertionMode,
  FileTranscriptionPhase,
  ConnectionErrorKind,
  ConnectionProbe,
  ModelPresence,
  State,
  AppearanceMode,
  AuthenticationMode,
  OverlayAnchor,
  OverlayLayout,
  OverlayMotion,
  OverlaySurface,
  OverlayVisibility,
  OverlayVisualizer,
  VADMode,
  PostProcessingPreset,
  TTSPhase,
  type ConnectionResult,
  type HistoryEntry,
  type ProfileDescriptor,
  type Settings,
  type Status,
} from "$lib/state";
import { Session, type SessionServices } from "$lib/stores/session.svelte";

vi.mock("$bindings/connection/service", () => ({}));
vi.mock("$bindings/dictation/service", () => ({}));
vi.mock("$bindings/filetranscription/service", () => ({}));
vi.mock("$bindings/history/service", () => ({}));
vi.mock("$bindings/input/service", () => ({}));
vi.mock("$bindings/settings/service", () => ({}));
vi.mock("$bindings/tts/service", () => ({
  CurrentStatus: vi.fn(),
  PlayHistoryEntry: vi.fn(),
  PlayFileTranscript: vi.fn(),
  PreviewVoice: vi.fn(),
  SpeakText: vi.fn(),
  Pause: vi.fn(),
  Resume: vi.fn(),
  Restart: vi.fn(),
  Stop: vi.fn(),
  SaveAudio: vi.fn(),
  ClearAudio: vi.fn(),
}));

const settings: Settings = {
  compatibilityProfile: ID.Generic,
  compatibilityProfiles: { transcription: [], postProcessing: [], speech: [] },
  baseURL: "https://example.test/v1",
  allowInsecureHTTP: false,
  authenticationMode: AuthenticationMode.AuthenticationModeAPIKey,
  model: "speech/stt",
  headers: { "X-Test": "applied" },
  toggleShortcut: "Ctrl+Shift+Space",
  showShortcut: "Ctrl+Shift+D",
  maxDurationSeconds: 120,
  transcriptionTimeoutSeconds: 120,
  fileTranscriptionTimeoutSeconds: 21600,
  autoInsert: true,
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
    compatibilityProfile: ID.Generic,
    enabled: false,
    baseURL: "http://127.0.0.1:8080/v1",
    allowInsecureHTTP: false,
    model: "",
    preset: PostProcessingPreset.PostProcessingPresetGeneric,
    systemPrompt: "Clean the transcript.",
    styling: "semi-casual",
    structure: "prose",
    context: "general",
    timeoutSeconds: 120,
  },
  textToSpeech: {
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
    preservedFields: [],
  },
  credentialConfigured: false,
  postProcessingCredentialConfigured: false,
  textToSpeechCredentialConfigured: false,
  holdAvailable: true,
  holdAvailabilityReason: "",
  micaActive: false,
  appearanceModeActive: AppearanceMode.AppearanceModeSystem,
};

const processingProfiles: ProfileDescriptor[] = [
  {
    id: PostProcessingPreset.PostProcessingPresetGeneric,
    name: "Custom instruction",
    description: "Use any OpenAI-compatible chat model.",
    instructionEditable: true,
    recommendedInstruction: "Clean the transcript.",
    maximumInstructionBytes: 8192,
  },
  {
    id: PostProcessingPreset.PostProcessingPresetS1Mini,
    name: "S1-mini by Superwhisper",
    description: "Use the fixed S1-mini normalization contract.",
    instructionEditable: false,
    systemInstruction: "Built-in S1-mini instruction.",
    controls: {
      styling: ["casual", "semi-casual", "semi-formal", "formal"],
      structure: ["prose", "lists"],
      context: ["general", "email"],
    },
  },
];

const idle: Status = {
  state: State.Idle,
  generation: 0,
  canCancel: false,
  canCopy: false,
};

const recording: Status = {
  state: State.Recording,
  generation: 1,
  canCancel: true,
  canCopy: false,
};

const historyEntry: HistoryEntry = {
  id: 1,
  text: "Recovered transcript",
  rawText: "Recovered transcript",
  completedAt: "2026-08-30T14:00:00Z",
  characterCount: 20,
  outcome: HistoryOutcome.HistoryCopyRequired,
  processingStatus: HistoryProcessingStatus.HistoryProcessingNotRequested,
  details: {
    source: HistorySource.HistorySourceVoice,
    startedAt: "2026-08-30T13:59:55Z",
    completedAt: "2026-08-30T14:00:00Z",
    elapsedMilliseconds: 5000,
    server: "https://example.test",
    route: "/audio/transcriptions",
    authenticationMode: "api-key",
    model: "speech/stt",
    responseMode: HistoryResponseMode.HistoryResponseCompleted,
    insertionMode: InsertionMode.DirectInput,
    buffered: false,
    vadEnabled: true,
    vadMode: "aggressive",
    speechPaddingMilliseconds: 300,
    silenceTrimming: false,
    autoStopEnabled: false,
    autoStopActive: false,
    autoStopped: false,
    silenceSplitting: false,
    segmentsTruncated: false,
    durationLimitReached: false,
    processing: {
      requested: false,
      status: HistoryProcessingStatus.HistoryProcessingNotRequested,
      deliveredCharacters: 20,
    },
  },
};

const connectionResult: ConnectionResult = {
  reachable: true,
  probe: ConnectionProbe.ConnectionProbeModels,
  requestedURL: "https://example.test/v1/models",
  httpStatus: 200,
  latencyMilliseconds: 42,
  errorKind: ConnectionErrorKind.$zero,
  checkedAt: "2026-08-30T15:00:00Z",
  modelPresence: ModelPresence.ModelPresenceListed,
  modelIDs: ["speech/stt", "speech/alternate"],
};

const serviceWithStatus = (
  CurrentStatus: SessionServices["dictation"]["CurrentStatus"],
  overrides: {
    [K in keyof SessionServices]?: Partial<SessionServices[K]>;
  } = {},
): SessionServices => ({
  dictation: {
    CurrentStatus,
    StartRecording: (_mode) => CancellablePromise.resolve(),
    StopRecording: () => CancellablePromise.resolve(),
    Cancel: () => CancellablePromise.resolve(),
    CopyPending: () => CancellablePromise.resolve(),
    ...overrides.dictation,
  },
  settings: {
    GetSettings: () => CancellablePromise.resolve(settings),
    GetPostProcessingProfiles: () =>
      CancellablePromise.resolve(processingProfiles),
    RetryConfiguration: () => CancellablePromise.resolve(settings),
    ResetConfiguration: () => CancellablePromise.resolve(settings),
    SaveSettings: () => CancellablePromise.resolve(settings),
    ...overrides.settings,
  },
  input: {
    ListMicrophones: () => CancellablePromise.resolve(null),
    ...overrides.input,
  },
  connection: {
    TestConnection: () => CancellablePromise.resolve(connectionResult),
    TestPostProcessingConnection: () =>
      CancellablePromise.resolve(connectionResult),
    TestTextToSpeechConnection: () =>
      CancellablePromise.resolve(connectionResult),
    ...overrides.connection,
  },
  history: {
    TranscriptHistory: () => CancellablePromise.resolve([]),
    CopyHistoryEntry: () => CancellablePromise.resolve(),
    CopyHistoryEntryVersion: () => CancellablePromise.resolve(),
    DeleteHistoryEntry: () => CancellablePromise.resolve(),
    ClearHistory: () => CancellablePromise.resolve(),
    ...overrides.history,
  },
  files: {
    CurrentFileTranscription: () =>
      CancellablePromise.resolve({
        generation: 0,
        phase: FileTranscriptionPhase.FileTranscriptionEmpty,
        streaming: false,
        buffered: false,
        streamingUnavailable: false,
        transcriptRevision: 0,
        canStart: false,
        canCancel: false,
        canCopy: false,
      }),
    ChooseAudioFile: () => CancellablePromise.reject(new Error("not used")),
    ClearAudioFile: () => CancellablePromise.resolve(),
    StartFileTranscription: () => CancellablePromise.resolve(),
    TryFileStreamingAgain: () => CancellablePromise.resolve(),
    CancelFileTranscription: () => CancellablePromise.resolve(),
    CopyFileTranscript: () => CancellablePromise.resolve(),
    ...overrides.files,
  },
  speech: {
    CurrentStatus: () =>
      CancellablePromise.resolve({
        generation: 0,
        phase: TTSPhase.Idle,
        positionMilliseconds: 0,
        durationMilliseconds: 0,
        canPause: false,
        canResume: false,
        canRestart: false,
        canStop: false,
        canSave: false,
        canClear: false,
      }),
    PlayHistoryEntry: () => CancellablePromise.resolve(),
    PlayFileTranscript: () => CancellablePromise.resolve(),
    PreviewVoice: () => CancellablePromise.resolve(),
    SpeakText: () => CancellablePromise.resolve(),
    Pause: () => CancellablePromise.resolve(),
    Resume: () => CancellablePromise.resolve(),
    Restart: () => CancellablePromise.resolve(),
    Stop: () => CancellablePromise.resolve(),
    SaveAudio: () => CancellablePromise.resolve(false),
    ClearAudio: () => CancellablePromise.resolve(),
    ...overrides.speech,
  },
});

export { Session } from "./session.svelte";
export {
  settings,
  processingProfiles,
  idle,
  recording,
  historyEntry,
  connectionResult,
  serviceWithStatus,
};

import { SessionMessages } from "./messages.svelte";
import { SettingsEditor } from "./editor.svelte";
import { FileTranscriptionState } from "./files.svelte";

export function createEditor(bindings: SessionServices) {
  const messages = new SessionMessages();
  const editor = new SettingsEditor(bindings, messages, async () => {
    await bindings.history.TranscriptHistory();
  });
  return { editor, messages };
}

export function createFiles(bindings: SessionServices) {
  const messages = new SessionMessages();
  return {
    files: new FileTranscriptionState(bindings.files, messages),
    messages,
  };
}
