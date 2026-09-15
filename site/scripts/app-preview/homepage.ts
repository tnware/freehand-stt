// Homepage photography: the production App, with deterministic sample data at
// the service boundary. No inference, credentials, or native actions are used.
import { mount } from "svelte";
import { CancellablePromise } from "@wailsio/runtime";
import App from "../../../src/App.svelte";
import "../../../src/app.css";
import { ID, Role } from "$bindings/compatibility";
import { ID as ModelID } from "$bindings/modelprofile";
import { ProviderID } from "$bindings/managedruntime";
import { VoiceScope } from "$bindings/inference";
import { createRuntimeFixture } from "../app/runtime-fixture";
import { createResourceFixture } from "../app/resource-fixture";
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
const runtime = createRuntimeFixture(
  true,
  (instances) => {
    current.managedRuntimes = instances;
  },
  true,
  ProviderID.NeMoSpeechCPP,
  "NeMo-Speech.cpp",
);
const row = runtime.control.snapshot()[0];
runtime.control.change(row.instance.id, {
  state: "running",
  backend: "cuda",
  models: row.status.models?.map((model) => ({ ...model, installed: true })),
});
const profiles = (role: Role) =>
  runtime.providers[0].models?.flatMap(
    (model) =>
      model.contracts
        ?.filter((contract) => contract.role === role)
        .map((contract) => contract.behavior) ?? [],
  ) ?? [];
current.modelProfiles.voiceTranscription = profiles(Role.Transcription);
current.modelProfiles.transcription = profiles(Role.Transcription);
current.modelProfiles.speech = profiles(Role.Speech);
current.baseURL = "";
current.managedInstanceID = row.instance.id;
current.allowInsecureHTTP = false;
current.compatibilityProfile = ID.NeMoSpeechV1;
current.model = row.instance.model;
current.modelProfile = ModelID.Nemotron35;
current.voiceTranscription = {
  ...current.voiceTranscription,
  managedInstanceID: row.instance.id,
  baseURL: "",
  model: current.model,
  compatibilityProfile: ID.NeMoSpeechV1,
  modelProfile: ModelID.Nemotron35,
  realtime: true,
};
current.postProcessing.enabled = false;
current.textToSpeech = {
  ...current.textToSpeech,
  enabled: true,
  compatibilityProfile: ID.NeMoSpeechV1,
  managedInstanceID: row.instance.id,
  baseURL: current.baseURL,
  allowInsecureHTTP: false,
  authenticationMode: AuthenticationMode.AuthenticationModeNone,
  model: "magpie-tts",
  modelProfile: ModelID.MagpieTTS,
  options: { language: "en-US", instructions: "" },
  voice: "Sofia",
  speed: 1,
};
current.savedConnections = {
  entries: [
    {
      id: "local-speech",
      name: "NeMo-Speech.cpp",
      builtIn: true,
      uses: [Purpose.Voice, Purpose.Transcription, Purpose.Speech],
      hasCredential: false,
      details: {
        compatibilityProfile: ID.NeMoSpeechV1,
        managedInstanceID: row.instance.id,
        baseURL: current.baseURL,
        allowInsecureHTTP: false,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    },
  ],
  selected: {
    [Purpose.Voice]: "local-speech",
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
  requestedURL: "http://127.0.0.1:17860/v1/models",
  checkedAt: new Date().toISOString(),
  modelIDs: [current.model, current.textToSpeech.model],
  models: [
    { id: current.model, capability: "transcription", device: "cuda" },
    { id: current.textToSpeech.model, capability: "speech", device: "cuda" },
  ],
};
const resources = createResourceFixture();
Object.assign(window.testResources.sample, {
  cpuPercent: 8,
  memoryAvailableBytes: 8 * 1024 ** 3,
});
Object.assign(window.testResources.sample.gpus![0], {
  name: "NVIDIA GPU",
  utilizationPercent: 12,
  memoryUsedBytes: 3 * 1024 ** 3,
});
const session = new Session({
  ...serviceWithStatus(() => CancellablePromise.resolve(idle), {
    settings: { GetSettings: () => CancellablePromise.resolve(current) },
    connection: {
      TestConnection: () => CancellablePromise.resolve(connection),
      TestSavedConnection: () => CancellablePromise.resolve(connection),
      TestTextToSpeechConnection: () => CancellablePromise.resolve(connection),
      ListSpeechVoices: () =>
        CancellablePromise.resolve({
          voices: ["John", "Sofia", "Aria", "Jason", "Leo"].map((id) => ({
            id,
            name: id,
            language: "",
          })),
          languages: ["en-US", "fr-FR"],
          scope: VoiceScope.VoiceScopeModel,
          errorKind: "",
          httpStatus: 200,
          latencyMilliseconds: 4,
          truncated: false,
        }),
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
  runtime: runtime.service,
  resources,
});
runtime.subscribe((row) => session.runtime.applyStatus(row));
session.files.streamingPreferred = false;
if (speech) session.speech.draft = narration;
mount(App, { target: document.getElementById("app")!, props: { session } });
