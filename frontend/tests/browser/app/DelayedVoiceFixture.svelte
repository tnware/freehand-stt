<script lang="ts">
  import { CancellablePromise } from "@wailsio/runtime";
  import { Session } from "$lib/stores/session.svelte";
  import { settings, idle, connectionResult, serviceWithStatus } from "$lib/stores/session-fixtures-data";
  import VoiceTranscriptionSettings from "$lib/components/home/VoiceTranscriptionSettings.svelte";
  const snapshot = structuredClone(settings);
  snapshot.savedConnections.selected = { ...snapshot.savedConnections.selected, voice: "a" };
  snapshot.voiceTranscription.model = "fixture/current";
  let release!: () => void;
  const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
  services.connection.TestSavedConnection = () => new CancellablePromise(resolve => {
    release = () => resolve({ ...connectionResult, modelIDs: ["fixture/discovered"] });
  });
  const session = new Session(services);
  session.editor.applySettingsSnapshot(snapshot);
  Object.assign(window, { delayedMetadata: { release: () => release() } });
</script>
<div class="mx-auto max-w-xl p-8">
  <VoiceTranscriptionSettings editor={session.editor} settings={session.editor.draft!} disabled={false} draft onAddConnection={() => {}} />
  <output data-testid="chosen-model">{session.editor.draft!.voiceTranscription.model}</output>
</div>
