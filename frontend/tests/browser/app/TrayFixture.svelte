<script lang="ts">
  import { CancellablePromise } from "@wailsio/runtime";
  import TrayPopover from "../../../src/TrayPopover.svelte";
  import {
    Session,
    settings,
    idle,
    serviceWithStatus,
  } from "$lib/stores/session-fixtures-data";
  import { State } from "$lib/state";
  import { createRuntimeFixture } from "./runtime-fixture";
  import { Purpose } from "$bindings/savedconnection";
  const params = new URLSearchParams(location.search);
  const calls: string[] = [];
  (window as any).trayCalls = calls;
  let current = structuredClone(settings);
  current.setupCompleted = true;
  if (params.has("no-shortcut")) {
    current.toggleShortcut = "";
    current.holdShortcut = "";
  }
  if (params.has("mac")) current.platform = "darwin";
  const runtimeFixture = params.has("managed")
    ? createRuntimeFixture(true, () => {}, true)
    : null;
  if (runtimeFixture) {
    const instance = runtimeFixture.control.snapshot()[0].instance;
    current.managedRuntimes = [instance];
    current.voiceTranscription.managedInstanceID = instance.id;
    current.voiceTranscription.model = instance.model;
  }
  current.voiceTranscription.baseURL = "https://example.test/v1";
  if (!runtimeFixture) current.voiceTranscription.model = "speech/stt";
  current.savedConnections = {
    entries: [
      {
        id: "voice",
        builtIn: false,
        name: runtimeFixture ? "Local speech" : "Voice server",
        uses: [Purpose.Voice],
        hasCredential: false,
        details: { ...current.voiceTranscription },
      },
    ],
    selected: { [Purpose.Voice]: "voice" },
  };
  const initial = {
    ...idle,
    generation: 7,
    transcript:
      params.has("result") || params.has("recovery")
        ? "A completed local fixture transcript. ".repeat(70)
        : "",
    state: params.has("recording")
      ? State.Recording
      : params.has("working")
        ? State.Transcribing
        : params.has("recovery") || params.has("error")
          ? State.Failed
          : State.Idle,
    canCancel: params.has("recording") || params.has("working"),
    canCopy: params.has("recovery"),
    message: params.has("error")
      ? "The server could not complete transcription. Review the connection and try again. ".repeat(
          8,
        )
      : "",
  };
  let release!: () => void;
  const services = serviceWithStatus(
    () =>
      params.has("loading")
        ? new CancellablePromise((resolve) => {
            release = () => resolve(initial);
          })
        : CancellablePromise.resolve(initial),
    {
      input: {
        ListMicrophones: () =>
          CancellablePromise.resolve(
            params.has("no-microphone")
              ? []
              : [
                  {
                    id: "default",
                    name: "Fixture microphone",
                    default: true,
                  },
                ],
          ),
      },
      dictation: {
        StartRecording: () => {
          calls.push("start");
          session.dictation.applyStatus({
            ...initial,
            state: State.Recording,
          });
          return CancellablePromise.resolve();
        },
        StopRecording: () => {
          calls.push("stop");
          session.dictation.applyStatus({
            ...initial,
            state: State.Transcribing,
          });
          return CancellablePromise.resolve();
        },
        Cancel: () => {
          calls.push("cancel");
          session.dictation.applyStatus({
            ...idle,
            generation: initial.generation,
          });
          return CancellablePromise.resolve();
        },
        CopyPending: () => {
          calls.push("copy-pending");
          return params.has("copy-error")
            ? CancellablePromise.reject(new Error("Clipboard unavailable"))
            : CancellablePromise.resolve();
        },
        CopyCurrent: () => {
          calls.push("copy");
          return params.has("copy-error")
            ? CancellablePromise.reject(new Error("Clipboard unavailable"))
            : CancellablePromise.resolve();
        },
      },
      settings: {
        GetSettings: () => CancellablePromise.resolve(current),
        SaveSettings: (request) => {
          calls.push("save");
          current = {
            ...current,
            ...structuredClone($state.snapshot(request.settings)),
          };
          if (request.connectionChange)
            current.savedConnections.selected = {
              [Purpose.Voice]: request.connectionChange.id,
            };
          return CancellablePromise.resolve(current);
        },
      },
    },
  );
  if (runtimeFixture) services.runtime = runtimeFixture.service;
  const session = new Session(services);
  if (runtimeFixture) {
    window.testRuntime = runtimeFixture.control;
    runtimeFixture.subscribe((row) => session.runtime.applyStatus(row));
  }
  (window as any).trayFixture = { session, release: () => release() };
  const windows = {
    HideTrayPopover: () => {
      calls.push("hide");
      return params.has("hide-error")
        ? CancellablePromise.reject(new Error("Cannot dismiss panel"))
        : CancellablePromise.resolve();
    },
    OpenMain: () => {
      calls.push("main");
      return CancellablePromise.resolve();
    },
    OpenSettings: (section: string) => {
      calls.push(`settings:${section}`);
      return CancellablePromise.resolve();
    },
    OpenTaskConnection: () => {
      calls.push("connections");
      return CancellablePromise.resolve();
    },
  };
</script>

<TrayPopover {session} {windows} />
