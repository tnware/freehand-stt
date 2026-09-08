<script lang="ts">
  import { ID } from "$bindings/modelprofile";
  import SpeechModelControls from "../SpeechModelControls.svelte";
  import type { VoicesResult } from "$bindings/inference";
  import ModelProfilePicker from "$lib/components/settings/ModelProfilePicker.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Switch } from "$lib/components/ui/switch";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import SquareIcon from "@lucide/svelte/icons/square";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import { TTSPhase, type Settings, type ConnectionResult, type TTSStatus } from "$lib/state";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  let {
    settings = $bindable(),
    status,
    busy = false,
    connection,
    connectionBusy = false,
    canPreview = true,
    draftModels = [],
    onChooseModel,
    onForgetModel,
    connectionStale = false,
    onTestConnection,
    voices = null,
    voicesBusy = false,
    onDiscoverVoices,
    onPreview,
    onStop,
    onSave,
    onClear,
  }: {
    settings: Settings;
    status: TTSStatus;
    busy?: boolean;
    connection: ConnectionResult | null;
    connectionBusy?: boolean;
    canPreview?: boolean;
    draftModels?: string[];
    onChooseModel: (model: string) => boolean;
    onForgetModel: () => void;
    connectionStale?: boolean;
    onTestConnection: () => void;
    voices?: VoicesResult | null;
    voicesBusy?: boolean;
    onDiscoverVoices: () => void;
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
  $effect(() => {
    if (settings.textToSpeech.enabled && settings.textToSpeech.speed === 0)
      settings.textToSpeech.speed = 1;
  });
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <SettingRow
      title="Enable text to speech"
      description="Add on-demand Listen controls to completed transcripts. Nothing is read automatically."
      >{#snippet control()}<Switch
          bind:checked={settings.textToSpeech.enabled}
          aria-label="Enable text to speech"
        />{/snippet}</SettingRow
    >
    <SpeechModelControls
      {settings}
      {draftModels}
      {voices}
      {voicesBusy}
      {busy}
      models={connection?.modelIDs ?? []}
      modelsBusy={connectionBusy}
      {onChooseModel}
      {onForgetModel}
      onDiscoverModels={onTestConnection}
      {onDiscoverVoices}
      onOptions={(options) => {
        settings.textToSpeech.options = options;
        return true;
      }}
      onVoice={(voice) => {
        settings.textToSpeech.voice = voice;
        return true;
      }}
      onSpeed={(speed) => {
        settings.textToSpeech.speed = speed;
        return true;
      }}
    >
      {#snippet modelDetails()}
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
          id="speech-model-profile"
          value={speech.modelProfile}
          profiles={settings.modelProfiles.speech ?? []}
          onChange={(id) => {
            settings.textToSpeech.modelProfile = id;
            settings.textToSpeech.options = { language: "", instructions: "" };
            if (id === ID.Qwen3TTS) settings.textToSpeech.voice = "ryan";
          }}
        />
      {/snippet}
    </SpeechModelControls>

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
  </SettingsCard>

  <SettingsCard>
    <div class="flex items-center justify-between gap-4 px-5 py-4">
      <div class="min-w-0">
        <p class="text-sm font-medium">Voice preview</p>
        <p class="mt-1 text-xs leading-relaxed text-muted-foreground">
          Try a short phrase with your current edits. Preview does not save your settings.
        </p>
      </div>
      <div class="flex items-center gap-2">
        {#if status.canSave}
          <Button variant="outline" size="sm" onclick={onSave}
            ><DownloadIcon data-icon="inline-start" />Save</Button
          >
        {/if}
        {#if active}
          <Button variant="secondary" size="sm" onclick={onStop}
            ><SquareIcon data-icon="inline-start" />Stop</Button
          >
        {:else if status.canClear}
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label="Clear generated speech from memory"
            onclick={onClear}><Trash2Icon /></Button
          >
          <Button
            size="sm"
            disabled={busy || !canPreview || !settings.textToSpeech.enabled}
            onclick={onPreview}><Volume2Icon data-icon="inline-start" />Preview again</Button
          >
        {:else}
          <Button
            size="sm"
            disabled={busy || !canPreview || !settings.textToSpeech.enabled}
            onclick={onPreview}
          >
            {#if busy}<LoaderCircleIcon
                data-icon="inline-start"
                class="animate-spin"
              />{:else}<Volume2Icon data-icon="inline-start" />{/if}
            Preview
          </Button>
        {/if}
      </div>
    </div>
  </SettingsCard>

  <p class="px-1 text-xs leading-relaxed text-muted-foreground">
    Generated audio stays in memory unless you choose Save. Clear, new speech, recording, or
    quitting Freehand releases it.
  </p>
  <details class="px-1 text-xs leading-relaxed text-muted-foreground">
    <summary class="cursor-pointer font-medium text-foreground">Speech request details</summary>
    <p class="mt-3">
      Freehand requests uncompressed WAV · PCM16 audio for native Windows playback. Save writes a
      WAV file to the location you choose. Connection checks stop after 15 seconds. Speech input is
      limited to 4,096 characters and generated WAV audio to 32 MiB.
    </p>
  </details>
</div>
