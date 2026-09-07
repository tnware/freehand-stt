<script lang="ts">
  import { onDestroy } from "svelte";
  import { CancellablePromise } from "@wailsio/runtime";
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
  current.textToSpeech = {
    ...current.textToSpeech,
    enabled: true,
    baseURL: "https://speech.test/v1",
    model: "speech/tts",
    voice: "Default",
  };
  const saves = controlledSaves((request) => {
    current = structuredClone({ ...current, ...request.settings });
    return structuredClone(current);
  });
  const session = new Session(
    serviceWithStatus(() => CancellablePromise.resolve(idle), {
      settings: {
        SaveSettings: (request) =>
          saves.save(structuredClone($state.snapshot(request))),
      },
    }),
  );
  session.editor.applySettingsSnapshot(structuredClone(current));
  session.editor.devices = [
    { id: "desk-mic", name: "Desk microphone", default: true },
  ];
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

<div
  class="flex h-screen flex-col overflow-hidden bg-background text-foreground"
>
  <AppHeader
    bind:inputMode
    settings={session.editor.applied}
    onSettings={noop}
  />
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
