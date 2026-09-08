<script lang="ts">
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
  import SettingsScreen from "$lib/components/settings/SettingsScreen.svelte";
  import type { SettingsSectionID } from "$lib/navigation";
  import { controlledSaves } from "./save-control";

  let current = structuredClone(settings);
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
  const saves = controlledSaves((request) => {
    const next = { ...current, ...request.settings };
    const change = request.connectionChange;
    if (change) {
      if (change.action !== Action.Create || !change.details)
        throw new Error("Unexpected connection action");
      const id = "created";
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
        selected: { ...current.savedConnections.selected, [Purpose.Transcription]: id },
      };
    }
    current = structuredClone(next);
    return structuredClone(current);
  });
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
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
        SaveSettings: (request) => saves.save(structuredClone($state.snapshot(request))),
      },
    }),
  );
  session.editor.applySettingsSnapshot(structuredClone(current));
  if (new URLSearchParams(location.search).has("workflows")) {
    session.editor.processingProfiles = structuredClone(processingProfiles);
  }

  window.testSaves = saves.control;
  let active = $state<SettingsSectionID>("server");
</script>

<div class="flex h-screen flex-col text-foreground">
  <SettingsScreen
    {session}
    bind:active
    onClose={() => {}}
    overlayPreviewing={false}
    onStartOverlayPreview={() => {}}
    onStopOverlayPreview={() => {}}
  />
</div>
