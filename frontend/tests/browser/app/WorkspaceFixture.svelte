<script lang="ts">
  import { configurePickerFixture } from "./picker-data";
  import { onDestroy } from "svelte";
  import { CancellablePromise } from "@wailsio/runtime";
  import { ID as ModelProfileID } from "$bindings/modelprofile";
  import { modelOptions } from "$lib/utils/modelSettings";
  import {
    TTSPhase,
    TTSSource,
    FileTranscriptionPhase,
    HistoryProcessingStatus,
    ConnectionErrorKind,
  } from "$lib/state";
  import { Purpose } from "$bindings/savedconnection";
  import { AuthenticationMode } from "$bindings/config";
  import { Session } from "$lib/stores/session.svelte";
  import {
    settings,
    idle,
    historyEntry,
    connectionResult,
    serviceWithStatus,
  } from "$lib/stores/session-fixtures-data";
  import HomeScreen from "$lib/components/home/HomeScreen.svelte";
  import AppHeader from "$lib/components/shell/AppHeader.svelte";
  import { controlledSaves } from "./save-control";

  const setupScenario = new URLSearchParams(location.search).get("setup");
  let openedSettings = $state("");
  let connectionChecks = 0;
  let current = structuredClone(settings);
  current.setupCompleted = true;
  current.historyEnabled = new URLSearchParams(location.search).get("history") !== "off";
  current.authenticationMode = AuthenticationMode.AuthenticationModeNone;
  current.postProcessing.model = "cleanup/standard";
  current.voiceTranscription = {
    ...current.voiceTranscription,
    baseURL: current.baseURL,
    compatibilityProfile: current.compatibilityProfile,
    authenticationMode: AuthenticationMode.AuthenticationModeNone,
    model: "speech/stt",
    modelProfile: ModelProfileID.Generic,
    realtime: false,
  };
  current.textToSpeech = {
    ...current.textToSpeech,
    enabled: true,
    baseURL: "https://speech.test/v1",
    model: "speech/tts",
    voice: "Default",
  };
  current.savedConnections.selected = {
    ...current.savedConnections.selected,
    speech: "speech-fixture",
    voice: "voice-fixture",
  };
  current.savedConnections.entries = [
    ...(current.savedConnections.entries ?? []),
    {
      id: "voice-fixture",
      name: "Example transcription server",
      uses: [Purpose.Voice],
      hasCredential: false,
      details: {
        compatibilityProfile: current.compatibilityProfile,
        baseURL: current.baseURL,
        allowInsecureHTTP: false,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    },
    {
      id: "speech-fixture",
      name: "Example speech server",
      uses: [Purpose.Speech],
      hasCredential: false,
      details: {
        compatibilityProfile: current.textToSpeech.compatibilityProfile,
        baseURL: current.textToSpeech.baseURL,
        allowInsecureHTTP: false,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    },
  ];
  current.modelProfiles.speech = [
    {
      id: ModelProfileID.Generic,
      name: "Generic",
      description: "Synthetic speech profile",
      reasoningOffRequired: false,
      capabilities: {
        speechInstructions: false,
        speechLanguage: false,
        realtime: false,
        serverLoadedModel: false,
        vllmTranscriptionEvents: false,
        cleanupOutputLimit: false,
        cleanupDisableReasoning: false,
        fileStreaming: false,
        typedTranscriptionEvents: false,
        legacyTranscriptionSegments: false,
        languageHint: false,
        voiceDiscovery: false,
        speechSpeed: true,
        transcriptionPrompt: false,
        transcriptionHotwords: false,
        transcriptionTemperature: false,
      },
    },
  ];
  current.rememberedModels.defaults = {
    ...current.rememberedModels.defaults,
    speech: modelOptions(current, Purpose.Speech),
    voice: modelOptions(current, Purpose.Voice),
  };
  current.rememberedModels.entries = [
    ...(current.rememberedModels.entries ?? []),
    {
      connectionID: "speech-fixture",
      purpose: Purpose.Speech,
      model: "speech/alternate",
      selected: false,
      options: {
        ...modelOptions(current, Purpose.Speech),
        voice: "Alternate voice",
      },
    },
  ];
  if (new URLSearchParams(location.search).has("pickers")) configurePickerFixture(current);
  if (setupScenario) {
    current.setupCompleted = !["first", "retry", "missing-model", "loading"].includes(
      setupScenario,
    );
    current.historyEnabled = false;
    if (setupScenario === "missing-model") current.voiceTranscription.model = "";
  }
  const saves = controlledSaves((request) => {
    current = structuredClone({ ...current, ...request.settings });
    return structuredClone(current);
  });
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
      connection: {
        TestSavedConnection: () => {
          connectionChecks++;
          return CancellablePromise.resolve({
            ...structuredClone(connectionResult),
            ...(connectionChecks === 1 && ["retry", "connection"].includes(setupScenario ?? "")
              ? {
                  reachable: false,
                  errorKind: ConnectionErrorKind.ConnectionErrorNetwork,
                  httpStatus: 0,
                }
              : {}),
          });
        },
      },
      files: {
        StartFileTranscription: () => {
          session.files.applyStatus({
            ...session.files.status,
            phase: FileTranscriptionPhase.FileTranscriptionUploading,
            canStart: false,
            canCancel: true,
          });
          return CancellablePromise.resolve();
        },
      },
      speech: {
        PlayVoiceTranscript: (generation) => {
          if (generation !== 7)
            return CancellablePromise.reject(new Error("Wrong result generation"));
          session.speech.applyStatus({
            ...session.speech.status,
            generation: 1,
            source: TTSSource.SourceVoice,
            phase: TTSPhase.Completed,
            canRestart: true,
            canClear: true,
          });
          return CancellablePromise.resolve();
        },
      },
      settings: {
        SaveSettings: (request) => saves.save(structuredClone($state.snapshot(request))),
      },
    }),
  );
  session.editor.applySettingsSnapshot(structuredClone(current));
  session.editor.devices = [{ id: "desk-mic", name: "Desk microphone", default: true }];
  session.editor.connection = structuredClone(connectionResult);
  session.dictation.status = { ...idle, generation: 7, transcript: "Testing, testing." };
  session.history.entries = (current.historyEnabled ? [1, 2] : []).map((id) => ({
    ...structuredClone(historyEntry),
    id,
    text: "Testing, testing.",
    rawText: "Testing, testing.",
  }));
  if (new URLSearchParams(location.search).get("history") === "review") {
    session.history.entries[0] = {
      ...session.history.entries[0],
      processingStatus: HistoryProcessingStatus.HistoryProcessingCompleted,
      text: "Testing, testing.",
      rawText: "um testing testing",
      processedText: "Testing, testing.",
    };
    session.history.entries[1] = {
      ...session.history.entries[1],
      processingStatus: HistoryProcessingStatus.HistoryProcessingFailed,
      processingMessage:
        "Cleanup could not finish because the server was unavailable. The raw transcript was kept and is ready to copy.",
    };
  }
  if (new URLSearchParams(location.search).has("file-error")) {
    session.files.applyStatus({
      ...session.files.status,
      generation: 8,
      phase: FileTranscriptionPhase.FileTranscriptionFailed,
      fileName: "Example recording.wav",
      fileSize: 10240,
      canStart: true,
      message:
        "The transcription server did not respond before the request timeout. Check the connection or increase the request timeout in Audio-file transcription settings, then retry this file.",
    });
  }
  if (setupScenario) {
    session.dictation.status = { ...idle, transcript: "" };
    session.editor.connection = null;
    if (setupScenario === "microphone" || setupScenario === "loading") session.editor.devices = [];
    if (setupScenario === "loading") session.editor.devicesBusy = true;
    if (setupScenario === "connection") void session.editor.testVoiceConnection();
  }
  window.testSaves = saves.control;
  onDestroy(() => session.dispose());
  let inputMode = $state("voice");
  const noop = () => {};
</script>

<div class="flex h-screen flex-col overflow-hidden bg-background text-foreground">
  <AppHeader bind:inputMode settings={session.editor.applied} onSettings={noop} />
  <HomeScreen
    {session}
    bind:inputMode
    onOpenHistorySettings={noop}
    onOpenServerSettings={noop}
    onOpenProcessingSettings={noop}
    onOpenAudioSettings={() => (openedSettings = "Audio settings")}
    onOpenShortcutSettings={() => (openedSettings = "Shortcut settings")}
    onOpenSpeechSettings={noop}
    onOpenGeneralSettings={noop}
  />
  <footer
    class="flex h-9 shrink-0 items-center border-t border-hairline bg-layer-fill px-4 text-xs text-muted-foreground"
  >
    {openedSettings || "Transcription: Reachable · Example connection"}
  </footer>
</div>
