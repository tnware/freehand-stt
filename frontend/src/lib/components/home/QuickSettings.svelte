<script lang="ts">
  import CheckIcon from "@lucide/svelte/icons/check";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import RackModule from "$lib/components/home/RackModule.svelte";
  import QuickControls from "$lib/components/home/QuickControls.svelte";
  import S1MiniControls from "$lib/components/settings/S1MiniControls.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Select from "$lib/components/ui/select";
  import { Switch } from "$lib/components/ui/switch";
  import {
    ConnectionProbe,
    PostProcessingPreset,
    type ConnectionResult,
    type Device,
    type ProfileDescriptor,
    type Settings,
  } from "$lib/state";
  import type { QuickSettingsField, QuickSettingsPatch } from "$lib/stores/session.svelte";
  import { connectionStatusLabel, connectionSucceeded, modelPresenceLabel } from "$lib/utils/connection";
  import { processingProfileName } from "$lib/utils/processingProfiles";
  import {
    readDisclosurePreference,
    writeDisclosurePreference,
  } from "$lib/utils/viewPreferences";
  import { cn } from "$lib/utils";

  let {
    settings,
    devices,
    processingProfiles,
    connection,
    processingConnection,
    sttStale = false,
    processingStale = false,
    pending = [],
    savedField = null,
    sttTesting = false,
    processingTesting = false,
    onUpdate,
    onTestConnection,
    onTestProcessingConnection,
    onOpenServerSettings,
    onOpenProcessingSettings,
    onOpenAudioSettings,
    onOpenDeliverySettings,
    disabled = false,
  }: {
    /** Applied settings: every edit in this rack is persisted immediately. */
    settings: Settings;
    devices: Device[];
    processingProfiles: ProfileDescriptor[];
    connection: ConnectionResult | null;
    processingConnection: ConnectionResult | null;
    sttStale?: boolean;
    processingStale?: boolean;
    pending?: QuickSettingsField[];
    savedField?: QuickSettingsField | null;
    sttTesting?: boolean;
    processingTesting?: boolean;
    onUpdate: (patch: QuickSettingsPatch, field: QuickSettingsField) => Promise<boolean>;
    onTestConnection: () => Promise<void>;
    onTestProcessingConnection: () => Promise<void>;
    onOpenServerSettings: () => void;
    onOpenProcessingSettings: () => void;
    onOpenAudioSettings: () => void;
    onOpenDeliverySettings: () => void;
    disabled?: boolean;
  } = $props();

  let endpointDraft = $state("");
  let modelDraft = $state("");
  let processingEndpointDraft = $state("");
  let processingModelDraft = $state("");
  let endpointTouched = $state(false);
  let modelTouched = $state(false);
  let processingEndpointTouched = $state(false);
  let processingModelTouched = $state(false);
  let sttOpen = $state(readDisclosurePreference("quick-stt", true));
  let cleanupOpen = $state(readDisclosurePreference("quick-cleanup", true));

  let endpointSave: Promise<boolean> | undefined;
  let modelSave: Promise<boolean> | undefined;
  let processingEndpointSave: Promise<boolean> | undefined;
  let processingModelSave: Promise<boolean> | undefined;

  $effect(() => {
    if (!endpointTouched) endpointDraft = settings.baseURL;
    if (!modelTouched) modelDraft = settings.model;
    if (!processingEndpointTouched) processingEndpointDraft = settings.postProcessing.baseURL;
    if (!processingModelTouched) processingModelDraft = settings.postProcessing.model;
  });

  // The native Settings window owns configuration while visible. Drop any
  // uncommitted field-local text so it cannot overwrite a newer backend
  // snapshot when the rack becomes interactive again.
  $effect(() => {
    if (!disabled) return;
    endpointDraft = settings.baseURL;
    modelDraft = settings.model;
    processingEndpointDraft = settings.postProcessing.baseURL;
    processingModelDraft = settings.postProcessing.model;
    endpointTouched = false;
    modelTouched = false;
    processingEndpointTouched = false;
    processingModelTouched = false;
  });

  const processingEnabled = $derived(settings.postProcessing.enabled);
  const discoveredModels = $derived(connection?.modelIDs ?? []);
  const processingDiscoveredModels = $derived(processingConnection?.modelIDs ?? []);
  const selectedProcessingProfile = $derived(
    processingProfiles.find((profile) => profile.id === settings.postProcessing.preset),
  );

  function modelMetadata(result: ConnectionResult | null, model: string): string {
    if (!result || result.probe !== ConnectionProbe.ConnectionProbeModels) return "Manual";
    if (result.modelIDs?.length) return `${result.modelIDs.length.toLocaleString()} found`;
    return modelPresenceLabel(result) || (model ? "Manual" : "Choose");
  }

  function healthDot(result: ConnectionResult | null, enabled = true, stale = false): string {
    if (!enabled) return "bg-border";
    if (stale) return "bg-warning";
    if (!result) return "bg-muted-foreground";
    return connectionSucceeded(result) ? "bg-success" : "bg-destructive";
  }

  function latency(result: ConnectionResult | null, enabled = true, stale = false): string {
    if (!enabled) return "off";
    if (stale) return "stale";
    if (!result) return "not checked";
    if (!connectionSucceeded(result)) return connectionStatusLabel(result);
    return result.latencyMilliseconds > 0 ? `${result.latencyMilliseconds.toLocaleString()} ms` : "ok";
  }

  function latencyTone(
    result: ConnectionResult | null,
    enabled = true,
    stale = false,
  ): "quiet" | "ok" | "warn" | "bad" {
    if (!enabled) return "quiet";
    if (stale) return "warn";
    if (!result) return "quiet";
    return connectionSucceeded(result) ? "ok" : "bad";
  }

  function collapsedConnectionSummary(
    model: string,
    result: ConnectionResult | null,
    enabled = true,
    stale = false,
  ): string {
    const status = latency(result, enabled, stale);
    return model.trim() ? `${model.trim()} · ${status}` : status;
  }

  function toggleSTT() {
    sttOpen = !sttOpen;
    writeDisclosurePreference("quick-stt", sttOpen);
  }

  function toggleCleanup() {
    cleanupOpen = !cleanupOpen;
    writeDisclosurePreference("quick-cleanup", cleanupOpen);
  }

  function handleDraftKey(event: KeyboardEvent, reset: () => void) {
    if (event.key === "Enter") (event.currentTarget as HTMLInputElement).blur();
    if (event.key === "Escape") {
      reset();
      (event.currentTarget as HTMLInputElement).blur();
    }
  }

  const isPending = (field: QuickSettingsField): boolean => pending.includes(field);
  const sttHealthStale = $derived(
    sttStale || endpointTouched || modelTouched || isPending("stt-endpoint") || isPending("stt-model"),
  );
  const processingHealthStale = $derived(
    processingStale ||
      processingEndpointTouched ||
      processingModelTouched ||
      isPending("processing-endpoint") ||
      isPending("processing-model"),
  );
  const panelFields: QuickSettingsField[] = [
    "stt-endpoint",
    "stt-model",
    "processing-enabled",
    "processing-endpoint",
    "processing-model",
    "processing-profile",
    "processing-controls",
  ];

  function commitEndpoint(): Promise<boolean> {
    const value = endpointDraft.trim();
    if (value === settings.baseURL) {
      endpointTouched = false;
      return Promise.resolve(true);
    }
    if (endpointSave) return endpointSave;
    endpointSave = onUpdate({ baseURL: value }, "stt-endpoint")
      .then((saved) => {
        if (saved) endpointTouched = false;
        return saved;
      })
      .finally(() => (endpointSave = undefined));
    return endpointSave;
  }

  function commitModel(): Promise<boolean> {
    const value = modelDraft.trim();
    if (value === settings.model) {
      modelTouched = false;
      return Promise.resolve(true);
    }
    if (modelSave) return modelSave;
    modelSave = onUpdate({ model: value }, "stt-model")
      .then((saved) => {
        if (saved) modelTouched = false;
        return saved;
      })
      .finally(() => (modelSave = undefined));
    return modelSave;
  }

  function chooseModel(value: string) {
    if (value && value !== settings.model) void onUpdate({ model: value }, "stt-model");
  }

  function commitProcessingEndpoint(): Promise<boolean> {
    const value = processingEndpointDraft.trim();
    if (value === settings.postProcessing.baseURL) {
      processingEndpointTouched = false;
      return Promise.resolve(true);
    }
    if (processingEndpointSave) return processingEndpointSave;
    processingEndpointSave = onUpdate({ postProcessing: { baseURL: value } }, "processing-endpoint")
      .then((saved) => {
        if (saved) processingEndpointTouched = false;
        return saved;
      })
      .finally(() => (processingEndpointSave = undefined));
    return processingEndpointSave;
  }

  function commitProcessingModel(): Promise<boolean> {
    const value = processingModelDraft.trim();
    if (value === settings.postProcessing.model) {
      processingModelTouched = false;
      return Promise.resolve(true);
    }
    if (processingModelSave) return processingModelSave;
    processingModelSave = onUpdate({ postProcessing: { model: value } }, "processing-model")
      .then((saved) => {
        if (saved) processingModelTouched = false;
        return saved;
      })
      .finally(() => (processingModelSave = undefined));
    return processingModelSave;
  }

  function chooseProcessingModel(value: string) {
    if (value && value !== settings.postProcessing.model) {
      void onUpdate({ postProcessing: { model: value } }, "processing-model");
    }
  }

  function chooseProcessingProfile(value: string) {
    if (
      value !== PostProcessingPreset.PostProcessingPresetGeneric &&
      value !== PostProcessingPreset.PostProcessingPresetS1Mini
    ) {
      return;
    }
    if (value !== settings.postProcessing.preset) {
      void onUpdate({ postProcessing: { preset: value } }, "processing-profile");
    }
  }

  async function testSTTConnection() {
    if ((await commitEndpoint()) && (await commitModel())) await onTestConnection();
  }

  async function testProcessingConnection() {
    if ((await commitProcessingEndpoint()) && (await commitProcessingModel())) {
      await onTestProcessingConnection();
    }
  }

  const rackAnnouncement = $derived.by(() => {
    if (sttTesting) return "Testing speech-to-text connection.";
    if (processingTesting) return "Testing post-processing connection.";
    if (pending.some((field) => panelFields.includes(field))) return "Saving quick settings.";
    if (!savedField || !panelFields.includes(savedField)) return "";
    if (savedField === "stt-endpoint") return "Speech-to-text endpoint saved.";
    if (savedField === "stt-model") return "Speech-to-text model saved.";
    if (savedField === "processing-endpoint") return "Post-processing endpoint saved.";
    if (savedField === "processing-model") return "Post-processing model saved.";
    if (savedField === "processing-profile") return "Post-processing behavior saved.";
    if (savedField === "processing-enabled") return "Post-processing preference saved.";
    if (savedField === "processing-controls") return "Post-processing controls saved.";
    return "";
  });
</script>

{#snippet field(label: string, id: string, meta: import("svelte").Snippet, control: import("svelte").Snippet)}
  <div class="field">
    <div class="field-head">
      <label class="caption truncate" for={id}>{label}</label>
      <span class="field-meta">{@render meta()}</span>
    </div>
    <div class="field-control">{@render control()}</div>
  </div>
{/snippet}

<fieldset class="m-0 flex min-w-0 flex-col gap-2.5 border-0 p-0" {disabled}>
  <span class="sr-only" role="status" aria-live="polite" aria-atomic="true">
    {rackAnnouncement}
  </span>

  <QuickControls
    {settings}
    {devices}
    {pending}
    {savedField}
    {onUpdate}
    {onOpenAudioSettings}
    {onOpenDeliverySettings}
    {disabled}
  />

  <RackModule
    label="Speech to text"
    dot={healthDot(connection, true, sttHealthStale)}
    meta={sttOpen
      ? latency(connection, true, sttHealthStale)
      : collapsedConnectionSummary(settings.model, connection, true, sttHealthStale)}
    metaTone={latencyTone(connection, true, sttHealthStale)}
    onSettings={onOpenServerSettings}
    settingsLabel="Open transcription settings"
    open={sttOpen}
    controls="quick-stt-details"
    onToggle={toggleSTT}
  >
    <div class="contents">
      {#snippet sttEndpointMeta()}
        {#if isPending("stt-endpoint")}
          saving
        {:else if endpointTouched}
          edited
        {:else if savedField === "stt-endpoint"}
          saved
        {/if}
      {/snippet}
      {#snippet sttEndpointControl()}
        <ValueInput
          id="quick-stt-endpoint"
          type="url"
          class="figure h-[30px] min-w-0 flex-1 bg-well text-[11px]"
          bind:value={endpointDraft}
          disabled={isPending("stt-endpoint")}
          spellcheck={false}
          oninput={() => (endpointTouched = true)}
          onblur={() => void commitEndpoint()}
          onkeydown={(event) =>
            handleDraftKey(event, () => {
              endpointDraft = settings.baseURL;
              endpointTouched = false;
            })}
        />
        <Button
          variant="ghost"
          size="sm"
          class="h-[30px] shrink-0 border border-accent-edge bg-accent-wash px-2.5 text-accent-text hover:bg-accent-wash-strong"
          disabled={sttTesting || isPending("stt-endpoint") || isPending("stt-model")}
          onclick={() => void testSTTConnection()}
        >
          {#if sttTesting}<LoaderCircleIcon class="animate-spin" />{/if}
          {sttTesting ? "Testing" : "Test"}
        </Button>
      {/snippet}
      {@render field("Endpoint", "quick-stt-endpoint", sttEndpointMeta, sttEndpointControl)}

      {#snippet sttModelMeta()}
        {#if isPending("stt-model")}
          <LoaderCircleIcon class="inline size-3 animate-spin" />
        {:else if modelTouched}
          edited
        {:else if savedField === "stt-model"}
          <CheckIcon class="inline size-3 text-success" />
        {:else}
          {modelMetadata(connection, settings.model)}
        {/if}
      {/snippet}
      {#snippet sttModelControl()}
        {#if discoveredModels.length > 0}
          <Select.Root
            type="single"
            value={settings.model}
            disabled={isPending("stt-model")}
            onValueChange={chooseModel}
          >
            <Select.Trigger id="quick-stt-model" size="sm" class="h-[30px] w-full min-w-0 bg-well">
              <span class="figure min-w-0 flex-1 truncate text-left text-[11.5px]">
                {settings.model || "Choose a discovered model"}
              </span>
            </Select.Trigger>
            <Select.Content class="max-h-72">
              <Select.Group>
                <Select.Label>Discovered models</Select.Label>
                {#each discoveredModels as model (model)}
                  <Select.Item value={model} label={model}>{model}</Select.Item>
                {/each}
              </Select.Group>
            </Select.Content>
          </Select.Root>
        {:else}
          <ValueInput
            id="quick-stt-model"
            class="figure h-[30px] min-w-0 flex-1 bg-well text-[11.5px]"
            bind:value={modelDraft}
            disabled={isPending("stt-model")}
            spellcheck={false}
            oninput={() => (modelTouched = true)}
            onblur={() => void commitModel()}
            onkeydown={(event) =>
              handleDraftKey(event, () => {
                modelDraft = settings.model;
                modelTouched = false;
              })}
          />
        {/if}
      {/snippet}
      {@render field("Model", "quick-stt-model", sttModelMeta, sttModelControl)}
    </div>
  </RackModule>

  <RackModule
    label="Cleanup"
    dot={healthDot(processingConnection, processingEnabled, processingHealthStale)}
    meta={cleanupOpen
      ? latency(processingConnection, processingEnabled, processingHealthStale)
      : collapsedConnectionSummary(
          settings.postProcessing.model,
          processingConnection,
          processingEnabled,
          processingHealthStale,
        )}
    metaTone={latencyTone(processingConnection, processingEnabled, processingHealthStale)}
    onSettings={onOpenProcessingSettings}
    settingsLabel="Open post-processing settings"
    open={cleanupOpen}
    controls="quick-cleanup-details"
    onToggle={toggleCleanup}
  >
    {#snippet actions()}
      <span class="flex shrink-0 items-center gap-1.5">
        {#if isPending("processing-enabled")}
          <LoaderCircleIcon class="size-3 animate-spin text-ink-quiet" aria-label="Saving" />
        {:else if savedField === "processing-enabled"}
          <CheckIcon class="size-3 text-success" aria-label="Saved" />
        {/if}
        <Switch
          id="quick-post-processing-enabled"
          size="sm"
          checked={processingEnabled}
          disabled={isPending("processing-enabled")}
          onCheckedChange={(enabled) =>
            void onUpdate({ postProcessing: { enabled } }, "processing-enabled")}
          aria-label="Post-process transcripts"
        />
      </span>
    {/snippet}

    <div class={cn("flex min-w-0 flex-col gap-2.5", !processingEnabled && "opacity-60")}>
      {#snippet cleanupEndpointMeta()}
        {#if isPending("processing-endpoint")}
          saving
        {:else if processingEndpointTouched}
          edited
        {:else if savedField === "processing-endpoint"}
          saved
        {/if}
      {/snippet}
      {#snippet cleanupEndpointControl()}
        <ValueInput
          id="quick-processing-endpoint"
          type="url"
          class="figure h-[30px] min-w-0 flex-1 bg-well text-[11px]"
          bind:value={processingEndpointDraft}
          disabled={isPending("processing-endpoint")}
          spellcheck={false}
          oninput={() => (processingEndpointTouched = true)}
          onblur={() => void commitProcessingEndpoint()}
          onkeydown={(event) =>
            handleDraftKey(event, () => {
              processingEndpointDraft = settings.postProcessing.baseURL;
              processingEndpointTouched = false;
            })}
        />
        <Button
          variant="ghost"
          size="sm"
          class="h-[30px] shrink-0 border border-accent-edge bg-accent-wash px-2.5 text-accent-text hover:bg-accent-wash-strong"
          disabled={processingTesting ||
            !processingEnabled ||
            isPending("processing-endpoint") ||
            isPending("processing-model")}
          onclick={() => void testProcessingConnection()}
        >
          {#if processingTesting}<LoaderCircleIcon class="animate-spin" />{/if}
          {processingTesting ? "Testing" : "Test"}
        </Button>
      {/snippet}
      {@render field("Endpoint", "quick-processing-endpoint", cleanupEndpointMeta, cleanupEndpointControl)}

      {#snippet cleanupModelMeta()}
        {#if isPending("processing-model")}
          <LoaderCircleIcon class="inline size-3 animate-spin" />
        {:else if processingModelTouched}
          edited
        {:else if savedField === "processing-model"}
          <CheckIcon class="inline size-3 text-success" />
        {:else}
          {modelMetadata(processingConnection, settings.postProcessing.model)}
        {/if}
      {/snippet}
      {#snippet cleanupModelControl()}
        {#if processingDiscoveredModels.length > 0}
          <Select.Root
            type="single"
            value={settings.postProcessing.model}
            disabled={isPending("processing-model")}
            onValueChange={chooseProcessingModel}
          >
            <Select.Trigger
              id="quick-processing-model"
              size="sm"
              class="h-[30px] w-full min-w-0 bg-well"
            >
              <span class="figure min-w-0 flex-1 truncate text-left text-[11.5px]">
                {settings.postProcessing.model || "Choose a discovered model"}
              </span>
            </Select.Trigger>
            <Select.Content class="max-h-72">
              <Select.Group>
                <Select.Label>Discovered models</Select.Label>
                {#each processingDiscoveredModels as model (model)}
                  <Select.Item value={model} label={model}>{model}</Select.Item>
                {/each}
              </Select.Group>
            </Select.Content>
          </Select.Root>
        {:else}
          <ValueInput
            id="quick-processing-model"
            class="figure h-[30px] min-w-0 flex-1 bg-well text-[11.5px]"
            bind:value={processingModelDraft}
            disabled={isPending("processing-model")}
            spellcheck={false}
            oninput={() => (processingModelTouched = true)}
            onblur={() => void commitProcessingModel()}
            onkeydown={(event) =>
              handleDraftKey(event, () => {
                processingModelDraft = settings.postProcessing.model;
                processingModelTouched = false;
              })}
          />
        {/if}
      {/snippet}
      {@render field("Model", "quick-processing-model", cleanupModelMeta, cleanupModelControl)}

      {#snippet profileMeta()}
        {#if isPending("processing-profile")}
          <LoaderCircleIcon class="inline size-3 animate-spin" />
        {:else if savedField === "processing-profile"}
          <CheckIcon class="inline size-3 text-success" />
        {:else}
          raw kept on failure
        {/if}
      {/snippet}
      {#snippet profileControl()}
        <Select.Root
          type="single"
          value={settings.postProcessing.preset}
          disabled={isPending("processing-profile") || !processingEnabled}
          onValueChange={chooseProcessingProfile}
        >
          <Select.Trigger
            id="quick-processing-profile"
            size="sm"
            class="h-[30px] w-full min-w-0 bg-well"
          >
            <span class="min-w-0 flex-1 truncate text-left text-[11.5px]">
              {processingProfileName(processingProfiles, settings.postProcessing.preset)}
            </span>
          </Select.Trigger>
          <Select.Content>
            <Select.Group>
              <Select.Label>Request behavior</Select.Label>
              {#each processingProfiles as profile (profile.id)}
                <Select.Item value={profile.id} label={profile.name}>{profile.name}</Select.Item>
              {/each}
            </Select.Group>
          </Select.Content>
        </Select.Root>
      {/snippet}
      {@render field("Profile", "quick-processing-profile", profileMeta, profileControl)}

      {#if settings.postProcessing.preset === PostProcessingPreset.PostProcessingPresetS1Mini && selectedProcessingProfile}
        <div class="min-w-0 border-t border-hairline pt-2.5">
          {#if isPending("processing-controls") || savedField === "processing-controls"}
            <div
              class="figure mb-1.5 flex h-4 items-center justify-end text-[10px] text-ink-quiet"
              aria-live="polite"
            >
              {#if isPending("processing-controls")}
                <LoaderCircleIcon class="mr-1.5 size-3 animate-spin" />saving
              {:else}
                <CheckIcon class="mr-1.5 size-3 text-success" />saved
              {/if}
            </div>
          {/if}
          <S1MiniControls
            processor={settings.postProcessing}
            profile={selectedProcessingProfile}
            idPrefix="quick-s1-mini"
            disabled={isPending("processing-controls") || !processingEnabled}
            compact
            onChange={(patch) => onUpdate({ postProcessing: patch }, "processing-controls")}
          />
        </div>
      {/if}
    </div>
  </RackModule>

</fieldset>

<style>
  /*
   * Label and value sit on their own line above the control, so every control
   * in the rack starts at one left edge and every reading ends at one right
   * edge. Inline labels could not: "Endpoint" and "Profile" are different
   * widths, which gave the controls two left edges, and a long model name in a
   * select had nowhere to go but over the reading beside it.
   */
  .field {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 0.3125rem;
  }
  .field-head {
    display: flex;
    min-width: 0;
    align-items: baseline;
    gap: 0.5rem;
  }
  .field-meta {
    margin-left: auto;
    flex: 0 0 auto;
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    font-size: 0.625rem;
    line-height: 1.2;
    color: var(--ink-quiet);
    white-space: nowrap;
  }
  .field-meta:empty {
    display: none;
  }
  .field-control {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 0.5rem;
  }
</style>
