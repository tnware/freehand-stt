import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
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
import { Session, type SessionService } from "$lib/stores/session.svelte";

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
  CurrentStatus: SessionService["CurrentStatus"],
  overrides: Partial<SessionService> = {},
): SessionService => ({
  CurrentStatus,
  GetSettings: () => CancellablePromise.resolve(settings),
  GetPostProcessingProfiles: () =>
    CancellablePromise.resolve(processingProfiles),
  RetryConfiguration: () => CancellablePromise.resolve(settings),
  ResetConfiguration: () => CancellablePromise.resolve(settings),
  SaveSettings: () => CancellablePromise.resolve(settings),
  ListMicrophones: () => CancellablePromise.resolve(null),
  TestConnection: () => CancellablePromise.resolve(connectionResult),
  TestPostProcessingConnection: () =>
    CancellablePromise.resolve(connectionResult),
  TestTextToSpeechConnection: () =>
    CancellablePromise.resolve(connectionResult),
  StartRecording: (_mode) => CancellablePromise.resolve(),
  StopRecording: () => CancellablePromise.resolve(),
  Cancel: () => CancellablePromise.resolve(),
  CopyPending: () => CancellablePromise.resolve(),
  TranscriptHistory: () => CancellablePromise.resolve([]),
  CopyHistoryEntry: () => CancellablePromise.resolve(),
  CopyHistoryEntryVersion: () => CancellablePromise.resolve(),
  DeleteHistoryEntry: () => CancellablePromise.resolve(),
  ClearHistory: () => CancellablePromise.resolve(),
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
  CurrentTTSStatus: () =>
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
  PauseTTS: () => CancellablePromise.resolve(),
  ResumeTTS: () => CancellablePromise.resolve(),
  RestartTTS: () => CancellablePromise.resolve(),
  StopTTS: () => CancellablePromise.resolve(),
  SaveTTSAudio: () => CancellablePromise.resolve(false),
  ClearTTSAudio: () => CancellablePromise.resolve(),
  ...overrides,
});

describe("Session status loading", () => {
  it("keeps a live event that arrives while the initial snapshot is pending", async () => {
    const snapshot = CancellablePromise.withResolvers<Status>();
    const session = new Session(serviceWithStatus(() => snapshot.promise));

    const loading = session.load();
    session.applyStatus(recording);
    snapshot.resolve(idle);
    await loading;

    expect(session.status).toEqual(recording);
    expect(session.processingProfiles).toEqual(processingProfiles);
  });

  it("applies the initial snapshot when no event arrives", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(recording)),
    );

    await session.load();

    expect(session.status).toEqual(recording);
  });
});

describe("Session configuration recovery", () => {
  const invalidSettings: Settings = {
    ...settings,
    configuration: {
      recoveryRequired: true,
      errorKind: "invalid_json",
      message: "The settings file contains invalid JSON.",
    },
  };

  it("keeps the recovery state visible when retry still cannot load the file", async () => {
    const RetryConfiguration: SessionService["RetryConfiguration"] = vi.fn(() =>
      CancellablePromise.resolve({
        ...invalidSettings,
        configuration: {
          ...invalidSettings.configuration,
          message: "The saved value for appearanceMode has the wrong type.",
        },
      }),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        GetSettings: () => CancellablePromise.resolve(invalidSettings),
        RetryConfiguration,
      }),
    );

    await session.loadSettings();
    expect(await session.retryConfiguration()).toBe(false);
    expect(session.appliedSettings?.configuration.recoveryRequired).toBe(true);
    expect(session.appliedSettings?.configuration.message).toContain(
      "appearanceMode",
    );
    expect(session.configurationRetrying).toBe(false);
  });

  it("adopts a recovered profile and refreshes dependent snapshots", async () => {
    const RetryConfiguration: SessionService["RetryConfiguration"] = vi.fn(() =>
      CancellablePromise.resolve({
        ...settings,
        model: "restored-model",
        configuration: {
          recoveryRequired: false,
          preservedFields: ["realtime"],
        },
      }),
    );
    const ListMicrophones: SessionService["ListMicrophones"] = vi.fn(() =>
      CancellablePromise.resolve([]),
    );
    const TranscriptHistory: SessionService["TranscriptHistory"] = vi.fn(() =>
      CancellablePromise.resolve([]),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        GetSettings: () => CancellablePromise.resolve(invalidSettings),
        RetryConfiguration,
        ListMicrophones,
        TranscriptHistory,
      }),
    );

    await session.loadSettings();
    expect(await session.retryConfiguration()).toBe(true);
    expect(session.appliedSettings?.model).toBe("restored-model");
    expect(session.appliedSettings?.configuration.preservedFields).toEqual([
      "realtime",
    ]);
    expect(ListMicrophones).toHaveBeenCalledOnce();
    expect(TranscriptHistory).toHaveBeenCalledOnce();
  });

  it("adopts explicit defaults and clears credential drafts after reset", async () => {
    const ResetConfiguration: SessionService["ResetConfiguration"] = vi.fn(() =>
      CancellablePromise.resolve(settings),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        GetSettings: () => CancellablePromise.resolve(invalidSettings),
        ResetConfiguration,
      }),
    );
    await session.loadSettings();
    session.apiKey = "unsaved-secret";
    session.processingAPIKey = "unsaved-processing-secret";

    expect(await session.resetConfiguration()).toBe(true);
    expect(session.appliedSettings?.configuration.recoveryRequired).toBe(false);
    expect(session.apiKey).toBe("");
    expect(session.processingAPIKey).toBe("");
    expect(session.notice).toContain("Settings reset");
  });
});

describe("Session file-transcription ordering", () => {
  it("asks Go to open the native picker without sending a path", async () => {
    const selected = {
      generation: 1,
      phase: FileTranscriptionPhase.FileTranscriptionSelected,
      fileName: "meeting.wav",
      fileSize: 4,
      streaming: false,
      buffered: false,
      streamingUnavailable: false,
      transcriptRevision: 0,
      canStart: true,
      canCancel: false,
      canCopy: false,
    };
    const ChooseAudioFile: SessionService["ChooseAudioFile"] = vi.fn(() =>
      CancellablePromise.resolve(selected),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        ChooseAudioFile,
      }),
    );

    await session.chooseAudioFile();

    expect(ChooseAudioFile).toHaveBeenCalledWith();
    expect(session.fileStatus).toEqual(selected);
  });

  it("ignores a late status from an older file generation", () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    session.applyFileStatus({
      generation: 4,
      phase: FileTranscriptionPhase.FileTranscriptionSelected,
      fileName: "new.wav",
      streaming: false,
      buffered: false,
      streamingUnavailable: false,
      transcriptRevision: 0,
      canStart: true,
      canCancel: false,
      canCopy: false,
    });
    session.applyFileStatus({
      generation: 3,
      phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      fileName: "old.wav",
      transcript: "stale",
      streaming: true,
      buffered: false,
      streamingUnavailable: false,
      transcriptRevision: 1,
      canStart: true,
      canCancel: false,
      canCopy: true,
    });

    expect(session.fileStatus.fileName).toBe("new.wav");
    expect(session.fileStatus.phase).toBe(
      FileTranscriptionPhase.FileTranscriptionSelected,
    );
  });

  it("applies ordered deltas, repairs gaps from a snapshot, and reconciles the final text", async () => {
    const recovered = {
      generation: 4,
      phase: FileTranscriptionPhase.FileTranscriptionStreaming,
      fileName: "meeting.wav",
      transcript: "one two three",
      transcriptRevision: 3,
      streaming: true,
      buffered: false,
      streamingUnavailable: false,
      canStart: false,
      canCancel: true,
      canCopy: false,
    };
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        CurrentFileTranscription: () => CancellablePromise.resolve(recovered),
      }),
    );
    session.applyFileStatus({
      ...recovered,
      transcript: "",
      transcriptRevision: 0,
    });

    expect(
      session.applyFileDelta({ generation: 4, revision: 1, text: "one " }),
    ).toBe("applied");
    expect(
      session.applyFileDelta({ generation: 4, revision: 1, text: "one " }),
    ).toBe("ignored");
    expect(
      session.applyFileDelta({ generation: 3, revision: 2, text: "stale " }),
    ).toBe("ignored");
    expect(
      session.applyFileDelta({ generation: 4, revision: 3, text: "three" }),
    ).toBe("gap");
    expect(session.fileStatus.transcript).toBe("one ");

    await session.refreshFileStatus();
    expect(session.fileStatus.transcript).toBe("one two three");
    expect(session.fileStatus.transcriptRevision).toBe(3);
    expect(
      session.applyFileDelta({ generation: 4, revision: 3, text: "three" }),
    ).toBe("ignored");

    session.applyFileStatus({
      ...recovered,
      phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      transcript: "authoritative final transcript",
      transcriptRevision: 4,
      canStart: true,
      canCancel: false,
      canCopy: true,
    });
    expect(
      session.applyFileDelta({ generation: 4, revision: 5, text: "late" }),
    ).toBe("ignored");
    expect(session.fileStatus.transcript).toBe(
      "authoritative final transcript",
    );
  });

  it("resets the endpoint streaming capability through the backend", async () => {
    const TryFileStreamingAgain: SessionService["TryFileStreamingAgain"] =
      vi.fn(() => CancellablePromise.resolve());
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TryFileStreamingAgain,
      }),
    );

    await session.tryFileStreamingAgain();

    expect(TryFileStreamingAgain).toHaveBeenCalledOnce();
    expect(session.notice).toContain("Streaming can be tried again");
  });
});

describe("Session credential draft", () => {
  it("clears the key and pending deletion choice together", () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    session.apiKey = "temporary-secret";
    session.clearKey = true;

    session.clearCredentialDraft();

    expect(session.apiKey).toBe("");
    expect(session.clearKey).toBe(false);
  });
});

describe("Session settings snapshots", () => {
  it("adopts a backend settings event when this renderer has no draft", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    await session.load();

    const changed = {
      ...settings,
      model: "speech/updated",
      postProcessing: { ...settings.postProcessing },
    };
    expect(session.applySettingsSnapshot(changed)).toBe(true);
    expect(session.appliedSettings?.model).toBe("speech/updated");
    expect(session.settings?.model).toBe("speech/updated");
    expect(session.connection).toBeNull();
  });

  it("does not overwrite an active settings-window draft", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    await session.load();
    if (!session.settings) throw new Error("expected settings");
    session.settings.model = "speech/draft";

    const changed = {
      ...settings,
      model: "speech/elsewhere",
      postProcessing: { ...settings.postProcessing },
    };
    expect(session.applySettingsSnapshot(changed)).toBe(false);
    expect(session.settings.model).toBe("speech/draft");
    expect(session.appliedSettings?.model).toBe("speech/stt");
    expect(session.info).toContain("another window");
  });

  it("tracks and discards unsaved settings and credential drafts", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );

    expect(session.settingsDirty).toBe(false);
    await session.load();
    expect(session.settingsDirty).toBe(false);
    if (!session.settings) throw new Error("expected settings");

    session.settings.baseURL = "https://draft.example/v1";
    session.apiKey = "temporary-secret";
    expect(session.settingsDirty).toBe(true);

    session.discardSettingsDraft();
    expect(session.settingsDirty).toBe(false);
    expect(session.settings.baseURL).toBe("https://example.test/v1");
    expect(session.apiKey).toBe("");
  });

  it("keeps draft fields and nested headers independent from applied settings", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );

    await session.load();
    if (!session.settings?.headers)
      throw new Error("expected settings headers");
    session.settings.baseURL = "https://draft.example/v1";
    session.settings.holdShortcut = "Ctrl+Alt+Space";
    session.settings.headers["X-Test"] = "draft";

    expect(session.appliedSettings).toMatchObject({
      baseURL: "https://example.test/v1",
      headers: { "X-Test": "applied" },
    });
    expect(session.appliedSettings?.holdShortcut).toBeUndefined();
  });

  it("replaces both snapshots after Go confirms a successful save", async () => {
    const SaveSettings: SessionService["SaveSettings"] = ({
      settings: draft,
    }) =>
      CancellablePromise.resolve({
        ...settings,
        ...draft,
        baseURL: "https://confirmed.example/v1",
        headers: draft.headers == null ? draft.headers : { ...draft.headers },
      });
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings,
      }),
    );

    await session.load();
    if (!session.settings?.headers)
      throw new Error("expected settings headers");
    session.settings.baseURL = "https://draft.example/v1";
    session.settings.overlayEnabled = false;
    session.settings.overlaySizePercent = 125;
    session.settings.overlayOpacityPercent = 80;
    session.settings.overlayTopOffset = 42;
    session.settings.overlayGlowPercent = 50;
    session.settings.headers["X-Test"] = "saved";
    await session.save();

    expect(session.settings?.baseURL).toBe("https://confirmed.example/v1");
    expect(session.appliedSettings?.baseURL).toBe(
      "https://confirmed.example/v1",
    );
    expect(session.appliedSettings?.headers).toEqual({ "X-Test": "saved" });
    expect(session.appliedSettings?.overlayEnabled).toBe(false);
    expect(session.appliedSettings?.overlaySizePercent).toBe(125);
    expect(session.appliedSettings?.overlayOpacityPercent).toBe(80);
    expect(session.appliedSettings?.overlayTopOffset).toBe(42);
    expect(session.appliedSettings?.overlayGlowPercent).toBe(50);
    expect(session.settings?.headers).not.toBe(
      session.appliedSettings?.headers,
    );
  });

  it("persists completion of the one-time setup without sending credential drafts", async () => {
    const SaveSettings: SessionService["SaveSettings"] = vi.fn(
      ({ settings: draft }) =>
        CancellablePromise.resolve({ ...draft, setupCompleted: true }),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings,
      }),
    );
    await session.load();

    expect(await session.completeSetup()).toBe(true);
    expect(SaveSettings).toHaveBeenCalledWith({
      settings: expect.objectContaining({ setupCompleted: true }),
      sttCredentialDraft: "",
      clearSTTCredential: false,
      postProcessingCredentialDraft: "",
      clearPostProcessingCredential: false,
      textToSpeechCredentialDraft: "",
      clearTextToSpeechCredential: false,
    });
    expect(session.appliedSettings?.setupCompleted).toBe(true);
    expect(session.settings?.setupCompleted).toBe(true);
    expect(session.notice).toBe("Freehand is ready to use.");
  });

  it("keeps the launch material active and requests a restart after changing Mica", async () => {
    const SaveSettings: SessionService["SaveSettings"] = ({
      settings: draft,
    }) =>
      CancellablePromise.resolve({
        ...settings,
        ...draft,
        useMica: true,
        micaActive: false,
      });
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings,
      }),
    );

    await session.load();
    if (!session.settings) throw new Error("expected settings");
    session.settings.useMica = true;
    await session.save();

    expect(session.appliedSettings?.useMica).toBe(true);
    expect(session.appliedSettings?.micaActive).toBe(false);
    expect(session.notice).toContain("Restart the app");
  });

  it("keeps the applied snapshot unchanged when saving fails", async () => {
    const response = CancellablePromise.withResolvers<Settings>();
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings: () => response.promise,
      }),
    );

    await session.load();
    if (!session.settings) throw new Error("expected settings");
    session.settings.baseURL = "https://rejected.example/v1";
    const saving = session.save();
    response.reject(new Error("save failed"));
    await saving;

    expect(session.settings.baseURL).toBe("https://rejected.example/v1");
    expect(session.appliedSettings?.baseURL).toBe("https://example.test/v1");
  });

  it("applies quick settings from the confirmed snapshot instead of an unrelated draft", async () => {
    let received: Parameters<SessionService["SaveSettings"]>[0] | undefined;
    const SaveSettings: SessionService["SaveSettings"] = vi.fn((request) => {
      received = request;
      return CancellablePromise.resolve({ ...settings, ...request.settings });
    });
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings,
      }),
    );

    await session.load();
    if (!session.settings) throw new Error("expected settings");
    session.settings.autoInsert = false;
    session.connection = connectionResult;
    session.processingConnection = connectionResult;

    const saved = await session.updateQuickSettings(
      {
        model: "speech/faster",
        postProcessing: { model: "processor/faster", styling: "formal" },
      },
      "stt-model",
    );

    expect(saved).toBe(true);
    expect(received?.settings.autoInsert).toBe(true);
    expect(received?.settings.model).toBe("speech/faster");
    expect(received?.settings.postProcessing.model).toBe("processor/faster");
    expect(received?.settings.postProcessing.styling).toBe("formal");
    expect(SaveSettings).toHaveBeenCalledWith({
      settings: expect.any(Object),
      sttCredentialDraft: "",
      clearSTTCredential: false,
      postProcessingCredentialDraft: "",
      clearPostProcessingCredential: false,
      textToSpeechCredentialDraft: "",
      clearTextToSpeechCredential: false,
    });
    expect(session.settings.autoInsert).toBe(true);
    expect(session.connection).toBeNull();
    expect(session.processingConnection).toBeNull();
    expect(session.sttConnectionStale).toBe(true);
    expect(session.processingConnectionStale).toBe(true);
    expect(session.quickSettingsSaved).toBe("stt-model");

    await session.testConnection(session.appliedSettings);
    await session.testPostProcessingConnection(session.appliedSettings);
    expect(session.sttConnectionStale).toBe(false);
    expect(session.processingConnectionStale).toBe(false);
  });

  it("keeps applied settings unchanged when a quick update fails", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings: () =>
          CancellablePromise.reject(new Error("quick save failed")),
      }),
    );

    await session.load();
    const saved = await session.updateQuickSettings(
      { model: "rejected-model" },
      "stt-model",
    );

    expect(saved).toBe(false);
    expect(session.appliedSettings?.model).toBe("speech/stt");
    expect(session.error).toContain("quick save failed");
  });

  it("persists the compact quick controls through the confirmed settings snapshot", async () => {
    let received: Parameters<SessionService["SaveSettings"]>[0] | undefined;
    const SaveSettings: SessionService["SaveSettings"] = vi.fn((request) => {
      received = request;
      return CancellablePromise.resolve(request.settings);
    });
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings,
      }),
    );
    await session.load();

    expect(
      await session.updateQuickSettings(
        {
          microphoneID: "usb-mic",
          vadEnabled: false,
          silenceTrimming: false,
          autoStopEnabled: false,
          silenceSplitting: false,
          maxDurationSeconds: 262,
          autoInsert: false,
          historyEnabled: true,
          overlayEnabled: false,
          postProcessing: { enabled: true },
        },
        "vad-enabled",
      ),
    ).toBe(true);

    expect(received?.settings).toMatchObject({
      microphoneID: "usb-mic",
      vadEnabled: false,
      silenceTrimming: false,
      autoStopEnabled: false,
      silenceSplitting: false,
      maxDurationSeconds: 262,
      autoInsert: false,
      historyEnabled: true,
      overlayEnabled: false,
      postProcessing: { enabled: true },
    });
    expect(session.microphoneChoice).toBe("usb-mic");
    expect(session.quickSettingsSaved).toBe("vad-enabled");
  });

  it("persists a processing behavior without changing its stored profile-specific values", async () => {
    let received: Parameters<SessionService["SaveSettings"]>[0] | undefined;
    const SaveSettings: SessionService["SaveSettings"] = vi.fn((request) => {
      received = request;
      return CancellablePromise.resolve({ ...settings, ...request.settings });
    });
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings,
      }),
    );
    await session.load();

    expect(
      await session.updateQuickSettings(
        {
          postProcessing: {
            preset: PostProcessingPreset.PostProcessingPresetS1Mini,
          },
        },
        "processing-profile",
      ),
    ).toBe(true);

    expect(received?.settings.postProcessing).toMatchObject({
      preset: PostProcessingPreset.PostProcessingPresetS1Mini,
      systemPrompt: "Clean the transcript.",
      styling: "semi-casual",
      structure: "prose",
      context: "general",
    });
    expect(session.quickSettingsSaved).toBe("processing-profile");
  });

  it("serializes quick changes so each save starts from the latest confirmed snapshot", async () => {
    const requests: Settings[] = [];
    const first = CancellablePromise.withResolvers<Settings>();
    const SaveSettings: SessionService["SaveSettings"] = vi.fn((request) => {
      requests.push(request.settings);
      if (requests.length === 1) return first.promise;
      return CancellablePromise.resolve(request.settings);
    });
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        SaveSettings,
      }),
    );
    await session.load();

    const endpoint = session.updateQuickSettings(
      { baseURL: "https://new.example/v1" },
      "stt-endpoint",
    );
    const model = session.updateQuickSettings(
      { model: "speech/new" },
      "stt-model",
    );
    await Promise.resolve();
    first.resolve({ ...settings, baseURL: "https://new.example/v1" });

    expect(await endpoint).toBe(true);
    expect(await model).toBe(true);
    expect(requests).toHaveLength(2);
    expect(requests[1]).toMatchObject({
      baseURL: "https://new.example/v1",
      model: "speech/new",
    });
  });
});

describe("Session microphone inventory", () => {
  it("refreshes devices without rewriting the selected microphone", async () => {
    let request = 0;
    const ListMicrophones: SessionService["ListMicrophones"] = vi.fn(() =>
      CancellablePromise.resolve(
        request++ === 0
          ? [
              { id: "", name: "System default microphone", default: true },
              { id: "usb-mic", name: "USB microphone", default: false },
            ]
          : [{ id: "webcam-mic", name: "Webcam microphone", default: true }],
      ),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        ListMicrophones,
      }),
    );
    await session.load();
    session.chooseMicrophone("usb-mic");

    await session.refreshDevices();

    expect(session.devices).toEqual([
      { id: "webcam-mic", name: "Webcam microphone", default: true },
    ]);
    expect(session.microphoneChoice).toBe("usb-mic");
    expect(session.settings?.microphoneID).toBe("usb-mic");
  });

  it("debounces repeated refreshes while enumeration is pending", async () => {
    const response = CancellablePromise.withResolvers<
      { id: string; name: string; default: boolean }[] | null
    >();
    const ListMicrophones: SessionService["ListMicrophones"] = vi.fn(
      () => response.promise,
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        ListMicrophones,
      }),
    );

    const first = session.refreshDevices();
    await session.refreshDevices();
    response.resolve([
      { id: "usb-mic", name: "USB microphone", default: true },
    ]);
    await first;

    expect(ListMicrophones).toHaveBeenCalledOnce();
    expect(session.devices).toHaveLength(1);
    expect(session.devicesBusy).toBe(false);
  });
});

describe("Session connection metadata", () => {
  it("keeps the structured result for the settings-window lifetime", async () => {
    const TestConnection: SessionService["TestConnection"] = vi.fn(() =>
      CancellablePromise.resolve(connectionResult),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TestConnection,
      }),
    );
    await session.load();

    await session.testConnection();

    expect(TestConnection).toHaveBeenCalledWith({
      baseURL: "https://example.test/v1",
      allowInsecureHTTP: false,
      authenticationMode: AuthenticationMode.AuthenticationModeAPIKey,
      model: "speech/stt",
      healthPath: "",
      headers: { "X-Test": "applied" },
      credentialDraft: "",
    });
    expect(session.connection).toEqual(connectionResult);
    expect(session.notice).toBe("");
  });

  it("debounces a repeated check while one is in flight", async () => {
    const response = CancellablePromise.withResolvers<ConnectionResult>();
    const TestConnection: SessionService["TestConnection"] = vi.fn(
      () => response.promise,
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TestConnection,
      }),
    );
    await session.load();

    const first = session.testConnection();
    await session.testConnection();
    response.resolve(connectionResult);
    await first;

    expect(TestConnection).toHaveBeenCalledOnce();
    expect(session.connection).toEqual(connectionResult);
  });

  it("does not clear an existing confirmation during an automatic check", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    await session.load();
    session.notice = "Settings saved and active.";

    await session.testConnection(session.appliedSettings, "", false);

    expect(session.connection).toEqual(connectionResult);
    expect(session.notice).toBe("Settings saved and active.");
  });

  it("sends only focused post-processing probe values", async () => {
    const TestPostProcessingConnection: SessionService["TestPostProcessingConnection"] =
      vi.fn(() => CancellablePromise.resolve(connectionResult));
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TestPostProcessingConnection,
      }),
    );
    await session.load();
    if (!session.settings) throw new Error("expected settings");
    session.settings.postProcessing.model = "processor/s1-mini";
    session.settings.postProcessing.systemPrompt = "unrelated unsaved prompt";

    await session.testPostProcessingConnection();

    expect(TestPostProcessingConnection).toHaveBeenCalledWith({
      baseURL: "http://127.0.0.1:8080/v1",
      allowInsecureHTTP: false,
      model: "processor/s1-mini",
      credentialDraft: "",
    });
  });

  it("discovers speech playback models with its dedicated connection values", async () => {
    const TestTextToSpeechConnection: SessionService["TestTextToSpeechConnection"] =
      vi.fn(() => CancellablePromise.resolve(connectionResult));
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TestTextToSpeechConnection,
      }),
    );
    await session.load();
    if (!session.settings) throw new Error("expected settings");
    session.settings.textToSpeech.baseURL = "http://127.0.0.1:8000/v1";
    session.settings.textToSpeech.allowInsecureHTTP = true;
    session.settings.textToSpeech.authenticationMode =
      AuthenticationMode.AuthenticationModeNone;
    session.settings.textToSpeech.model = "local/kokoro";
    session.settings.textToSpeech.voice = "unrelated-voice";

    await session.testTextToSpeechConnection();

    expect(TestTextToSpeechConnection).toHaveBeenCalledWith({
      baseURL: "http://127.0.0.1:8000/v1",
      allowInsecureHTTP: true,
      authenticationMode: AuthenticationMode.AuthenticationModeNone,
      model: "local/kokoro",
      credentialDraft: "",
    });
    expect(session.ttsConnection).toEqual(connectionResult);
  });
});

describe("Session transcript history", () => {
  it("loads history from its dedicated binding", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TranscriptHistory: () => CancellablePromise.resolve([historyEntry]),
      }),
    );

    await session.load();

    expect(session.history).toEqual([historyEntry]);
  });

  it("does not let an older refresh overwrite a newer history result", async () => {
    const older = CancellablePromise.withResolvers<HistoryEntry[] | null>();
    let request = 0;
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TranscriptHistory: () => {
          request++;
          return request === 1
            ? older.promise
            : CancellablePromise.resolve([historyEntry]);
        },
      }),
    );

    const staleRefresh = session.refreshHistory();
    await session.refreshHistory();
    older.resolve([]);
    await staleRefresh;

    expect(session.history).toEqual([historyEntry]);
  });

  it("marks a completed file generation synchronized only after history refreshes", async () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        TranscriptHistory: () => CancellablePromise.resolve([historyEntry]),
      }),
    );
    session.applyFileStatus({
      generation: 7,
      phase: FileTranscriptionPhase.FileTranscriptionCompleted,
      fileName: "recording.mp3",
      transcript: historyEntry.text,
      streaming: true,
      buffered: false,
      streamingUnavailable: false,
      transcriptRevision: 1,
      canStart: true,
      canCancel: false,
      canCopy: true,
    });

    expect(session.fileHistoryGeneration).toBe(0);
    await session.refreshHistory();

    expect(session.fileHistoryGeneration).toBe(7);
    expect(session.history).toEqual([historyEntry]);
  });

  it("copies, removes, and clears only through explicit history actions", async () => {
    const CopyHistoryEntry: SessionService["CopyHistoryEntry"] = vi.fn(() =>
      CancellablePromise.resolve(),
    );
    const ClearHistory: SessionService["ClearHistory"] = vi.fn(() =>
      CancellablePromise.resolve(),
    );
    const DeleteHistoryEntry: SessionService["DeleteHistoryEntry"] = vi.fn(() =>
      CancellablePromise.resolve(),
    );
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        CopyHistoryEntry,
        DeleteHistoryEntry,
        ClearHistory,
      }),
    );
    session.history = [historyEntry];

    await session.copyHistoryEntry(historyEntry.id);
    expect(CopyHistoryEntry).toHaveBeenCalledWith(historyEntry.id);
    expect(session.history).toEqual([historyEntry]);

    await session.deleteHistoryEntry(historyEntry.id);
    expect(DeleteHistoryEntry).toHaveBeenCalledWith(historyEntry.id);
    expect(session.history).toEqual([]);

    session.history = [historyEntry];

    await session.clearHistory();
    expect(ClearHistory).toHaveBeenCalledOnce();
    expect(session.history).toEqual([]);
  });
});

describe("Session renderer failures", () => {
  it("uses the visible error channel and clears a stale notice", () => {
    const session = new Session(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    session.notice = "Earlier confirmation";

    session.reportFailure("Unable to close the window.");

    expect(session.notice).toBe("");
    expect(session.error).toBe("Unable to close the window.");
  });
});
