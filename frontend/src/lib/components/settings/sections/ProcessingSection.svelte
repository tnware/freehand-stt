<script lang="ts">
  import { ID } from "$bindings/compatibility";
  import CleanupControls from "$lib/components/settings/CleanupControls.svelte";
  import CompatibilityProfilePicker from "$lib/components/settings/CompatibilityProfilePicker.svelte";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import * as Alert from "$lib/components/ui/alert";
  import * as Select from "$lib/components/ui/select";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import CustomInstructionEditor from "$lib/components/settings/CustomInstructionEditor.svelte";
  import ProcessingProfilePicker from "$lib/components/settings/ProcessingProfilePicker.svelte";
  import S1MiniProfileSettings from "$lib/components/settings/S1MiniProfileSettings.svelte";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import {
    ConnectionProbe,
    ModelPresence,
    PostProcessingPreset,
    type ConnectionResult,
    type ProfileDescriptor,
    type Settings,
  } from "$lib/state";
  import {
    connectionDescription,
    connectionProbeLabel,
    connectionStatusLabel,
    connectionSucceeded,
  } from "$lib/utils/connection";
  import { processingProfile } from "$lib/utils/processingProfiles";

  let {
    settings = $bindable(),
    apiKey = $bindable(),
    clearKey = $bindable(),
    profiles,
    connection,
    busy = false,
    onTestConnection,
  }: {
    settings: Settings;
    apiKey: string;
    clearKey: boolean;
    profiles: ProfileDescriptor[];
    connection: ConnectionResult | null;
    busy?: boolean;
    onTestConnection: () => void;
  } = $props();

  const processor = $derived(settings.postProcessing);
  const compatibility = $derived(
    settings.compatibilityProfiles.postProcessing?.find(
      (profile) => profile.id === (processor.compatibilityProfile || ID.Generic),
    ),
  );
  const selectedProfile = $derived(processingProfile(profiles, processor.preset));
  const discoveredModels = $derived(connection?.modelIDs ?? []);
  const canSelectModel = $derived(discoveredModels.length > 0);
  const modelPresence = $derived.by(() => {
    if (!connection || connection.probe !== ConnectionProbe.ConnectionProbeModels) return "Manual";
    if (discoveredModels.includes(processor.model)) return "Listed";
    if (connection.modelPresence === ModelPresence.ModelPresenceUnavailable) return "Unavailable";
    return processor.model ? "Not listed" : "Choose a model";
  });
  const credentialStatus = $derived(
    apiKey.trim()
      ? "Draft entered"
      : settings.postProcessingCredentialConfigured
        ? "Configured"
        : "Optional",
  );
  const connectionHint = $derived.by(() => {
    if (!connection) return "Checks /models metadata without invoking the language model.";
    const checked = new Date(connection.checkedAt).toLocaleString([], {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
    const details = [
      connection.requestedURL,
      connectionProbeLabel(connection),
      `${connection.latencyMilliseconds.toLocaleString()} ms`,
      connection.httpStatus ? `HTTP ${connection.httpStatus}` : "",
      checked,
    ].filter(Boolean);
    return `${connectionDescription(connection)} ${details.join(" · ")}`;
  });

  function updateS1Mini(
    patch: Partial<Pick<Settings["postProcessing"], "styling" | "structure" | "context">>,
  ) {
    Object.assign(settings.postProcessing, patch);
  }
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <CompatibilityProfilePicker
      id="postProcessing-compatibility-profile"
      bind:value={settings.postProcessing.compatibilityProfile}
      profiles={settings.compatibilityProfiles.postProcessing ?? []}
    />
    <SettingRow
      title="Post-process completed transcripts"
      description="Send successful raw transcripts to a separate OpenAI-compatible /v1/chat/completions endpoint. Any failure falls back to the raw transcript."
    >
      {#snippet control()}
        <Switch
          id="post-processing-enabled"
          bind:checked={settings.postProcessing.enabled}
          aria-label="Post-process completed transcripts"
        />
      {/snippet}
    </SettingRow>

    <ValueRow
      id="post-processing-base-url"
      label="Base URL"
      hint="OpenAI-compatible /v1/chat/completions endpoint."
    >
      {#snippet control()}
        <ValueInput
          id="post-processing-base-url"
          type="url"
          bind:value={settings.postProcessing.baseURL}
          spellcheck={false}
        />
      {/snippet}
    </ValueRow>

    <SettingRow title="Connection" description={connectionHint}>
      {#snippet control()}
        <div class="flex items-center gap-2">
          <Badge
            variant={!connection
              ? "outline"
              : connectionSucceeded(connection)
                ? "default"
                : "destructive"}>{connectionStatusLabel(connection)}</Badge
          >
          <Button variant="secondary" size="sm" disabled={busy} onclick={onTestConnection}>
            {#if busy}<LoaderCircleIcon data-icon="inline-start" class="animate-spin" />{/if}
            Test
          </Button>
        </div>
      {/snippet}
    </SettingRow>

    <SettingRow
      title="Allow insecure HTTP"
      description="Useful for a model server running on this PC or another trusted machine on your private network."
    >
      {#snippet control()}
        <Switch
          id="post-processing-allow-insecure-http"
          bind:checked={settings.postProcessing.allowInsecureHTTP}
          aria-label="Allow insecure HTTP for post-processing"
        />
      {/snippet}
    </SettingRow>

    {#if settings.postProcessing.allowInsecureHTTP}
      <Alert.Root variant="destructive" class="rounded-none border-x-0 border-b-0 px-5">
        <TriangleAlertIcon />
        <Alert.Title>Transcript text may be readable on the network</Alert.Title>
        <Alert.Description>
          Use <code class="font-mono">http://</code> only for a local or trusted private endpoint. API
          credentials and transcript text are otherwise sent without TLS.
        </Alert.Description>
      </Alert.Root>
    {/if}

    <ValueRow
      id="post-processing-model"
      label="Model"
      hint="Discovered through /v1/models metadata only."
    >
      {#snippet control()}
        {#if canSelectModel}
          <Select.Root type="single" bind:value={settings.postProcessing.model}>
            <Select.Trigger id="post-processing-model" class="w-full">
              {settings.postProcessing.model || "Choose a discovered model"}
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
            id="post-processing-model"
            bind:value={settings.postProcessing.model}
            placeholder="Test the endpoint to discover models"
            spellcheck={false}
          />
        {/if}
      {/snippet}
      {#snippet action()}<Badge variant="outline">{modelPresence}</Badge>{/snippet}
    </ValueRow>

    <ValueRow
      id="post-processing-api-key"
      label="API key"
      hint="Stored in Windows Credential Manager and never returned to this window."
    >
      {#snippet control()}
        <ValueInput
          id="post-processing-api-key"
          type="password"
          autocomplete="new-password"
          maxlength={2048}
          bind:value={apiKey}
          mono={false}
          placeholder={settings.postProcessingCredentialConfigured
            ? "Stored in Windows Credential Manager"
            : "Optional for local servers"}
        />
      {/snippet}
      {#snippet action()}
        <Badge
          variant={settings.postProcessingCredentialConfigured || apiKey.trim()
            ? "secondary"
            : "outline"}>{credentialStatus}</Badge
        >
      {/snippet}
    </ValueRow>

    <ValueRow
      id="post-processing-timeout"
      label="Request timeout"
      hint="If cleanup exceeds this time, Freehand preserves and delivers the raw transcript instead."
    >
      {#snippet control()}
        <ValueInput
          id="post-processing-timeout"
          type="number"
          min={10}
          max={3600}
          step={10}
          bind:value={settings.postProcessing.timeoutSeconds}
        />
      {/snippet}
      {#snippet action()}<Badge variant="outline">seconds</Badge>{/snippet}
    </ValueRow>

    <SettingRow
      title="Clear stored API key"
      description="Remove only the post-processing credential when these settings are saved."
    >
      {#snippet control()}
        <Switch
          id="clear-post-processing-api-key"
          bind:checked={clearKey}
          aria-label="Clear stored post-processing API key"
        />
      {/snippet}
    </SettingRow>
  </SettingsCard>

  <SettingsCard>
    <div class="px-5 py-5">
      <ProcessingProfilePicker {profiles} bind:value={settings.postProcessing.preset} />
    </div>

    <div class="px-5 py-5">
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
    </div>
  </SettingsCard>

  <CleanupControls
    bind:options={settings.postProcessing.generationOptions}
    capabilities={compatibility?.capabilities}
    s1Mini={processor.preset === PostProcessingPreset.PostProcessingPresetS1Mini}
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
