<script lang="ts">
  import type { Capabilities, TranscriptionOptions } from "$bindings/compatibility";
  import { Button } from "$lib/components/ui/button";
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
  const hotwordBytes = $derived(new TextEncoder().encode(options.hotwords).length);
  const unsupportedHotwords = $derived(
    Boolean(options.hotwords) && !capabilities?.transcriptionHotwords,
  );
</script>

<details class="rounded-xl border border-card-stroke bg-card shadow-lift">
  <summary
    class="cursor-pointer rounded-xl px-5 py-4 text-sm font-medium focus-visible:outline-2 focus-visible:outline-ring"
  >
    Transcription controls
  </summary>
  <div class="border-t border-hairline px-5 py-4">
    <p class="text-xs leading-relaxed text-muted-foreground">
      Optional hints for recordings, checkpoints, and audio files. Model support varies. Context and
      hotwords are saved in your local settings and sent with each transcription.
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
    <div class="mt-4 space-y-2">
      <label for="transcription-hotwords" class="text-sm font-medium">Hotwords</label>
      <Textarea
        id="transcription-hotwords"
        bind:value={options.hotwords}
        rows={2}
        disabled={!capabilities?.transcriptionHotwords}
        maxlength={2048}
        aria-describedby="transcription-hotwords-help"
        aria-invalid={hotwordBytes > 2048 || unsupportedHotwords}
        placeholder="Freehand, Speaches, project names"
      />
      <p id="transcription-hotwords-help" class="text-xs text-muted-foreground">
        {capabilities?.transcriptionHotwords
          ? "Optional terms to favor during recognition; not guaranteed replacements."
          : "Available with the Speaches profile."}
        {hotwordBytes.toLocaleString()} / 2,048 UTF-8 bytes.
      </p>
      {#if unsupportedHotwords}
        <p role="alert" class="text-xs text-destructive">
          Clear hotwords before saving this profile, or switch back to Speaches.
        </p>
        <Button
          variant="secondary"
          size="sm"
          onclick={() => {
            options.hotwords = "";
          }}>Clear hotwords</Button
        >
      {/if}
      {#if hotwordBytes > 2048}<p role="alert" class="text-xs text-destructive">
          Shorten the hotwords before saving.
        </p>{/if}
    </div>
  </div>
  <div class="border-t border-hairline">
    <SettingRow
      title="Override temperature"
      description="Off uses the server default. Turn on to send an explicit value, including zero."
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
</details>
