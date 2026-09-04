<script lang="ts">
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import SquareIcon from "@lucide/svelte/icons/square";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import * as Alert from "$lib/components/ui/alert";
  import * as Select from "$lib/components/ui/select";
  import * as Slider from "$lib/components/ui/slider";
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
    TTSPhase,
    type ConnectionResult,
    type Settings,
    type TTSStatus,
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
    status,
    busy = false,
    connection,
    connectionBusy = false,
    canPreview = true,
    onTestConnection,
    onPreview,
    onStop,
    onSave,
    onClear,
  }: {
    settings: Settings;
    apiKey: string;
    clearKey: boolean;
    status: TTSStatus;
    busy?: boolean;
    connection: ConnectionResult | null;
    connectionBusy?: boolean;
    canPreview?: boolean;
    onTestConnection: () => void;
    onPreview: () => void;
    onStop: () => void;
    onSave: () => void;
    onClear: () => void;
  } = $props();

  const speech = $derived(settings.textToSpeech);
  const active = $derived(
    status.phase === TTSPhase.Generating ||
      status.phase === TTSPhase.Playing ||
      status.phase === TTSPhase.Paused,
  );
  const discoveredModels = $derived(connection?.modelIDs ?? []);
  const canSelectModel = $derived(discoveredModels.length > 0);
  const modelPresence = $derived.by(() => {
    if (!connection || connection.probe !== ConnectionProbe.ConnectionProbeModels) return "Manual";
    if (discoveredModels.includes(speech.model)) return "Listed";
    if (connection.modelPresence === ModelPresence.ModelPresenceUnavailable) return "Unavailable";
    return "Not listed";
  });
  const connectionHint = $derived.by(() => {
    if (!connection) return "Discover models with standard OpenAI-compatible /v1/models metadata.";
    const details = [
      connection.requestedURL,
      connectionProbeLabel(connection),
      `${connection.latencyMilliseconds.toLocaleString()} ms`,
      connection.httpStatus ? `HTTP ${connection.httpStatus}` : "",
    ].filter(Boolean);
    return `${connectionDescription(connection)} ${details.join(" · ")}`;
  });
  $effect(() => {
    if (settings.textToSpeech.enabled && settings.textToSpeech.speed === 0) {
      settings.textToSpeech.speed = 1;
    }
  });
  const credentialStatus = $derived(
    apiKey.trim()
      ? "Draft entered"
      : settings.textToSpeechCredentialConfigured
        ? "Configured"
        : speech.authenticationMode === AuthenticationMode.AuthenticationModeNone
          ? "Not used"
          : "Required",
  );

  function chooseAuthenticationMode(value: string) {
    settings.textToSpeech.authenticationMode = value as AuthenticationMode;
    if (value === AuthenticationMode.AuthenticationModeNone) apiKey = "";
  }
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <SettingRow
      title="Enable speech playback"
      description="Add on-demand Listen controls to completed transcripts. Freehand never reads a transcript automatically."
    >
      {#snippet control()}
        <Switch
          id="tts-enabled"
          bind:checked={settings.textToSpeech.enabled}
          aria-label="Enable speech playback"
        />
      {/snippet}
    </SettingRow>

    <ValueRow id="tts-base-url" label="Base URL" hint="OpenAI-compatible /v1/audio/speech endpoint.">
      {#snippet control()}
        <ValueInput id="tts-base-url" type="url" bind:value={settings.textToSpeech.baseURL} spellcheck={false} placeholder="http://127.0.0.1:8000/v1" />
      {/snippet}
    </ValueRow>

    <SettingRow title="Connection" description={connectionHint}>
      {#snippet control()}
        <div class="flex items-center gap-2">
          <Badge variant={!connection ? "outline" : connectionSucceeded(connection) ? "default" : "destructive"}>
            {connectionStatusLabel(connection)}
          </Badge>
          <Button variant="secondary" size="sm" disabled={connectionBusy} onclick={onTestConnection}>
            {#if connectionBusy}<LoaderCircleIcon data-icon="inline-start" class="animate-spin" />{/if}
            Test
          </Button>
        </div>
      {/snippet}
    </SettingRow>

    <ValueRow id="tts-authentication" label="Authentication" hint="Use a separate credential from transcription and post-processing.">
      {#snippet control()}
        <Select.Root type="single" value={speech.authenticationMode} onValueChange={chooseAuthenticationMode}>
          <Select.Trigger id="tts-authentication" class="w-full">
            {speech.authenticationMode === AuthenticationMode.AuthenticationModeAPIKey ? "API key" : "None"}
          </Select.Trigger>
          <Select.Content>
            <Select.Item value={AuthenticationMode.AuthenticationModeNone}>None</Select.Item>
            <Select.Item value={AuthenticationMode.AuthenticationModeAPIKey}>API key</Select.Item>
          </Select.Content>
        </Select.Root>
      {/snippet}
    </ValueRow>

    <SettingRow title="Allow insecure HTTP" description="Useful for a speech server on this PC or a trusted private network.">
      {#snippet control()}
        <Switch id="tts-insecure" bind:checked={settings.textToSpeech.allowInsecureHTTP} aria-label="Allow insecure HTTP for speech playback" />
      {/snippet}
    </SettingRow>

    {#if settings.textToSpeech.allowInsecureHTTP}
      <Alert.Root variant="destructive" class="rounded-none border-x-0 border-b-0 px-5">
        <TriangleAlertIcon />
        <Alert.Title>Transcript text may be readable on the network</Alert.Title>
        <Alert.Description>Use HTTP only for a local or trusted private endpoint. Speech input and credentials are otherwise sent without TLS.</Alert.Description>
      </Alert.Root>
    {/if}

    <ValueRow id="tts-model" label="Model" hint="The speech model accepted by this endpoint.">
      {#snippet control()}
        {#if canSelectModel}
          <Select.Root type="single" bind:value={settings.textToSpeech.model}>
            <Select.Trigger id="tts-model" class="w-full">{speech.model || "Choose a discovered model"}</Select.Trigger>
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
          <ValueInput id="tts-model" bind:value={settings.textToSpeech.model} placeholder="tts-1" spellcheck={false} />
        {/if}
      {/snippet}
      {#snippet action()}<Badge variant="outline">{modelPresence}</Badge>{/snippet}
    </ValueRow>

    <ValueRow id="tts-voice" label="Voice" hint="A provider voice ID. The compatible API does not define voice discovery.">
      {#snippet control()}<ValueInput id="tts-voice" bind:value={settings.textToSpeech.voice} placeholder="af_heart" spellcheck={false} />{/snippet}
    </ValueRow>

    <ValueRow id="tts-format" label="Audio format" hint="Freehand requests uncompressed audio for deterministic native Windows playback.">
      {#snippet control()}<Badge id="tts-format" variant="outline" class="justify-self-start font-mono">WAV · PCM16</Badge>{/snippet}
    </ValueRow>

    <ValueRow id="tts-speed" label="Speaking speed" hint="OpenAI-compatible endpoints accept 0.25× through 4×.">
      {#snippet control()}
        <div class="flex items-center gap-3">
          <Slider.Root id="tts-speed" type="single" min={0.25} max={4} step={0.05} value={settings.textToSpeech.speed} onValueChange={(value) => (settings.textToSpeech.speed = value)} aria-label="Speech playback speed" />
          <Badge variant="outline" class="min-w-14 justify-center font-mono">{speech.speed.toFixed(2)}×</Badge>
        </div>
      {/snippet}
    </ValueRow>

    <ValueRow
      id="tts-timeout"
      label="Generation timeout"
      hint="Maximum time to wait for the endpoint to produce playable speech."
    >
      {#snippet control()}
        <ValueInput
          id="tts-timeout"
          type="number"
          min={10}
          max={3600}
          step={10}
          bind:value={settings.textToSpeech.timeoutSeconds}
        />
      {/snippet}
      {#snippet action()}<Badge variant="outline">seconds</Badge>{/snippet}
    </ValueRow>

    {#if speech.authenticationMode === AuthenticationMode.AuthenticationModeAPIKey}
      <ValueRow id="tts-api-key" label="API key" hint="Stored separately in Windows Credential Manager and never returned to this window.">
        {#snippet control()}
          <ValueInput id="tts-api-key" type="password" autocomplete="new-password" maxlength={2048} bind:value={apiKey} mono={false} placeholder={settings.textToSpeechCredentialConfigured ? "Stored in Windows Credential Manager" : "Enter an API key"} />
        {/snippet}
        {#snippet action()}<Badge variant={settings.textToSpeechCredentialConfigured || apiKey.trim() ? "secondary" : "outline"}>{credentialStatus}</Badge>{/snippet}
      </ValueRow>

      <SettingRow title="Clear stored API key" description="Remove only the speech playback credential when these settings are saved.">
        {#snippet control()}<Switch id="clear-tts-api-key" bind:checked={clearKey} aria-label="Clear stored speech playback API key" />{/snippet}
      </SettingRow>
    {/if}
  </SettingsCard>

  <SettingsCard>
    <div class="flex items-center justify-between gap-4 px-5 py-4">
      <div class="min-w-0">
        <p class="text-sm font-medium">Voice preview</p>
        <p class="mt-1 text-xs leading-relaxed text-muted-foreground">Save these settings, then explicitly synthesize one short phrase to verify the complete endpoint and native playback path.</p>
      </div>
      <div class="flex items-center gap-2">
        {#if status.canSave}
          <Button variant="outline" size="sm" onclick={onSave}><DownloadIcon data-icon="inline-start" />Save</Button>
        {/if}
        {#if active}
          <Button variant="secondary" size="sm" onclick={onStop}><SquareIcon data-icon="inline-start" />Stop</Button>
        {:else if status.canClear}
          <Button variant="ghost" size="icon-sm" aria-label="Clear generated speech from memory" onclick={onClear}><Trash2Icon /></Button>
          <Button size="sm" disabled={busy || !canPreview || !settings.textToSpeech.enabled} onclick={onPreview}><Volume2Icon data-icon="inline-start" />Preview again</Button>
        {:else}
          <Button size="sm" disabled={busy || !canPreview || !settings.textToSpeech.enabled} onclick={onPreview}>
            {#if busy}<LoaderCircleIcon data-icon="inline-start" class="animate-spin" />{:else}<Volume2Icon data-icon="inline-start" />{/if}
            Preview
          </Button>
        {/if}
      </div>
    </div>
  </SettingsCard>

  <p class="px-1 text-xs leading-relaxed text-muted-foreground">WAV audio is kept only in memory for the active playback session. Save writes a user-selected WAV directly; Clear, replacement, recording, or app shutdown releases the retained audio. Connection checks stop after 15 seconds. Speech input is limited to 4,096 characters and generated WAV audio to 32 MiB.</p>
</div>
