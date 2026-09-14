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
  import { Purpose } from "$bindings/savedconnection";
  const params = new URLSearchParams(location.search);
  const calls: string[] = [];
  (window as any).trayCalls = calls;
  let current = structuredClone(settings);
  current.setupCompleted = true;
  current.voiceTranscription.baseURL = "https://example.test/v1";
  current.voiceTranscription.model = "speech/stt";
  current.savedConnections = {
    entries: [
      {
        id: "voice",
        builtIn: false,
        name: "Voice server",
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
    transcript: params.has("result")
      ? "A completed local fixture transcript. ".repeat(70)
      : "",
    state: params.has("working") ? State.Transcribing : State.Idle,
  };
  let release!: () => void;
  const session = new Session(
    serviceWithStatus(
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
    ),
  );
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
