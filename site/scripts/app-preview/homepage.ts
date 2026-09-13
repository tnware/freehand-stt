// Homepage photography: the production App, with deterministic sample data at
// the service boundary. No inference, credentials, or native actions are used.
import { mount } from "svelte";
import { CancellablePromise } from "@wailsio/runtime";
import App from "../../../src/App.svelte";
import "../../../src/app.css";
import { ID } from "$bindings/compatibility";
import { Purpose } from "$bindings/savedconnection";
import {
  AppearanceMode,
  AuthenticationMode,
  FileTranscriptionPhase,
  TTSPhase,
  TTSSource,
} from "$lib/state";
import {
  Session,
  settings,
  idle,
  connectionResult,
  serviceWithStatus,
} from "$lib/stores/session-fixtures-data";

const speech = new URLSearchParams(location.search).get("task") === "speech";
const current = structuredClone(settings);
current.setupCompleted = true;
current.historyEnabled = false;
current.appearanceMode = AppearanceMode.AppearanceModeDark;
current.authenticationMode = AuthenticationMode.AuthenticationModeNone;
current.headers = {};
current.baseURL = "http://localhost:8000/v1";
current.allowInsecureHTTP = true;
current.compatibilityProfile = ID.Speaches;
current.model = "Systran/faster-whisper-large-v3";
current.postProcessing.enabled = false;
current.textToSpeech = {
  ...current.textToSpeech,
  enabled: true,
  compatibilityProfile: ID.Speaches,
  baseURL: current.baseURL,
  allowInsecureHTTP: true,
  authenticationMode: AuthenticationMode.AuthenticationModeNone,
  model: "hexgrad/Kokoro-82M",
  voice: "af_heart",
  speed: 1,
};
current.savedConnections = {
  entries: [
    {
      id: "local-speech",
      name: "Local speech server",
      uses: [Purpose.Transcription, Purpose.Speech],
      hasCredential: false,
      details: {
        compatibilityProfile: ID.Speaches,
        baseURL: current.baseURL,
        allowInsecureHTTP: true,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    },
  ],
  selected: {
    [Purpose.Transcription]: "local-speech",
    [Purpose.Speech]: "local-speech",
  },
};
const transcript = [
  "I took the coastal path just after sunrise. The tide was still out, and the whole beach was quiet except for a few gulls and the sound of the water.",
  "There’s a small café at the end of the trail, just past the old lighthouse. I stopped for coffee and watched the fishing boats come back into the harbour.",
  "Next time, I’d like to bring a camera and take the longer route along the cliffs. It’s the kind of place that makes you want to slow down and stay a little longer.",
].join("\n\n");
const narration = [
  "Take the scenic route.",
  "Follow the coast as the city gives way to open water, quiet beaches, and little towns worth stopping for. Find a café with a view. Walk a trail you’ve never taken. Leave a little room for the unexpected.",
  "You don’t need a perfect itinerary. Just a free afternoon, a good playlist, and somewhere new to go.",
].join("\n\n");
const connection = {
  ...structuredClone(connectionResult),
  requestedURL: `${current.baseURL}/models`,
  checkedAt: new Date().toISOString(),
  modelIDs: [current.model, current.textToSpeech.model],
};
const session = new Session(
  serviceWithStatus(() => CancellablePromise.resolve(idle), {
    settings: { GetSettings: () => CancellablePromise.resolve(current) },
    connection: {
      TestConnection: () => CancellablePromise.resolve(connection),
      TestSavedConnection: () => CancellablePromise.resolve(connection),
      TestTextToSpeechConnection: () => CancellablePromise.resolve(connection),
    },
    files: {
      CurrentFileTranscription: () =>
        CancellablePromise.resolve({
          generation: 1,
          phase: speech
            ? FileTranscriptionPhase.FileTranscriptionEmpty
            : FileTranscriptionPhase.FileTranscriptionCompleted,
          fileName: speech ? "" : "Coastal walk.m4a",
          fileSize: speech ? 0 : 1480000,
          transcript: speech ? "" : transcript,
          transcriptRevision: 1,
          streaming: false,
          buffered: false,
          streamingProfileUnavailable: false,
          streamingUnavailable: false,
          canStart: !speech,
          canCancel: false,
          canCopy: !speech,
        }),
    },
    speech: {
      CurrentStatus: () =>
        CancellablePromise.resolve({
          generation: 1,
          source: TTSSource.SourceCompose,
          phase: speech ? TTSPhase.Paused : TTSPhase.Idle,
          positionMilliseconds: speech ? 12000 : 0,
          durationMilliseconds: speech ? 29000 : 0,
          canPause: false,
          canResume: speech,
          canRestart: speech,
          canSeek: speech,
          canStop: speech,
          canSave: speech,
          canClear: speech,
        }),
    },
  }),
);
if (speech) session.speech.draft = narration;
mount(App, { target: document.getElementById("app")!, props: { session } });
