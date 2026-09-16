<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import ManagedRuntimeControls from "./ManagedRuntimeControls.svelte";
  import VocabularyLink from "../settings/VocabularyLink.svelte";
  import RuntimeModelPicker from "../settings/RuntimeModelPicker.svelte";
  import LanguagePicker from "../settings/LanguagePicker.svelte";
  import QuickSaveStatus from "../settings/QuickSaveStatus.svelte";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import ConnectionSelect from "$lib/components/settings/ConnectionSelect.svelte";
  import { Purpose, type Change } from "$bindings/savedconnection";
  import { usesServerLoadedModel } from "$lib/utils/compatibility";
  import SaveIndicator from "../settings/SaveIndicator.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import WorkflowSection from "./WorkflowSection.svelte";
  import S1MiniControls from "$lib/components/settings/S1MiniControls.svelte";
  import * as Select from "$lib/components/ui/select";
  import { Switch } from "$lib/components/ui/switch";
  import {
    PostProcessingPreset,
    type ConnectionResult,
    type ProfileDescriptor,
    type Settings,
  } from "$lib/state";
  import type {
    QuickSettingsField,
    QuickSettingsPatch,
  } from "$lib/stores/editor.svelte";
  import {
    connectionStatusLabel,
    connectionSucceeded,
  } from "$lib/utils/connection";
  import { processingProfileName } from "$lib/utils/processingProfiles";
  import {
    readDisclosurePreference,
    writeDisclosurePreference,
  } from "$lib/utils/viewPreferences";

  let {
    runtime,
    runtimeWorkBusy = false,
    onManageRuntime = () => {},
    showTranscription = true,
    embedded = false,
    sidebar = false,
    showCleanup = true,
    onAddConnection,
    settings,
    processingProfiles,
    connection,
    processingConnection,
    sttStale = false,
    processingStale = false,
    pending = [],
    savedField = null,
    failedField = null,
    sttTesting = false,
    processingTesting = false,
    onUpdate,
    onChangeConnection,
    onEnterTranscription,
    onEnterCleanup,
    sttMetadataStatus = "idle",
    processingMetadataStatus = "idle",
    onTestConnection,
    onTestProcessingConnection,
    onOpenServerSettings,
    onOpenProcessingSettings,
    disabled = false,
  }: {
    /** Applied settings: every edit in this rack is persisted immediately. */
    runtime?: ManagedRuntimeState;
    runtimeWorkBusy?: boolean;
    onManageRuntime?: () => void;
    showTranscription?: boolean;
    embedded?: boolean;
    sidebar?: boolean;
    showCleanup?: boolean;
    onAddConnection?: (purpose: Purpose) => void;
    settings: Settings;
    processingProfiles: ProfileDescriptor[];
    connection: ConnectionResult | null;
    processingConnection: ConnectionResult | null;
    sttStale?: boolean;
    processingStale?: boolean;
    pending?: QuickSettingsField[];
    savedField?: QuickSettingsField | null;
    failedField?: QuickSettingsField | null;
    sttTesting?: boolean;
    processingTesting?: boolean;
    onChangeConnection: (change: Change) => Promise<boolean>;
    onUpdate: (
      patch: QuickSettingsPatch,
      field: QuickSettingsField,
    ) => Promise<boolean>;
    onEnterTranscription?: () => void;
    onEnterCleanup?: () => void;
    sttMetadataStatus?: "idle" | "loading" | "ready" | "empty" | "failed";
    processingMetadataStatus?:
      | "idle"
      | "loading"
      | "ready"
      | "empty"
      | "failed";
    onTestConnection: () => Promise<void>;
    onTestProcessingConnection: () => Promise<void>;
    onOpenServerSettings: () => void;
    onOpenProcessingSettings: () => void;
    disabled?: boolean;
  } = $props();
  const uid = $props.id();
  const controlID = (id: string) => (sidebar ? `${uid}-${id}` : id);
  const fileRuntime = $derived(
    runtime?.statusFor(settings.managedInstanceID ?? ""),
  );
  const fileModel = $derived(
    fileRuntime?.status.models?.find(
      (model) => model.id === fileRuntime.instance.model,
    ),
  );
  const fileProfile = $derived(
    fileModel?.behavior ??
      settings.modelProfiles.transcription?.find(
        (profile) => profile.id === settings.modelProfile,
      ),
  );
  const fileLanguageSupported = $derived(
    !!fileProfile?.capabilities.languageHint ||
      !!fileProfile?.languages?.length,
  );

  let sttOpen = $state(readDisclosurePreference("quick-stt", true));
  let cleanupOpen = $state(readDisclosurePreference("quick-cleanup", true));

  const serverLoadedModel = $derived(usesServerLoadedModel(settings));
  const processingEnabled = $derived(settings.postProcessing.enabled);
  const selectedProcessingProfile = $derived(
    processingProfiles.find(
      (profile) => profile.id === settings.postProcessing.preset,
    ),
  );

  function healthDot(
    result: ConnectionResult | null,
    enabled = true,
    stale = false,
  ): string {
    if (!enabled) return "bg-border";
    if (stale) return "bg-warning";
    if (!result) return "bg-muted-foreground";
    return connectionSucceeded(result) ? "bg-success" : "bg-destructive";
  }

  function latency(
    result: ConnectionResult | null,
    enabled = true,
    stale = false,
  ): string {
    if (!enabled) return "off";
    if (stale) return "stale";
    if (!result) return "not checked";
    if (!connectionSucceeded(result)) return connectionStatusLabel(result);
    return result.latencyMilliseconds > 0
      ? `${result.latencyMilliseconds.toLocaleString()} ms`
      : "ok";
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

  const isPending = (field: QuickSettingsField) => pending.includes(field);
  const sttHealthStale = $derived(sttStale || isPending("stt-model"));
  const processingHealthStale = $derived(
    processingStale || isPending("processing-model"),
  );
  function chooseProcessingProfile(value: string) {
    if (
      value === PostProcessingPreset.PostProcessingPresetGeneric ||
      value === PostProcessingPreset.PostProcessingPresetS1Mini
    )
      void onUpdate(
        { postProcessing: { preset: value } },
        "processing-profile",
      );
  }
</script>

{#snippet field(
  label: string,
  id: string,
  meta: import("svelte").Snippet,
  control: import("svelte").Snippet,
)}
  <div class="field">
    <div class="field-head">
      <label class="caption truncate" for={id}>{label}</label>
      <span class="field-meta">{@render meta()}</span>
    </div>
    <div class="field-control">{@render control()}</div>
  </div>
{/snippet}

<fieldset class="m-0 flex min-w-0 flex-col gap-2.5 border-0 p-0" {disabled}>
  {#if showTranscription}
    <WorkflowSection
      {sidebar}
      {embedded}
      label="Transcription"
      dot={sidebar ? "" : healthDot(connection, true, sttHealthStale)}
      meta={sidebar
        ? ""
        : embedded || sttOpen
          ? latency(connection, true, sttHealthStale)
          : collapsedConnectionSummary(
              serverLoadedModel ? "Server-loaded model" : settings.model,
              connection,
              true,
              sttHealthStale,
            )}
      metaTone={latencyTone(connection, true, sttHealthStale)}
      onSettings={onOpenServerSettings}
      settingsLabel="Open transcription settings"
      open={embedded ? undefined : sttOpen}
      controls={controlID("quick-stt-details")}
      onToggle={toggleSTT}
    >
      {#snippet icon()}{#if !sidebar}<ProviderIcon
            profile={settings.compatibilityProfile}
            size={20}
          />{/if}{/snippet}
      <div
        class={sidebar
          ? "flex min-w-0 flex-col gap-2"
          : "flex min-w-0 flex-col gap-3"}
      >
        {#snippet sttEndpointMeta()}{/snippet}
        {#snippet sttEndpointControl()}
          <ConnectionSelect
            id={controlID("quick-stt-endpoint")}
            catalog={settings.savedConnections}
            runtimeInstances={runtime?.instances}
            purpose={Purpose.Transcription}
            onAdd={onAddConnection
              ? () => onAddConnection(Purpose.Transcription)
              : undefined}
            compact
            disabled={disabled || pending.length > 0}
            onChange={onChangeConnection}
          />
        {/snippet}
        {@render field(
          "Connection",
          controlID("quick-stt-endpoint"),
          sttEndpointMeta,
          sttEndpointControl,
        )}

        {#if settings.managedInstanceID}
          {#if runtime}<ManagedRuntimeControls
              {sidebar}
              workBusy={runtimeWorkBusy}
              {runtime}
              instanceID={settings.managedInstanceID}
              disabled={disabled || pending.length > 0}
              onManage={onManageRuntime}
            />{/if}
          {#if !sidebar}<VocabularyLink {settings} />{/if}
        {:else}<RuntimeModelPicker
            {sidebar}
            id={controlID("quick-stt-model")}
            value={settings.model}
            compact
            immediate
            profileName={settings.modelProfiles.transcription?.find(
              (p) => p.id === settings.modelProfile,
            )?.name ?? settings.modelProfile}
            models={sttStale ? [] : (connection?.modelIDs ?? [])}
            savedModels={rememberedModels(settings, Purpose.Transcription).map(
              (e) => e.model,
            )}
            serverLoaded={serverLoadedModel}
            busy={sttTesting}
            disabled={disabled ||
              pending.length > 0 ||
              !settings.savedConnections.selected?.stt}
            onChoose={(model) => onUpdate({ model }, "stt-model")}
            onEnter={onEnterTranscription}
            metadataStatus={sttMetadataStatus}
            onDiscover={onTestConnection}
          />
        {/if}
        {#if sidebar}
          <div class="space-y-1.5">
            <label
              for={controlID("quick-file-language")}
              class="text-xs font-medium">Language</label
            >
            <LanguagePicker
              id={controlID("quick-file-language")}
              immediate
              bind:value={
                () => settings.language ?? "",
                (language) => {
                  if (!disabled && !pending.length && fileLanguageSupported)
                    void onUpdate({ language }, "stt-language");
                }
              }
              restricted={!!fileProfile?.languages?.length}
              languages={fileProfile?.languages?.length
                ? fileProfile.languages
                : (settings.transcriptionLanguages ?? [])}
              disabled={disabled ||
                pending.length > 0 ||
                !fileLanguageSupported}
              unavailableReason={!fileLanguageSupported
                ? "This model chooses the language automatically."
                : ""}
            />
          </div>
        {/if}
        <QuickSaveStatus
          quiet={sidebar}
          fields={["stt-model", "stt-language"]}
          {pending}
          saved={savedField}
          failed={failedField}
        />
      </div>
    </WorkflowSection>
  {/if}

  {#if showCleanup}
    <WorkflowSection
      {sidebar}
      {embedded}
      label={sidebar ? "Cleanup · shared" : "Cleanup"}
      dot={sidebar
        ? ""
        : healthDot(
            processingConnection,
            processingEnabled,
            processingHealthStale,
          )}
      meta={sidebar
        ? ""
        : embedded || cleanupOpen
          ? latency(
              processingConnection,
              processingEnabled,
              processingHealthStale,
            )
          : collapsedConnectionSummary(
              settings.postProcessing.model,
              processingConnection,
              processingEnabled,
              processingHealthStale,
            )}
      metaTone={latencyTone(
        processingConnection,
        processingEnabled,
        processingHealthStale,
      )}
      onSettings={onOpenProcessingSettings}
      settingsLabel="Open cleanup settings"
      open={embedded ? undefined : cleanupOpen}
      controls={controlID("quick-cleanup-details")}
      onToggle={toggleCleanup}
    >
      {#snippet icon()}{#if !sidebar}<ProviderIcon
            profile={settings.postProcessing.compatibilityProfile}
            size={20}
          />{/if}{/snippet}
      {#snippet actions()}
        <span class="flex shrink-0 items-center gap-1.5">
          <SaveIndicator
            pending={isPending("processing-enabled")}
            saved={savedField === "processing-enabled"}
            failed={failedField === "processing-enabled"}
          />
          <Switch
            id={controlID("quick-post-processing-enabled")}
            size="sm"
            disabled={disabled || pending.length > 0}
            bind:checked={
              () => processingEnabled,
              (enabled) => {
                if (!disabled && !pending.length)
                  void onUpdate(
                    { postProcessing: { enabled } },
                    "processing-enabled",
                  );
              }
            }
            aria-label="Post-process transcripts"
          />
        </span>
      {/snippet}

      <div
        class={sidebar
          ? "flex min-w-0 flex-col gap-2"
          : "flex min-w-0 flex-col gap-2.5"}
      >
        {#snippet cleanupEndpointMeta()}{/snippet}
        {#snippet cleanupEndpointControl()}
          <ConnectionSelect
            id={controlID("quick-processing-endpoint")}
            catalog={settings.savedConnections}
            runtimeInstances={runtime?.instances}
            purpose={Purpose.Cleanup}
            onAdd={onAddConnection
              ? () => onAddConnection(Purpose.Cleanup)
              : undefined}
            compact
            disabled={disabled || pending.length > 0}
            onChange={onChangeConnection}
          />
        {/snippet}
        {@render field(
          "Connection",
          controlID("quick-processing-endpoint"),
          cleanupEndpointMeta,
          cleanupEndpointControl,
        )}

        {#if settings.postProcessing.managedInstanceID}
          {#if runtime}<ManagedRuntimeControls
              {sidebar}
              workBusy={runtimeWorkBusy}
              {runtime}
              instanceID={settings.postProcessing.managedInstanceID}
              disabled={disabled || pending.length > 0}
              onManage={onManageRuntime}
            />{/if}
        {:else}<RuntimeModelPicker
            {sidebar}
            id={controlID("quick-processing-model")}
            value={settings.postProcessing.model}
            compact
            immediate
            profileName={processingProfileName(
              processingProfiles,
              settings.postProcessing.preset,
            )}
            models={processingStale
              ? []
              : (processingConnection?.modelIDs ?? [])}
            savedModels={rememberedModels(settings, Purpose.Cleanup).map(
              (e) => e.model,
            )}
            busy={processingTesting}
            disabled={disabled ||
              pending.length > 0 ||
              !settings.savedConnections.selected?.cleanup}
            onChoose={(model) =>
              onUpdate({ postProcessing: { model } }, "processing-model")}
            onEnter={onEnterCleanup}
            metadataStatus={processingMetadataStatus}
            onDiscover={onTestProcessingConnection}
          />
        {/if}
        {#snippet profileMeta()}
          <span class="inline-flex items-center gap-1.5">
            raw kept on failure
            <SaveIndicator
              pending={isPending("processing-profile")}
              saved={savedField === "processing-profile"}
              failed={failedField === "processing-profile"}
            />
          </span>
        {/snippet}
        {#snippet profileControl()}
          <Select.Root
            type="single"
            value={settings.postProcessing.preset}
            disabled={isPending("processing-profile") || !processingEnabled}
            onValueChange={chooseProcessingProfile}
          >
            <Select.Trigger
              id={controlID("quick-processing-profile")}
              class="w-full min-w-0 bg-well"
            >
              <span class="min-w-0 flex-1 truncate text-left text-[13px]">
                {processingProfileName(
                  processingProfiles,
                  settings.postProcessing.preset,
                )}
              </span>
            </Select.Trigger>
            <Select.Content>
              <Select.Group>
                <Select.Label>Request behavior</Select.Label>
                {#each processingProfiles as profile (profile.id)}
                  <Select.Item value={profile.id} label={profile.name}
                    >{profile.name}</Select.Item
                  >
                {/each}
              </Select.Group>
            </Select.Content>
          </Select.Root>
        {/snippet}
        {#if !sidebar && !settings.postProcessing.managedInstanceID}{@render field(
            "Model profile",
            controlID("quick-processing-profile"),
            profileMeta,
            profileControl,
          )}{/if}

        {#if !sidebar && settings.postProcessing.preset === PostProcessingPreset.PostProcessingPresetS1Mini && selectedProcessingProfile}
          <div class="min-w-0 border-t border-hairline pt-2.5">
            <S1MiniControls
              processor={settings.postProcessing}
              profile={selectedProcessingProfile}
              idPrefix={controlID("quick-s1-mini")}
              disabled={isPending("processing-controls") || !processingEnabled}
              compact
              onChange={(patch) =>
                onUpdate({ postProcessing: patch }, "processing-controls")}
            />
          </div>
        {/if}
        <QuickSaveStatus
          quiet={sidebar}
          fields={[
            "processing-model",
            "processing-profile",
            "processing-controls",
            "processing-enabled",
          ]}
          {pending}
          saved={savedField}
          failed={failedField}
        />
      </div>
    </WorkflowSection>
  {/if}
</fieldset>

<style>
  /*
   * Label and value sit on their own line above the control, so every control
   * in the section starts at one left edge and every reading ends at one right
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
    font-size: 0.75rem;
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
