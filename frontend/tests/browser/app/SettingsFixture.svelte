<script lang="ts">
  import { setContext } from "svelte";
  import {
    NATIVE_PERMISSION_SERVICES,
    type NativePermissionServices,
  } from "$lib/stores/nativePermissions.svelte";
  import type { PermissionStatus } from "$bindings/input";
  let permissionStatus: PermissionStatus = {
    required: true,
    microphone: "not-determined",
    accessibility: false,
    keyboard: false,
  };
  setContext<NativePermissionServices>(NATIVE_PERMISSION_SERVICES, {
    NativePermissions: async () => ({ ...permissionStatus }),
    RequestPermission: async () => {
      permissionStatus = { ...permissionStatus, microphone: "denied" };
      return { ...permissionStatus };
    },
    OpenPermissionSettings: async () => {},
  });
  import { configurePickerFixture } from "./picker-data";
  import { CancellablePromise } from "@wailsio/runtime";
  import { Action, Purpose } from "$bindings/savedconnection";
  import { CheckKind, CheckStatus } from "$bindings/connection";
  import { ID as ModelID, type Profile } from "$bindings/modelprofile";
  import { AuthenticationMode } from "$bindings/config";
  import { Session } from "$lib/stores/session.svelte";
  import {
    settings,
    idle,
    serviceWithStatus,
    processingProfiles,
    connectionResult,
  } from "$lib/stores/session-fixtures-data";
  import App from "../../../src/App.svelte";
  import SettingsHost from "../../../src/SettingsHost.svelte";
  import { controlledSaves } from "./save-control";
  import { installConnectionWindows, wire } from "./connection-window-bridge";
  const params = new URLSearchParams(location.search);
  const child = params.has("settings-frame");
  const integrated = !child && (params.has("blank") || params.has("main"));
  if (child) window.testConnectionWindows = window.parent.testConnectionWindows;

  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import { ShortcutAction } from "$bindings/hotkey";
  let current = structuredClone(settings);
  if (new URLSearchParams(location.search).has("hold-degraded")) {
    current.holdShortcut = "F13";
    current.holdAvailable = false;
    current.holdAvailabilityReason = "Keyboard event tap stopped.";
    shortcutCapture.policies = [
      ShortcutAction.ToggleRecording,
      ShortcutAction.ShowFreehand,
      ShortcutAction.HoldToTalk,
    ].map((action) => ({
      action,
      required: false,
      modifiedPrimaryGroups: [],
      dedicatedPrimaryGroups: [],
      modifierOnlyMinimum: 0,
      externalAvailabilityKnown: false,
    }));
  }
  let retries = 0;
  if (!new URLSearchParams(location.search).has("hold-binding")) setContext("hold-shortcut-retry", async () => {
    retries++;
    if (retries === 1) throw new Error("Secure Input is still active.");
    current.holdAvailable = true;
    current.holdAvailabilityReason = "Hold-to-talk is ready.";
    return structuredClone(current);
  });
  if (new URLSearchParams(location.search).get("platform") === "darwin") {
    current.platform = "darwin";
    current.useMica = true;
    current.micaActive = true;
  }
  current.savedConnections = {
    entries: [
      {
        id: "original",
        name: "Original server",
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
      },
    ],
    selected: { [Purpose.Transcription]: "original" },
  };
  if (new URLSearchParams(location.search).has("workflows")) {
    const uses = [Purpose.Transcription, Purpose.Voice, Purpose.Cleanup, Purpose.Speech];
    current.savedConnections.entries![0].uses = uses;
    current.savedConnections.selected = Object.fromEntries(uses.map((use) => [use, "original"]));
    const profile: Profile = {
      id: ModelID.Generic,
      name: "Generic",
      description: "Model options",
      reasoningOffRequired: false,
      capabilities: {
        realtime: false,
        serverLoadedModel: false,
        vllmTranscriptionEvents: false,
        cleanupOutputLimit: true,
        cleanupDisableReasoning: true,
        fileStreaming: false,
        typedTranscriptionEvents: false,
        legacyTranscriptionSegments: false,
        languageHint: true,
        voiceDiscovery: false,
        speechInstructions: false,
        speechLanguage: false,
        speechSpeed: true,
        transcriptionPrompt: true,
        transcriptionHotwords: false,
        transcriptionTemperature: true,
      },
    };
    current.modelProfiles = {
      transcription: [profile],
      voiceTranscription: [profile],
      postProcessing: [profile],
      speech: [profile],
      realtime: [],
    };
    current.voiceTranscription.modelProfile = ModelID.Generic;
    current.voiceTranscription.model = "speech/stt";
    current.textToSpeech.enabled = true;
    current.textToSpeech.model = "speech/tts";
    current.textToSpeech.voice = "alloy";
  }
  if (new URLSearchParams(location.search).has("pickers")) configurePickerFixture(current);
  const blank = new URLSearchParams(location.search).has("blank");
  if (blank) {
    current.savedConnections = { entries: [], selected: {} };
    current.setupCompleted = false;
    current.voiceTranscription.model = "";
    current.voiceTranscription.modelProfile = ModelID.Generic;
    current.rememberedModels.defaults![Purpose.Voice]!.profile = ModelID.Generic;
    current.model = "";
  }
  const saves = controlledSaves((request) => {
    const next = { ...current, ...request.settings };
    const change = request.connectionChange;
    if (change) {
      if (change.action !== Action.Create || !change.details)
        throw new Error("Unexpected connection action");
      const id = "created";
      if (change.activateFor === Purpose.Voice) {
        next.voiceTranscription = { ...next.voiceTranscription, ...change.details };
      }
      next.savedConnections = {
        entries: [
          ...(current.savedConnections.entries ?? []),
          {
            id,
            name: change.name,
            uses: change.uses ?? [],
            details: change.details,
            hasCredential: false,
          },
        ],
        selected: {
          ...current.savedConnections.selected,
          ...(change.activateFor ? { [change.activateFor]: id } : {}),
        },
      };
    }
    current = structuredClone(next);
    return structuredClone(current);
  });
  if (child) current = wire(window.testConnectionWindows.settings());
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
      input: { ListMicrophones: () => CancellablePromise.resolve([{ id: "default", name: "Fixture microphone", default: true }]) },
      connection: {
        TestConnection: () =>
          CancellablePromise.resolve(
            new URLSearchParams(location.search).has("attention")
              ? {
                  ...connectionResult,
                  checks: [
                    {
                      kind: CheckKind.CheckModel,
                      status: CheckStatus.CheckAttention,
                      summary: "The selected model was not listed.",
                      detail: "Refresh models to choose another model.",
                    },
                  ],
                }
              : connectionResult,
          ),
      },
      settings: {
        GetSettings: () => CancellablePromise.resolve(child ? wire(window.testConnectionWindows.settings()) : structuredClone(current)),
        SaveSettings: (request) => child
          ? CancellablePromise.resolve(window.testConnectionWindows.save(wire(request)))
          : saves.save(wire(request)),
      },
    }),
  );
  session.editor.applySettingsSnapshot(structuredClone(current));
  if (new URLSearchParams(location.search).has("workflows")) {
    session.editor.processingProfiles = structuredClone(processingProfiles);
  }

  if (!child) {
    window.testSaves = saves.control;
    installConnectionWindows({
      settings: () => structuredClone(current),
      save: (request) => saves.save(wire(request)),
      apply: (saved) => session.editor.applySettingsSnapshot(wire(saved)),
    }, integrated, params.has("general"));
  }
</script>

{#if integrated}
  <App {session} />
{:else}
  <SettingsHost {session} />
{/if}
