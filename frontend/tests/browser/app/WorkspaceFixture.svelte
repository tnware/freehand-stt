<script lang="ts">
  import { onDestroy } from "svelte";
  import { CancellablePromise } from "@wailsio/runtime";
  import { ID as ModelProfileID } from "$bindings/modelprofile";
  import { modelOptions } from "$lib/utils/modelSettings";
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

  let current = structuredClone(settings);
  current.setupCompleted = true;
  current.historyEnabled = true;
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
      options: { ...modelOptions(current, Purpose.Speech), voice: "Alternate voice" },
    },
  ];
  const saves = controlledSaves((request) => {
    current = structuredClone({ ...current, ...request.settings });
    return structuredClone(current);
  });
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
      settings: {
        SaveSettings: (request) => saves.save(structuredClone($state.snapshot(request))),
      },
    }),
  );
  session.editor.applySettingsSnapshot(structuredClone(current));
  session.editor.devices = [{ id: "desk-mic", name: "Desk microphone", default: true }];
  session.editor.connection = structuredClone(connectionResult);
  session.dictation.status = { ...idle, transcript: "Testing, testing." };
  session.history.entries = [1, 2].map((id) => ({
    ...structuredClone(historyEntry),
    id,
    text: "Testing, testing.",
    rawText: "Testing, testing.",
  }));
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
    onOpenAudioSettings={noop}
    onOpenShortcutSettings={noop}
    onOpenSpeechSettings={noop}
    onOpenGeneralSettings={noop}
  />
  <footer
    class="flex h-9 shrink-0 items-center border-t border-hairline bg-layer-fill px-4 text-xs text-muted-foreground"
  >
    Transcription: Reachable · Example connection
  </footer>
</div>
