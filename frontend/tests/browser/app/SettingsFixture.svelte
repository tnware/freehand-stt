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
  import { controlledMetadata } from "./metadata-control";
  import { createRuntimeFixture } from "./runtime-fixture";
  import {
    publishRuntimeConnections,
    selectRuntimeConnection,
  } from "./runtime-use-fixture";
  import { createResourceFixture } from "./resource-fixture";
  import { createHistoryFixture } from "./history-fixture";
  import { ProviderID } from "$bindings/managedruntime";
  import { State, PostProcessingPreset } from "$lib/state";
  import { CancellablePromise } from "@wailsio/runtime";
  import { Action, Purpose } from "$bindings/savedconnection";
  import { CheckKind, CheckStatus } from "$bindings/connection";
  import { ID as ModelID, type Profile } from "$bindings/modelprofile";
  import { AuthenticationMode } from "$bindings/config";
  import { ID as BackendID, Role } from "$bindings/compatibility";
  import { Session } from "$lib/stores/session.svelte";
  import {
    settings,
    idle,
    serviceWithStatus,
    processingProfiles,
    connectionResult,
  } from "$lib/stores/session-fixtures-data";
  import App from "../../../src/App.svelte";
  import { controlledSaves } from "./save-control";
  import { installConnectionWindows, wire } from "./connection-window-bridge";
  const params = new URLSearchParams(location.search);
  const managedCleanup = params.has("managed-cleanup");
  const child = params.has("settings-frame");
  const integrated = !child && (params.has("blank") || params.has("main"));
  if (child) window.testConnectionWindows = window.parent.testConnectionWindows;

  import { shortcutCapture } from "$lib/stores/shortcutCapture.svelte";
  import { ShortcutAction } from "$bindings/hotkey";
  let current = structuredClone(settings);
  if (params.has("runtime-ready") || params.has("setup-ready"))
    current.setupCompleted = true;
  const historyFixture = params.has("history-workbench")
    ? createHistoryFixture()
    : null;
  if (historyFixture) current.historyEnabled = true;
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
  if (!new URLSearchParams(location.search).has("hold-binding"))
    setContext("hold-shortcut-retry", async () => {
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
        builtIn: false,
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
  if (params.has("unused-connection")) {
    current.savedConnections.entries!.push({
      ...current.savedConnections.entries![0],
      id: "unused",
      name: "Unused server",
    });
  }
  if (new URLSearchParams(location.search).has("workflows")) {
    const uses = [
      Purpose.Transcription,
      Purpose.Voice,
      Purpose.Cleanup,
      Purpose.Speech,
    ];
    current.savedConnections.entries![0].uses = uses;
    current.savedConnections.selected = Object.fromEntries(
      uses.map((use) => [use, "original"]),
    );
    const profile: Profile = {
      id: ModelID.Generic,
      name: "Generic",
      description: "Model options",
      reasoningOffRequired: false,
      capabilities: {
        nemoTranscriptionControls: false,
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
  if (new URLSearchParams(location.search).has("pickers"))
    configurePickerFixture(current);
  if (params.has("workflow-identity")) {
    // Explicit backend metadata exercises identity separately from the saved
    // connection name. It never comes from model-ID inference.
    const profile = {
      id: BackendID.Generic,
      label: "Generic OpenAI-compatible",
      available: true,
      description: "Synthetic connection metadata",
      capabilities: { ...current.modelProfiles.transcription![0].capabilities },
    };
    current.compatibilityProfiles.transcription = [profile];
    current.compatibilityProfiles.speech = [profile];
    current.voiceTranscription.compatibilityProfile = profile.id;
  }
  const blank = new URLSearchParams(location.search).has("blank");
  if (blank) {
    current.savedConnections = { entries: [], selected: {} };
    current.setupCompleted = false;
    current.voiceTranscription.model = "";
    current.voiceTranscription.modelProfile = ModelID.Generic;
    current.rememberedModels.defaults![Purpose.Voice]!.profile =
      ModelID.Generic;
    current.model = "";
  }
  let pendingPreviewSave = $state<number | null>(null);
  const saves = controlledSaves((request) => {
    const next = { ...current, ...request.settings };
    const change = request.connectionChange;
    if (params.has("runtime-use-flow") && change?.action === Action.Select) {
      selectRuntimeConnection(next, change, runtimeFixture?.providers ?? []);
      current = structuredClone(next);
      return structuredClone(current);
    }
    if (change?.action === Action.Delete) {
      next.savedConnections = {
        ...current.savedConnections,
        entries: (current.savedConnections.entries ?? []).filter(
          (entry) => entry.id !== change.id,
        ),
      };
      current = structuredClone(next);
      return structuredClone(current);
    }
    if (change) {
      if (change.action !== Action.Create || !change.details)
        throw new Error("Unexpected connection action");
      const id = "created";
      if (change.activateFor === Purpose.Voice) {
        next.voiceTranscription = {
          ...next.voiceTranscription,
          ...change.details,
        };
      }
      next.savedConnections = {
        entries: [
          ...(current.savedConnections.entries ?? []),
          {
            id,
            name: change.name,
            builtIn: false,
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
  const runtimeFixture =
    params.has("runtime") || managedCleanup
      ? createRuntimeFixture(
          current.platform !== "darwin",
          (instances, providers) => {
            current = { ...current, managedRuntimes: instances };
            if (params.has("runtime-use-flow"))
              publishRuntimeConnections(current, instances, providers);
            // Match Go's selected-connection projection after inventory commits.
            // Runtime status alone must not leave the applied task model stale.
            for (const [endpoint, role] of [
              [current, Role.Transcription],
              [current.voiceTranscription, Role.Transcription],
              [current.postProcessing, Role.PostProcessing],
              [current.textToSpeech, Role.Speech],
            ] as const) {
              const instance = instances.find(
                (entry) => entry.id === endpoint.managedInstanceID,
              );
              if (!instance) continue;
              const modelID =
                role === Role.Speech
                  ? instance.speechModel || instance.model
                  : instance.model;
              const model = providers
                .find((provider) => provider.id === instance.provider)
                ?.models?.find((entry) => entry.id === modelID);
              const contract = model?.contracts?.find(
                (entry) => entry.role === role,
              );
              if (!contract) continue;
              endpoint.model = modelID;
              endpoint.compatibilityProfile = contract.compatibilityProfile;
              if ("modelProfile" in endpoint)
                endpoint.modelProfile = contract.modelProfile;
              else
                endpoint.preset =
                  contract.modelProfile === ModelID.S1Mini
                    ? PostProcessingPreset.PostProcessingPresetS1Mini
                    : PostProcessingPreset.PostProcessingPresetGeneric;
              if (endpoint === current.voiceTranscription && !model?.realtime)
                current.voiceTranscription.realtime = false;
            }
          },
          params.has("runtime-ready") || managedCleanup,
          managedCleanup || params.get("runtime-provider") === "llama-cpp"
            ? ProviderID.LlamaCPP
            : params.get("runtime-provider") === "whisper-cpp"
              ? ProviderID.WhisperCPP
              : ProviderID.NeMoSpeechCPP,
          managedCleanup ? "Local cleanup" : "Local speech",
        )
      : null;
  if (
    runtimeFixture &&
    params.has("runtime-whisper-variants") &&
    runtimeFixture.providers[0]?.id === ProviderID.WhisperCPP
  ) {
    runtimeFixture.control.useWhisperVariantCatalog();
  }
  if (runtimeFixture && params.has("runtime-all-providers")) {
    runtimeFixture.providers.unshift(
      createRuntimeFixture(true, () => {}, false, ProviderID.LlamaCPP)
        .providers[0],
    );
  }
  if (runtimeFixture) window.testRuntime = runtimeFixture.control;
  if (managedCleanup && runtimeFixture) {
    const instance = runtimeFixture.control.snapshot()[0].instance;
    const remote = current.savedConnections.entries!.find(
      (entry) => entry.id === "original",
    )!;
    remote.details.baseURL = "https://speech.example.test/v1";
    current.baseURL = remote.details.baseURL;
    current.voiceTranscription.baseURL = remote.details.baseURL;
    current.savedConnections.entries!.push({
      id: "local-cleanup",
      name: instance.name,
      builtIn: true,
      uses: [Purpose.Cleanup],
      hasCredential: false,
      details: {
        managedInstanceID: instance.id,
        compatibilityProfile: BackendID.$zero,
        baseURL: "",
        allowInsecureHTTP: false,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    });
    current.savedConnections.selected = {
      ...current.savedConnections.selected,
      [Purpose.Cleanup]: "local-cleanup",
    };
    current.postProcessing = {
      ...current.postProcessing,
      enabled: false,
      managedInstanceID: instance.id,
      model: instance.model,
      preset: PostProcessingPreset.PostProcessingPresetS1Mini,
      compatibilityProfile: BackendID.LlamaCPP,
      baseURL: "",
      allowInsecureHTTP: false,
    };
    current.modelProfiles.postProcessing =
      runtimeFixture.providers[0].models?.flatMap((model) =>
        model.behavior ? [model.behavior] : [],
      ) ?? [];
  }
  if (
    runtimeFixture &&
    current.managedRuntimes?.length &&
    runtimeFixture.providers[0].id === ProviderID.NeMoSpeechCPP &&
    params.has("runtime-ready")
  ) {
    const instance = current.managedRuntimes[0];
    const profiles =
      runtimeFixture.providers[0].models
        ?.filter((m) =>
          m.contracts?.some((contract) => contract.role === Role.Transcription),
        )
        .flatMap((m) => (m.behavior ? [m.behavior] : [])) ?? [];
    current.modelProfiles.voiceTranscription = profiles;
    current.modelProfiles.transcription = profiles;
    if (params.has("runtime-combined")) {
      current.modelProfiles.speech =
        runtimeFixture.providers[0].models
          ?.filter((m) =>
            m.contracts?.some((contract) => contract.role === Role.Speech),
          )
          .flatMap((m) => (m.behavior ? [m.behavior] : [])) ?? [];
      current.textToSpeech = {
        ...current.textToSpeech,
        enabled: true,
        managedInstanceID: instance.id,
        model: "magpie-tts",
        modelProfile: ModelID.MagpieTTS,
        compatibilityProfile: BackendID.NeMoSpeechV1,
        baseURL: "",
        voice: "default",
        speed: 1,
        options: { language: "", instructions: "" },
      };
    }
    current.savedConnections.entries!.push({
      id: "local-speech",
      name: instance.name,
      builtIn: !params.has("legacy-runtime"),
      uses: [
        Purpose.Voice,
        Purpose.Transcription,
        ...(params.has("runtime-combined") ? [Purpose.Speech] : []),
      ],
      hasCredential: false,
      details: {
        managedInstanceID: instance.id,
        compatibilityProfile: BackendID.$zero,
        baseURL: "",
        allowInsecureHTTP: false,
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        healthPath: "",
        headers: {},
      },
    });
    const managed = {
      managedInstanceID: instance.id,
      model: instance.model,
      compatibilityProfile: BackendID.NeMoSpeechV1,
      modelProfile: ModelID.Nemotron35,
      baseURL: "",
      allowInsecureHTTP: false,
      authenticationMode: AuthenticationMode.AuthenticationModeNone,
      healthPath: "",
      headers: {},
    };
    current = {
      ...current,
      ...managed,
      voiceTranscription: {
        ...current.voiceTranscription,
        ...managed,
        realtime: true,
      },
    };
    current.savedConnections.selected = {
      [Purpose.Voice]: "local-speech",
      [Purpose.Transcription]: "local-speech",
      ...(params.has("runtime-combined")
        ? { [Purpose.Speech]: "local-speech" }
        : {}),
    };
  }
  const metadata =
    params.has("metadata-control") || params.has("voice-metadata-control")
      ? controlledMetadata()
      : null;
  if (metadata) {
    window.testMetadata = metadata.control;
    current.setupCompleted = true;
    if (params.has("metadata-readiness")) current.model = "";
    current.authenticationMode = AuthenticationMode.AuthenticationModeNone;
    current.postProcessing.enabled = true;
  }
  const session = new Session({
    resources: createResourceFixture(),
    ...serviceWithStatus(
      () =>
        CancellablePromise.resolve(
          params.has("work-busy") ? { ...idle, state: State.Recording } : idle,
        ),
      {
        history: historyFixture?.service,
        input: {
          ListMicrophones: () =>
            CancellablePromise.resolve([
              { id: "default", name: "Fixture microphone", default: true },
            ]),
        },
        connection: {
          TestSavedConnection: () =>
            params.has("voice-metadata-control") && metadata
              ? metadata.request(Purpose.Voice)
              : CancellablePromise.resolve(connectionResult),
          TestConnection: () =>
            metadata?.request(Purpose.Transcription) ??
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
          TestPostProcessingConnection: () =>
            metadata?.request(Purpose.Cleanup) ??
            CancellablePromise.resolve(connectionResult),
          TestTextToSpeechConnection: () =>
            metadata?.request(Purpose.Speech) ??
            CancellablePromise.resolve(connectionResult),
        },
        settings: {
          GetSettings: () =>
            CancellablePromise.resolve(
              child
                ? wire(window.testConnectionWindows.settings())
                : structuredClone(current),
            ),
          SaveSettings: (request) => {
            if (child)
              return CancellablePromise.resolve(
                window.testConnectionWindows.save(wire(request)),
              );
            const result = saves.save(wire(request));
            if (params.has("runtime-use-preview"))
              void saves.control.waitForStart(0).then((id) => {
                pendingPreviewSave = id;
              });
            return result;
          },
        },
      },
    ),
    runtime: runtimeFixture?.service,
  });
  // Model a cold startup for the delayed Voice probe: preloading here would
  // start a check before App's initial load adopts the same settings again.
  if (!params.has("voice-metadata-control"))
    session.editor.applySettingsSnapshot(structuredClone(current));
  if (historyFixture) {
    window.testHistory = {
      failNextClear: historyFixture.failNextClear,
      appendVoice: async () => {
        historyFixture.appendVoice();
        await session.history.refresh();
      },
    };
  }
  runtimeFixture?.subscribe((status) => session.runtime.applyStatus(status));
  if (new URLSearchParams(location.search).has("workflows")) {
    session.editor.processingProfiles = structuredClone(processingProfiles);
  }

  if (!child) {
    window.testSaves = saves.control;
    installConnectionWindows(
      {
        settings: () => structuredClone(current),
        save: (request) => saves.save(wire(request)),
        apply: (saved) => session.editor.applySettingsSnapshot(wire(saved)),
      },
      integrated,
      params.has("general"),
    );
  }
</script>

<!-- Settings is a pane of the main window, so every scenario mounts the one
     shell rather than a standalone settings renderer. -->
<App {session} />

{#if params.has("runtime-use-preview") && runtimeFixture}
  <aside
    aria-label="Browser fixture controls"
    class="fixed bottom-8 right-2 z-[100] max-w-64 space-y-1 rounded border border-border bg-background p-2 text-xs shadow"
  >
    <p class="font-semibold">Browser fixture controls · no native work</p>
    <button
      class="block underline"
      onclick={() => {
        const row = runtimeFixture.control
          .snapshot()
          .find(
            (row) =>
              row.status.operation.kind === "download" &&
              row.status.operation.outcome === "running",
          );
        if (row) runtimeFixture.control.finishDownload(row.instance.id);
      }}>Complete fixture model download</button
    >
    <button
      class="block underline disabled:opacity-40"
      disabled={pendingPreviewSave === null}
      onclick={() => {
        if (pendingPreviewSave === null) return;
        saves.control.complete(pendingPreviewSave, "success");
        pendingPreviewSave = null;
      }}>Complete fixture settings save</button
    >
    <button
      class="block underline disabled:opacity-40"
      disabled={pendingPreviewSave === null}
      onclick={() => {
        if (pendingPreviewSave === null) return;
        saves.control.complete(pendingPreviewSave, "failure");
        pendingPreviewSave = null;
      }}>Fail fixture settings save</button
    >
  </aside>
{/if}
