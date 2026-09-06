<script lang="ts">
  import ModelProfilePicker from "$lib/components/settings/ModelProfilePicker.svelte";
  import { ID } from "$bindings/compatibility";
  import RuntimeModelPicker from "$lib/components/settings/RuntimeModelPicker.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import { Switch } from "$lib/components/ui/switch";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import * as Slider from "$lib/components/ui/slider";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import SquareIcon from "@lucide/svelte/icons/square";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import { TTSPhase, type Settings, type ConnectionResult, type TTSStatus } from "$lib/state";
  import { connectionDescription } from "$lib/utils/connection";
  let {
    settings = $bindable(),
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
  const compatibility = $derived(
    settings.modelProfiles.speech?.find((p) => p.id === (speech.modelProfile || ID.Generic)),
  );
  $effect(() => {
    if (settings.textToSpeech.enabled && settings.textToSpeech.speed === 0)
      settings.textToSpeech.speed = 1;
  });
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <SettingRow
      title="Enable speech playback"
      description="Add on-demand Listen controls to completed transcripts. Nothing is read automatically."
      >{#snippet control()}<Switch
          bind:checked={settings.textToSpeech.enabled}
          aria-label="Enable speech playback"
        />{/snippet}</SettingRow
    >
    <RuntimeModelPicker
      id="tts-model"
      bind:value={settings.textToSpeech.model}
      models={connection?.modelIDs ?? []}
      busy={connectionBusy}
      onDiscover={onTestConnection}
    />
    <ModelProfilePicker
      id="speech-model-profile"
      value={speech.modelProfile}
      profiles={settings.modelProfiles.speech ?? []}
      onChange={(id) => (settings.textToSpeech.modelProfile = id)}
    />
    <ValueRow
      id="tts-voice"
      label="Voice"
      hint="A provider voice ID. The compatible API does not define voice discovery."
    >
      {#snippet control()}<ValueInput
          id="tts-voice"
          bind:value={settings.textToSpeech.voice}
          placeholder="af_heart"
          spellcheck={false}
        />{/snippet}
    </ValueRow>

    <ValueRow
      id="tts-format"
      label="Audio format"
      hint="Freehand requests uncompressed audio for deterministic native Windows playback."
    >
      {#snippet control()}<Badge
          id="tts-format"
          variant="outline"
          class="justify-self-start font-mono">WAV · PCM16</Badge
        >{/snippet}
    </ValueRow>

    <ValueRow
      id="tts-speed"
      label="Speaking speed"
      hint="Requests 0.25× through 4×; support and effect depend on the server and model."
    >
      {#snippet control()}
        <div class="flex items-center gap-3">
          <Slider.Root
            disabled={!compatibility?.capabilities.speechSpeed}
            id="tts-speed"
            type="single"
            min={0.25}
            max={4}
            step={0.05}
            value={settings.textToSpeech.speed}
            onValueChange={(value) => (settings.textToSpeech.speed = value)}
            aria-label="Speech playback speed"
          />
          <Badge variant="outline" class="min-w-14 justify-center font-mono"
            >{speech.speed.toFixed(2)}×</Badge
          >
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
  </SettingsCard>
  {#if connection}<p role="status" class="text-xs text-muted-foreground">
      {connectionDescription(connection)}
    </p>{/if}
  <SettingsCard>
    <div class="flex items-center justify-between gap-4 px-5 py-4">
      <div class="min-w-0">
        <p class="text-sm font-medium">Voice preview</p>
        <p class="mt-1 text-xs leading-relaxed text-muted-foreground">
          Save these settings, then explicitly synthesize one short phrase to verify the complete
          endpoint and native playback path.
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
    WAV audio is kept only in memory for the active playback session. Save writes a user-selected
    WAV directly; Clear, replacement, recording, or app shutdown releases the retained audio.
    Connection checks stop after 15 seconds. Speech input is limited to 4,096 characters and
    generated WAV audio to 32 MiB.
  </p>
</div>
