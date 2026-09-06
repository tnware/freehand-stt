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
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
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
      description="Clean raw transcripts with the selected connection and model. Failures preserve the raw transcript."
      >{#snippet control()}<Switch
          bind:checked={settings.postProcessing.enabled}
          aria-label="Post-process completed transcripts"
        />{/snippet}</SettingRow
    >
    <RuntimeModelPicker
      id="post-processing-model"
      value={settings.postProcessing.model}
      {draftModels}
      onChoose={onChooseModel}
      onForget={onForgetModel}
      savedModels={rememberedModels(settings, Purpose.Cleanup).map((e) => e.model)}
      models={connection?.modelIDs ?? []}
      {busy}
      onDiscover={onTestConnection}
    />
    {#if connection}
      <div class="p-5">
        <ConnectionDiagnostics
          result={connection}
          stale={connectionStale}
          {busy}
          onCheck={onTestConnection}
        />
      </div>
    {/if}
    <ModelProfilePicker
      id="cleanup-model-profile"
      value={processor.preset}
      profiles={settings.modelProfiles.postProcessing ?? []}
      onChange={(id) => {
        const profile = profiles.find((p) => String(p.id) === id);
        if (profile) settings.postProcessing.preset = profile.id;
      }}
    />
    <ValueRow
      id="post-processing-timeout"
      label="Request timeout"
      hint="Maximum seconds for cleanup before falling back to the raw transcript."
      >{#snippet control()}<ValueInput
          id="post-processing-timeout"
          type="number"
          min={10}
          max={3600}
          step={10}
          bind:value={settings.postProcessing.timeoutSeconds}
        />{/snippet}</ValueRow
    >
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

  <p class="px-1 text-xs leading-relaxed text-muted-foreground">
    Raw transcription always completes first. With history enabled, raw and processed text are kept
    together for comparison. A processor error never turns the transcription into a failure. The
    selected behavior determines the request format, not which endpoint or model you may use.
    S1-mini remains an explicit specialized profile rather than the default for all models. The
    connection check stops after 15 seconds; processing requests and responses have fixed 2 MiB and
    1 MiB safety ceilings.
  </p>
</div>
