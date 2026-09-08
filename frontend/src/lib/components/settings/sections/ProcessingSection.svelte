<script lang="ts">
  import { Purpose } from "$bindings/savedconnection";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import { ID } from "$bindings/compatibility";
  import RuntimeModelPicker from "$lib/components/settings/RuntimeModelPicker.svelte";
  import CleanupControls from "$lib/components/settings/CleanupControls.svelte";
  import CustomInstructionEditor from "$lib/components/settings/CustomInstructionEditor.svelte";
  import ModelProfilePicker from "$lib/components/settings/ModelProfilePicker.svelte";
  import S1MiniProfileSettings from "$lib/components/settings/S1MiniProfileSettings.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Switch } from "$lib/components/ui/switch";
  import {
    PostProcessingPreset,
    type Settings,
    type ProfileDescriptor,
    type ConnectionResult,
  } from "$lib/state";
  import { processingProfile } from "$lib/utils/processingProfiles";
  import RequestSettings from "$lib/components/settings/RequestSettings.svelte";
  let {
    settings = $bindable(),
    profiles,
    connection,
    busy = false,
    draftModels = [],
    onChooseModel,
    onForgetModel,
    connectionStale = false,
    onTestConnection,
  }: {
    settings: Settings;
    profiles: ProfileDescriptor[];
    connection: ConnectionResult | null;
    busy?: boolean;
    draftModels?: string[];
    onChooseModel: (model: string) => boolean;
    onForgetModel: () => void;
    connectionStale?: boolean;
    onTestConnection: () => void;
  } = $props();
  const processor = $derived(settings.postProcessing);
  const compatibility = $derived(
    settings.modelProfiles.postProcessing?.find(
      (p) => String(p.id) === (processor.preset || ID.Generic),
    ),
  );
  const selectedProfile = $derived(processingProfile(profiles, processor.preset));
  function updateS1Mini(
    patch: Partial<Pick<Settings["postProcessing"], "styling" | "structure" | "context">>,
  ) {
    Object.assign(settings.postProcessing, patch);
  }
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <SettingRow
      title="Post-process completed transcripts"
      controlID="cleanup-enabled"
      compact
      description="Clean raw transcripts with the selected connection and model. Failures preserve the raw transcript."
      >{#snippet control()}<Switch
          id="cleanup-enabled"
          bind:checked={settings.postProcessing.enabled}
          aria-label="Post-process completed transcripts"
        />{/snippet}</SettingRow
    >
    <RuntimeModelPicker
      showProfileName={false}
      profileName={compatibility?.name ?? processor.preset}
      id="cleanup-model"
      value={settings.postProcessing.model}
      {draftModels}
      onChoose={onChooseModel}
      onForget={onForgetModel}
      savedModels={rememberedModels(settings, Purpose.Cleanup).map((e) => e.model)}
      models={connectionStale ? [] : (connection?.modelIDs ?? [])}
      {busy}
      onDiscover={onTestConnection}
    />

    <ModelProfilePicker
      id="cleanup-model-profile"
      value={processor.preset}
      profiles={settings.modelProfiles.postProcessing ?? []}
      onChange={(id) => {
        const profile = profiles.find((p) => String(p.id) === id);
        if (profile) settings.postProcessing.preset = profile.id;
      }}
    />
  </SettingsCard>

  {#if selectedProfile?.id === PostProcessingPreset.PostProcessingPresetS1Mini}
    <S1MiniProfileSettings
      processor={settings.postProcessing}
      profile={selectedProfile}
      onChange={updateS1Mini}
    />
  {:else if selectedProfile}
    <CustomInstructionEditor
      bind:value={settings.postProcessing.systemPrompt}
      recommended={selectedProfile.recommendedInstruction ?? ""}
      maximumBytes={selectedProfile.maximumInstructionBytes ?? 0}
    />
  {/if}

  <CleanupControls
    bind:options={settings.postProcessing.generationOptions}
    capabilities={compatibility?.capabilities}
    s1Mini={!!compatibility?.reasoningOffRequired}
  />

  <RequestSettings {connection} stale={connectionStale} {busy} onCheck={onTestConnection}>
    <ValueRow
      id="cleanup-timeout"
      label="Request timeout"
      hint="Maximum seconds for cleanup before falling back to the raw transcript."
      >{#snippet control()}<ValueInput
          id="cleanup-timeout"
          type="number"
          min={10}
          max={3600}
          step={10}
          bind:value={settings.postProcessing.timeoutSeconds}
        />{/snippet}</ValueRow
    >
    <p class="px-5 py-4 text-xs leading-relaxed text-muted-foreground">
      Raw transcription completes before cleanup starts. With history enabled, raw and cleaned text
      are saved together. Connection checks stop after 15 seconds. Cleanup requests are capped at 2
      MiB and responses at 1 MiB.
    </p>
  </RequestSettings>
</div>
