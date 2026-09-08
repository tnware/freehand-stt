<script lang="ts">
  import SettingsDisclosure from "./SettingsDisclosure.svelte";
  import type { Capabilities, TranscriptionOptions } from "$bindings/compatibility";
  import { Switch } from "$lib/components/ui/switch";
  import { Textarea } from "$lib/components/ui/textarea";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";

  let {
    options = $bindable(),
    capabilities,
  }: {
    options: TranscriptionOptions;
    capabilities?: Capabilities;
  } = $props();
  function updateTemperature(event: Event) {
    const value = (event.currentTarget as HTMLInputElement).valueAsNumber;
    options.temperature = Number.isFinite(value) ? value : -1;
  }
  function toggleTemperature(enabled: boolean) {
    options.temperatureOverride = enabled;
    if (
      !enabled &&
      (!Number.isFinite(options.temperature) || options.temperature < 0 || options.temperature > 1)
    ) {
      options.temperature = 0;
    }
  }
  const promptBytes = $derived(new TextEncoder().encode(options.prompt).length);
</script>

<SettingsDisclosure title="Transcription controls" description="Context hints and temperature">
  <div class="border-t border-hairline px-5 py-4">
    <p class="text-xs leading-relaxed text-muted-foreground">
      Give the model context for recognizing speech. Shared terms are managed in Vocabulary.
    </p>
    <div class="mt-4 space-y-2">
      <label for="transcription-prompt" class="text-sm font-medium">Transcription context</label>
      <Textarea
        id="transcription-prompt"
        bind:value={options.prompt}
        rows={3}
        disabled={!capabilities?.transcriptionPrompt}
        maxlength={8192}
        aria-describedby="transcription-prompt-help"
        aria-invalid={promptBytes > 8192}
        placeholder="Names, subject matter, or examples of expected wording"
      />
      <p id="transcription-prompt-help" class="text-xs text-muted-foreground">
        A recognition hint, separate from the cleanup instruction. Leave blank to omit.
        {promptBytes.toLocaleString()} / 8,192 UTF-8 bytes.
      </p>
      {#if promptBytes > 8192}<p role="alert" class="text-xs text-destructive">
          Shorten the context before saving.
        </p>{/if}
    </div>
  </div>
  <div class="border-t border-hairline">
    <SettingRow
      title="Override temperature"
      controlID="transcription-temperature-override"
      compact
      description="Off uses the server default."
    >
      {#snippet control()}
        <Switch
          id="transcription-temperature-override"
          checked={options.temperatureOverride}
          onCheckedChange={toggleTemperature}
          disabled={!capabilities?.transcriptionTemperature}
          aria-label="Override transcription temperature"
        />
      {/snippet}
    </SettingRow>
    {#if options.temperatureOverride}
      <ValueRow
        id="transcription-temperature"
        label="Temperature"
        hint="0–1. Higher values may increase variation; effects depend on the model and decoder."
      >
        {#snippet control()}
          <ValueInput
            id="transcription-temperature"
            type="number"
            min={0}
            max={1}
            step={0.1}
            disabled={!capabilities?.transcriptionTemperature}
            value={options.temperature === -1 ? "" : options.temperature}
            oninput={updateTemperature}
            aria-invalid={options.temperature < 0 || options.temperature > 1}
          />
        {/snippet}
      </ValueRow>
      {#if options.temperature < 0 || options.temperature > 1}
        <p role="alert" class="px-5 pb-4 text-xs text-destructive">
          Enter a temperature from 0 to 1, or turn off the override.
        </p>
      {/if}
    {/if}
  </div>
</SettingsDisclosure>
