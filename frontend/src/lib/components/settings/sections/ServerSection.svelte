<script lang="ts">
  import VocabularyLink from "../VocabularyLink.svelte";
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
  import { ID } from "$bindings/modelprofile";
  import RequestSettings from "$lib/components/settings/RequestSettings.svelte";
  let {
    settings = $bindable(),
    connection,
    busy = false,
    draftModels = [],
    onChooseModel,
    onForgetModel,
    connectionStale = false,
    onTestConnection,
  }: {
    settings: Settings;
    connection: ConnectionResult | null;
    busy?: boolean;
    draftModels?: string[];
    onChooseModel: (model: string) => boolean;
    onForgetModel: () => void;
    connectionStale?: boolean;
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
    profileName={compatibility?.name ?? settings.modelProfile}
    id="model"
    value={settings.model}
    {draftModels}
    onChoose={onChooseModel}
    onForget={onForgetModel}
    savedModels={rememberedModels(settings, Purpose.Transcription).map((e) => e.model)}
    models={connectionStale ? [] : (connection?.modelIDs ?? [])}
    serverLoaded={!!compatibility?.capabilities.serverLoadedModel &&
      !settings.compatibilityProfiles.transcription?.find(
        (p) => p.id === settings.compatibilityProfile,
      )?.capabilities.realtime}
    {busy}
    onDiscover={onTestConnection}
  />

  <ModelProfilePicker
    id="transcription-model-profile"
    value={settings.modelProfile}
    profiles={settings.modelProfiles.transcription ?? []}
    onChange={(id) => {
      settings.modelProfile = id;
      settings.transcriptionOptions = {
        prompt: "",
        hotwords: "",
        temperatureOverride: false,
        temperature: 0,
      };
      const languages = settings.modelProfiles.transcription?.find((p) => p.id === id)?.languages;
      if (languages?.length && !languages.some((l) => l.code === settings.language)) {
        settings.language = languages[0]?.code ?? "auto";
      }
    }}
  />
  <ValueRow
    id="language"
    label="Spoken language"
    hint="Choose the language in the recording, or let the model detect it. This does not translate audio."
  >
    {#snippet control()}
      <LanguagePicker
        id="language"
        restricted={!!compatibility?.languages?.length}
        languages={compatibility?.languages?.length
          ? compatibility.languages
          : (settings.transcriptionLanguages ?? [])}
        disabled={!compatibility?.capabilities.languageHint && !compatibility?.languages?.length}
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
</SettingsCard>

<VocabularyLink {settings} />
<TranscriptionControls
  bind:options={settings.transcriptionOptions}
  capabilities={compatibility?.capabilities}
/>

<RequestSettings {connection} stale={connectionStale} {busy} onCheck={onTestConnection}>
  <ValueRow
    id="file-transcription-timeout"
    label="Request timeout"
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
</RequestSettings>
