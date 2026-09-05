<script lang="ts">
  import TranscriptionControls from "$lib/components/settings/TranscriptionControls.svelte";
  import { ID } from "$bindings/compatibility";
  import CompatibilityProfilePicker from "$lib/components/settings/CompatibilityProfilePicker.svelte";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import * as Alert from "$lib/components/ui/alert";
  import * as Select from "$lib/components/ui/select";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import { Switch } from "$lib/components/ui/switch";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import {
    AuthenticationMode,
    ConnectionProbe,
    ModelPresence,
    type ConnectionResult,
    type Settings,
  } from "$lib/state";
  import {
    connectionDescription,
    connectionProbeLabel,
    connectionStatusLabel,
    connectionSucceeded,
  } from "$lib/utils/connection";

  let {
    settings = $bindable(),
    apiKey = $bindable(),
    clearKey = $bindable(),
    connection,
    busy = false,
    onTestConnection,
  }: {
    settings: Settings;
    apiKey: string;
    clearKey: boolean;
    connection: ConnectionResult | null;
    busy?: boolean;
    onTestConnection: () => void;
  } = $props();

  // The health path is optional configuration; revealing its field only when
  // it is wanted keeps the common case to three values.
  let customHealthPath = $state((settings.healthPath ?? "") !== "");
  const compatibility = $derived(
    settings.compatibilityProfiles.transcription?.find(
      (profile) => profile.id === (settings.compatibilityProfile || ID.Generic),
    ),
  );
  const discoveredModels = $derived(connection?.modelIDs ?? []);
  const canSelectModel = $derived(discoveredModels.length > 0);
  const modelPresence = $derived.by(() => {
    if (!connection || connection.probe !== ConnectionProbe.ConnectionProbeModels) return "Manual";
    if (discoveredModels.includes(settings.model)) return "Listed";
    if (connection.modelPresence === ModelPresence.ModelPresenceUnavailable) return "Unavailable";
    return "Not listed";
  });
  const credentialStatus = $derived(
    apiKey.trim()
      ? "Draft entered"
      : settings.credentialConfigured
        ? "Configured"
        : "Not configured",
  );
  const noAuthentication = $derived(
    settings.authenticationMode === AuthenticationMode.AuthenticationModeNone,
  );
  const authenticationLabel = $derived(noAuthentication ? "None" : "API key");
  const connectionHint = $derived.by(() => {
    if (!connection) {
      return "Test the values shown here using only health or model-list metadata.";
    }
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

  function toggleHealthPath(on: boolean) {
    customHealthPath = on;
    if (!on) settings.healthPath = "";
  }

  function chooseAuthenticationMode(value: string) {
    settings.authenticationMode = value as AuthenticationMode;
    if (settings.authenticationMode === AuthenticationMode.AuthenticationModeNone) apiKey = "";
  }

  function updateFileTimeoutMinutes(event: Event) {
    const minutes = (event.currentTarget as HTMLInputElement).valueAsNumber;
    if (Number.isFinite(minutes)) {
      settings.fileTranscriptionTimeoutSeconds = Math.round(minutes * 60);
    }
  }
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <CompatibilityProfilePicker
      id="transcription-compatibility-profile"
      bind:value={settings.compatibilityProfile}
      profiles={settings.compatibilityProfiles.transcription ?? []}
    />
    <ValueRow id="base-url" label="Base URL" hint="OpenAI-compatible /v1 audio endpoint.">
      {#snippet control()}
        <ValueInput id="base-url" type="url" bind:value={settings.baseURL} spellcheck={false} />
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
                : "destructive"}
          >
            {connectionStatusLabel(connection)}
          </Badge>
          <Button variant="secondary" size="sm" disabled={busy} onclick={onTestConnection}>
            {#if busy}<LoaderCircleIcon data-icon="inline-start" class="animate-spin" />{/if}
            Test
          </Button>
        </div>
      {/snippet}
    </SettingRow>

    <ValueRow
      id="authentication-mode"
      label="Authentication"
      hint="Choose explicitly whether this speech-to-text endpoint expects a Bearer API key."
    >
      {#snippet control()}
        <Select.Root
          type="single"
          value={settings.authenticationMode}
          onValueChange={chooseAuthenticationMode}
        >
          <Select.Trigger id="authentication-mode" class="w-full">
            {authenticationLabel}
          </Select.Trigger>
          <Select.Content>
            <Select.Item value={AuthenticationMode.AuthenticationModeAPIKey}>API key</Select.Item>
            <Select.Item value={AuthenticationMode.AuthenticationModeNone}>None</Select.Item>
          </Select.Content>
        </Select.Root>
      {/snippet}
    </ValueRow>

    {#if noAuthentication}
      <Alert.Root class="rounded-none border-x-0 border-b-0 px-5">
        <TriangleAlertIcon />
        <Alert.Title>Authorization will be omitted</Alert.Title>
        <Alert.Description>
          Use this only for a trusted local endpoint. This does not enable insecure HTTP, and any
          stored API key is preserved without being sent.
        </Alert.Description>
      </Alert.Root>
    {/if}

    <SettingRow
      title="Allow insecure HTTP"
      description="Explicitly permit a server connection without transport encryption. Keep this off unless you trust the network and server."
    >
      {#snippet control()}
        <Switch
          id="allow-insecure-http"
          bind:checked={settings.allowInsecureHTTP}
          aria-label="Allow insecure HTTP"
        />
      {/snippet}
    </SettingRow>

    {#if settings.allowInsecureHTTP}
      <Alert.Root variant="destructive" class="rounded-none border-x-0 border-b-0 px-5">
        <TriangleAlertIcon />
        <Alert.Title>Traffic may be readable on the network</Alert.Title>
        <Alert.Description>
          {noAuthentication ? "Uploaded or recorded audio is" : "API credentials and audio are"}
          sent without TLS when the Base URL uses <code class="font-mono">http://</code>.
        </Alert.Description>
      </Alert.Root>
    {/if}

    <ValueRow id="model" label="Model" hint="Discovered through /v1/models metadata only.">
      {#snippet control()}
        {#if canSelectModel}
          <Select.Root type="single" bind:value={settings.model}>
            <Select.Trigger id="model" class="w-full">{settings.model}</Select.Trigger>
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
          <ValueInput id="model" bind:value={settings.model} spellcheck={false} />
        {/if}
      {/snippet}
      {#snippet action()}
        <Badge variant="outline">{modelPresence}</Badge>
      {/snippet}
    </ValueRow>

    <ValueRow
      id="language"
      label="Language"
      hint="Optional. Sent with each request; whether it changes anything depends on the backend."
    >
      {#snippet control()}
        <ValueInput
          id="language"
          disabled={!compatibility?.capabilities.languageHint}
          bind:value={settings.language}
          placeholder="Auto"
          spellcheck={false}
        />
      {/snippet}
    </ValueRow>

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

    {#if noAuthentication}
      <ValueRow
        id="api-key-preserved"
        label="Stored API key"
        hint="Kept in Windows Credential Manager for API-key mode and never returned to this window."
      >
        {#snippet control()}
          <span id="api-key-preserved" class="text-[13px] text-muted-foreground">
            {settings.credentialConfigured ? "Preserved but not sent" : "No credential stored"}
          </span>
        {/snippet}
      </ValueRow>
    {:else}
      <ValueRow
        id="api-key"
        label="API key"
        hint="Stored in Windows Credential Manager and never returned to this window."
      >
        {#snippet control()}
          <ValueInput
            id="api-key"
            type="password"
            autocomplete="new-password"
            maxlength={2048}
            bind:value={apiKey}
            mono={false}
            placeholder={settings.credentialConfigured
              ? "Stored in Windows Credential Manager"
              : "Enter API key"}
          />
        {/snippet}
        {#snippet action()}
          <Badge variant={settings.credentialConfigured || apiKey.trim() ? "secondary" : "outline"}>
            {credentialStatus}
          </Badge>
        {/snippet}
      </ValueRow>
    {/if}

    {#if settings.credentialConfigured || clearKey}
      <SettingRow
        title="Clear stored API key"
        description="Remove the saved credential when these settings are saved."
      >
        {#snippet control()}
          <Switch id="clear-api-key" bind:checked={clearKey} aria-label="Clear stored API key" />
        {/snippet}
      </SettingRow>
    {/if}

    <SettingRow
      title="Custom health path"
      description="Append a health path to the base URL instead of requesting the model list."
    >
      {#snippet control()}
        <Switch
          id="custom-health-path"
          checked={customHealthPath}
          onCheckedChange={toggleHealthPath}
          aria-label="Custom health path"
        />
      {/snippet}
    </SettingRow>

    {#if customHealthPath}
      <ValueRow
        id="health-path"
        label="Health path"
        hint="Include the leading slash. A base URL ending in /v1 with /health requests /v1/health."
      >
        {#snippet control()}
          <ValueInput
            id="health-path"
            bind:value={settings.healthPath}
            placeholder="/health"
            spellcheck={false}
          />
        {/snippet}
      </ValueRow>
    {/if}
  </SettingsCard>

  <TranscriptionControls
    bind:options={settings.transcriptionOptions}
    capabilities={compatibility?.capabilities}
  />

  <p class="px-1 text-xs leading-relaxed text-muted-foreground">
    The check reads only <code class="font-mono">/health</code> or
    <code class="font-mono">/models</code> metadata, never invokes a model, and stops after 15 seconds.
    Safety ceilings are fixed at 8 MiB per microphone request, 2 GiB per stored audio file, 1 MiB per
    completed response, and 8 MiB per stored-file transcript.
  </p>
</div>
