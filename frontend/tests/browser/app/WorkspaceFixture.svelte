<script lang="ts">
  import StatusStrip from "$lib/components/shell/StatusStrip.svelte";
  import { taskConnectionDetails, taskConnectionStatus } from "$lib/utils/connection";
  import { configurePickerFixture } from "./picker-data";
  import { onDestroy } from "svelte";
  import { CancellablePromise } from "@wailsio/runtime";
  import { ID as ModelProfileID } from "$bindings/modelprofile";
  import { modelOptions } from "$lib/utils/modelSettings";
  import {
    TTSPhase,
    State,
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

  const playbackScenario = new URLSearchParams(location.search).has("playback");
  const fileStreamingScenario = new URLSearchParams(location.search).has("file-streaming");
  let fileRequests = $state(0);
  let lastFileStream = $state(false);
  let seekCalls = $state(0);
  const historyExpansion = new URLSearchParams(location.search).get("history") === "expansion";
  const diagnosticsScenario = new URLSearchParams(location.search).get("diagnostics");
  const setupScenario = new URLSearchParams(location.search).get("setup");
  let openedSettings = $state("");
  let connectionChecks = 0;
  const feedbackScenario = new URLSearchParams(location.search).has("feedback");
  let speechRequests = $state(0);
  let audioSaves = 0;
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
  if (diagnosticsScenario) {
    current.baseURL = "https://files.test/v1";
    current.savedConnections.selected[Purpose.Transcription] = "file-fixture";
    current.savedConnections.entries!.push({
      id: "file-fixture",
      name: "Example file server",
      uses: [Purpose.Transcription],
      hasCredential: false,
      details: {
        compatibilityProfile: current.compatibilityProfile,
        baseURL: current.baseURL,
        allowInsecureHTTP: false,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    });
  }
  if (diagnosticsScenario === "off") current.textToSpeech.enabled = false;
  if (diagnosticsScenario === "missing") current.savedConnections.selected.voice = "";
  const saves = controlledSaves((request) => {
    current = structuredClone({ ...current, ...request.settings });
    return structuredClone(current);
  });
  const session: Session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
      connection: {
        TestSavedConnection: () => {
          connectionChecks++;
          return CancellablePromise.resolve({
            ...structuredClone(connectionResult),
            ...(connectionChecks === 1 &&
            (["retry", "connection"].includes(setupScenario ?? "") ||
              diagnosticsScenario === "failure")
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
        TryFileStreamingAgain: () => {
          session.files.applyStatus({ ...session.files.status, streamingUnavailable: false });
          return CancellablePromise.resolve();
        },
        StartFileTranscription: (stream) => {
          fileRequests++;
          lastFileStream = stream;
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
        Seek: (request) => {
          seekCalls++;
          const status = session.speech.status;
          if (request.generation !== status.generation)
            return CancellablePromise.reject(new Error("stale seek"));
          const atEnd = request.positionMilliseconds === status.durationMilliseconds;
          const playing = status.phase === TTSPhase.Playing && !atEnd;
          const next = {
            ...status,
            positionMilliseconds: request.positionMilliseconds,
            phase: atEnd ? TTSPhase.Completed : playing ? TTSPhase.Playing : TTSPhase.Paused,
            canPause: playing,
            canResume: !playing && !atEnd,
            canStop: !atEnd,
          };
          session.speech.applyStatus(next);
          return CancellablePromise.resolve(next);
        },
        Pause: () => {
          session.speech.applyStatus({
            ...session.speech.status,
            phase: TTSPhase.Paused,
            canPause: false,
            canResume: true,
          });
          return CancellablePromise.resolve();
        },
        Resume: () => {
          session.speech.applyStatus({
            ...session.speech.status,
            phase: TTSPhase.Playing,
            canPause: true,
            canResume: false,
          });
          return CancellablePromise.resolve();
        },
        SpeakText: () => {
          speechRequests++;
          if (playbackScenario) {
            session.speech.applyStatus({
              ...session.speech.status,
              generation: session.speech.status.generation + 1,
              source: TTSSource.SourceCompose,
              phase: TTSPhase.Generating,
              durationMilliseconds: 0,
              positionMilliseconds: 0,
              canSeek: false,
              canPause: false,
              canResume: false,
              canRestart: false,
              canSave: false,
              canStop: true,
              canClear: false,
            });
            return CancellablePromise.resolve();
          }
          session.speech.applyStatus({
            ...session.speech.status,
            generation: speechRequests,
            source: TTSSource.SourceCompose,
            phase: speechRequests === 1 && feedbackScenario ? TTSPhase.Failed : TTSPhase.Completed,
            message:
              speechRequests === 1 && feedbackScenario
                ? "The speech service could not complete this request. Check the connection and selected voice, then try again. ".repeat(
                    5,
                  )
                : "",
            canClear: true,
            canSave: speechRequests > 1,
            canRestart: speechRequests > 1,
          });
          return CancellablePromise.resolve();
        },
        SaveAudio: () => {
          audioSaves++;
          return feedbackScenario && audioSaves === 1
            ? CancellablePromise.reject(
                new Error(
                  "The audio file could not be saved. Choose another folder and try again. ".repeat(
                    8,
                  ),
                ),
              )
            : CancellablePromise.resolve(true);
        },
        ClearAudio: () => {
          session.speech.applyStatus({
            ...session.speech.status,
            phase: TTSPhase.Idle,
            canClear: false,
            canSave: false,
            canRestart: false,
          });
          return CancellablePromise.resolve();
        },
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
  const sampleTranscript =
    "The first paragraph stays readable in full without opening the latest transcript.\n\nThe second paragraph preserves the original spacing and continues beyond a compact history preview.\n\nThe final paragraph is visible too, while older results take less space.";
  let nextHistoryID = 3;
  if (historyExpansion) {
    session.history.entries = [2, 1].map((id) => ({
      ...structuredClone(historyEntry),
      id,
      text: sampleTranscript,
      rawText: sampleTranscript,
      characterCount: sampleTranscript.length,
    }));
  }
  function addHistoryTranscript() {
    session.history.entries = [
      {
        ...structuredClone(historyEntry),
        id: nextHistoryID++,
        text: sampleTranscript,
        rawText: sampleTranscript,
        characterCount: sampleTranscript.length,
      },
      ...session.history.entries,
    ];
  }
  function updateHistoryTranscript() {
    session.history.entries = session.history.entries.map((entry, index) =>
      index
        ? entry
        : {
            ...entry,
            processingStatus: HistoryProcessingStatus.HistoryProcessingCompleted,
            processedText: "Cleaned. " + sampleTranscript,
            text: "Cleaned. " + sampleTranscript,
          },
    );
  }
  function showFileTranscript() {
    session.files.applyStatus({
      ...session.files.status,
      generation: 99,
      phase: FileTranscriptionPhase.FileTranscriptionFailed,
      transcript: sampleTranscript,
      fileName: "Example recording.wav",
      canCopy: true,
    });
    inputMode = "file";
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
  if (new URLSearchParams(location.search).get("feedback") === "voice") {
    session.dictation.status = {
      ...idle,
      generation: 8,
      state: State.Failed,
      canCopy: false,
      message:
        "The transcription request timed out. Review the connection before recording again. ".repeat(
          6,
        ),
    };
  }
  if (diagnosticsScenario) {
    session.editor.connection = null;
    if (diagnosticsScenario === "stale") {
      session.editor.draft!.textToSpeech.baseURL = "https://unsaved-draft.test/v1";
      void session.editor.testTextToSpeechConnection();
    }
  }
  function finishGeneration() {
    session.speech.applyStatus({
      ...session.speech.status,
      phase: TTSPhase.Playing,
      durationMilliseconds: 60000,
      positionMilliseconds: 10000,
      canSeek: true,
      canPause: true,
      canResume: false,
      canRestart: true,
      canSave: true,
      canClear: true,
      canStop: true,
    });
  }
  function fileCapability(unavailable: boolean, profile = false) {
    session.files.applyStatus({
      ...session.files.status,
      generation: session.files.status.generation + 1,
      phase: FileTranscriptionPhase.FileTranscriptionSelected,
      fileName: "Example recording.wav",
      fileSize: 10240,
      canStart: true,
      canCancel: false,
      streamingUnavailable: unavailable,
      streamingProfileUnavailable: profile,
    });
  }
  window.testSaves = saves.control;
  onDestroy(() => session.dispose());
  let inputMode = $state("voice");
  const noop = () => {};
  const footerStatus = $derived(taskConnectionStatus(inputMode, session.editor, Date.now()));
  const footerConnection = $derived(taskConnectionDetails(inputMode, session.editor));
</script>

<div class="flex h-screen flex-col overflow-hidden bg-background text-foreground">
  <AppHeader bind:inputMode settings={session.editor.applied} onSettings={noop} />
  <HomeScreen
    {session}
    bind:inputMode
    onOpenHistorySettings={noop}
    onOpenServerSettings={() => (openedSettings = "File transcription settings")}
    onOpenProcessingSettings={noop}
    onOpenAudioSettings={() => (openedSettings = "Audio settings")}
    onOpenShortcutSettings={() => (openedSettings = "Shortcut settings")}
    onOpenSpeechSettings={() => (openedSettings = "Speech settings")}
    onOpenGeneralSettings={noop}
  />
  {#if diagnosticsScenario}
    <StatusStrip
      connectionState={footerStatus}
      connectionDetails={footerConnection}
      onCheck={() => session.editor.testAppliedConnection(footerConnection.purpose)}
      onEdit={() =>
        (openedSettings = `Edit ${footerConnection.selected?.id || "connections"} for ${footerConnection.purpose}`)}
      onSettings={() => (openedSettings = "Speech settings")}
      onAbout={noop}
      version="Review"
    />
    {#if openedSettings}<p class="sr-only" role="status">{openedSettings}</p>{/if}
  {:else}
    <footer
      class="flex h-9 shrink-0 items-center border-t border-hairline bg-layer-fill px-4 text-xs text-muted-foreground"
    >
      {#if fileStreamingScenario}
        <div class="flex flex-wrap gap-3">
          <button onclick={() => fileCapability(false)}>Streaming supported</button>
          <button onclick={() => fileCapability(true)}>Streaming rejected</button>
          <button onclick={() => fileCapability(true, true)}>Completed-only profile</button>
          <span role="status"
            >File requests: {fileRequests}; streaming: {String(lastFileStream)}</span
          >
        </div>
      {:else if playbackScenario}
        <div class="flex flex-wrap gap-3">
          <button onclick={finishGeneration}>Finish generation</button>
          <button
            onclick={() =>
              session.speech.applyStatus({
                ...session.speech.status,
                source: TTSSource.SourceVoice,
              })}>Show transcript playback</button
          >
          <button
            onclick={() =>
              session.speech.applyStatus({
                ...session.speech.status,
                positionMilliseconds: session.speech.status.positionMilliseconds + 137,
              })}>Advance playback</button
          >
          <button
            onclick={() =>
              session.speech.applyStatus({
                ...session.speech.status,
                generation: session.speech.status.generation + 1,
                positionMilliseconds: 5000,
              })}>Replace audio</button
          >
          <span role="status">Speech requests: {speechRequests}; seek requests: {seekCalls}</span>
        </div>
      {:else if historyExpansion}
        <div class="flex gap-3">
          <button onclick={addHistoryTranscript}>Add transcript</button>
          <button onclick={updateHistoryTranscript}>Update latest</button>
          <button onclick={showFileTranscript}>Show file result</button>
          <button onclick={() => void session.history.clearHistory()}>Clear history</button>
        </div>
      {:else}
        {openedSettings || "Transcription: Reachable · Example connection"}
      {/if}
    </footer>
  {/if}
</div>
