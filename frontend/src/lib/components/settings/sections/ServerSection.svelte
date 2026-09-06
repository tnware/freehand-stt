<script lang="ts">
  import { Purpose } from "$bindings/savedconnection";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import ModelProfilePicker from "$lib/components/settings/ModelProfilePicker.svelte";
  import LanguagePicker from "$lib/components/settings/LanguagePicker.svelte";
  import RuntimeModelPicker from "$lib/components/settings/RuntimeModelPicker.svelte";
  import TranscriptionControls from "$lib/components/settings/TranscriptionControls.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Badge } from "$lib/components/ui/badge";
  import { PostProcessingPreset, type Settings, type ConnectionResult } from "$lib/state";
  import { ID } from "$bindings/compatibility";
  import { connectionDescription } from "$lib/utils/connection";
  let {
    settings = $bindable(),
    connection,
    busy = false,
    draftModels = [],
    onChooseModel,
    onForgetModel,
    onTestConnection,
  }: {
    settings: Settings;
    connection: ConnectionResult | null;
    busy?: boolean;
    draftModels?: string[];
    onChooseModel: (model: string) => boolean;
    onForgetModel: () => void;
    onTestConnection: () => void;
  } = $props();
  const compatibility = $derived(
    settings.modelProfiles.transcription?.find(
      (p) => p.id === (settings.modelProfile || ID.Generic),
    ),
  );
  function updateFileTimeoutMinutes(event: Event) {
    const n = (event.currentTarget as HTMLInputElement).valueAsNumber;
    if (Number.isFinite(n)) settings.fileTranscriptionTimeoutSeconds = Math.round(n * 60);
  }
</script>

<SettingsCard>
  <RuntimeModelPicker
    id="model"
    value={settings.model}
    {draftModels}
    onChoose={onChooseModel}
    onForget={onForgetModel}
    savedModels={rememberedModels(settings, Purpose.Transcription).map((e) => e.model)}
    models={connection?.modelIDs ?? []}
    serverLoaded={!!compatibility?.capabilities.serverLoadedModel}
    {busy}
    onDiscover={onTestConnection}
  />
  <ModelProfilePicker
    id="transcription-model-profile"
    value={settings.modelProfile}
    profiles={settings.modelProfiles.transcription ?? []}
    onChange={(id) => (settings.modelProfile = id)}
  />
  <ValueRow
    id="language"
    label="Language"
    hint="Used for microphone and file transcription. Server default leaves the language unset; Automatic detection uses the selected provider’s detection contract. This does not request translation."
  >
    {#snippet control()}
      <LanguagePicker
        id="language"
        languages={settings.transcriptionLanguages ?? []}
        disabled={!compatibility?.capabilities.languageHint}
        bind:value={() => settings.language ?? "", (value) => (settings.language = value)}
      />
    {/snippet}
  </ValueRow>

  {#if settings.postProcessing.enabled && settings.postProcessing.preset === PostProcessingPreset.PostProcessingPresetS1Mini}
    <p class="px-5 pb-3 text-xs leading-relaxed text-muted-foreground">
      S1-mini cleanup is English only. Non-English selections or detected results keep the raw
      transcript. When no language is known, S1-mini runs assuming English. Choose custom cleanup or
      turn cleanup off for other languages.
    </p>
  {/if}

  <ValueRow
    id="transcription-timeout"
    label="Recording request timeout"
    hint="Maximum time for each microphone transcription request after its audio is captured. Checkpoints each receive a fresh budget."
  >
    {#snippet control()}
      <ValueInput
        id="transcription-timeout"
        type="number"
        min={10}
        max={3600}
        step={10}
        bind:value={settings.transcriptionTimeoutSeconds}
      />
    {/snippet}
    {#snippet action()}<Badge variant="outline">seconds</Badge>{/snippet}
  </ValueRow>

  <ValueRow
    id="file-transcription-timeout"
    label="Stored audio timeout"
    hint="Maximum time for one stored-file upload and transcription, including a streamed response."
  >
    {#snippet control()}
      <ValueInput
        id="file-transcription-timeout"
        type="number"
        min={1}
        max={1440}
        step={1}
        value={Math.round(settings.fileTranscriptionTimeoutSeconds / 60)}
        oninput={updateFileTimeoutMinutes}
      />
    {/snippet}
    {#snippet action()}<Badge variant="outline">minutes</Badge>{/snippet}
  </ValueRow>
</SettingsCard>
{#if connection}<p role="status" class="text-xs text-muted-foreground">
    {connectionDescription(connection)}
  </p>{/if}
<TranscriptionControls
  bind:options={settings.transcriptionOptions}
  capabilities={compatibility?.capabilities}
/>
