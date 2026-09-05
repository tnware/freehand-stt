<script lang="ts">
  import type { Capabilities, CleanupOptions } from "$bindings/compatibility";
  import { Badge } from "$lib/components/ui/badge";
  import { Switch } from "$lib/components/ui/switch";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import ValueRow from "$lib/components/settings/ValueRow.svelte";
  import ValueInput from "$lib/components/settings/ValueInput.svelte";

  let {
    options = $bindable(),
    capabilities,
    s1Mini,
  }: {
    options: CleanupOptions;
    capabilities?: Capabilities;
    s1Mini: boolean;
  } = $props();

  const invalidLimit = $derived(
    !Number.isInteger(options.maxOutputTokens) ||
      options.maxOutputTokens < 1 ||
      options.maxOutputTokens > 65536,
  );
  const reasoningOffRequired = $derived(s1Mini && Boolean(capabilities?.cleanupDisableReasoning));
  const unsupportedReasoning = $derived(
    options.disableReasoning && !capabilities?.cleanupDisableReasoning,
  );

  function toggleLimit(enabled: boolean) {
    if (enabled && invalidLimit) options.maxOutputTokens = 2048;
    if (!enabled && invalidLimit) options.maxOutputTokens = 0;
    options.limitOutputTokens = enabled;
  }
  function updateLimit(event: Event) {
    const value = (event.currentTarget as HTMLInputElement).valueAsNumber;
    options.maxOutputTokens = Number.isFinite(value) ? value : -1;
  }
</script>

<details class="rounded-xl border border-card-stroke bg-card shadow-lift">
  <summary
    class="cursor-pointer rounded-xl px-5 py-4 text-sm font-medium focus-visible:outline-2 focus-visible:outline-ring"
    >Generation controls</summary
  >
  <div class="border-t border-hairline">
    <SettingRow
      title="Limit output tokens"
      description="Off uses the server's output limit. A limit that is too small can leave cleanup incomplete; Freehand then uses the raw transcript."
    >
      {#snippet control()}
        <Switch
          id="cleanup-limit-output"
          checked={options.limitOutputTokens}
          onCheckedChange={toggleLimit}
          disabled={!capabilities?.cleanupOutputLimit && !options.limitOutputTokens}
          aria-label="Limit cleanup output tokens"
        />
      {/snippet}
    </SettingRow>
    {#if options.limitOutputTokens}
      <ValueRow
        id="cleanup-output-tokens"
        label="Maximum output tokens"
        hint="1–65,536 tokens, not words. The model may have a smaller limit. This does not split long transcripts or enlarge its context window."
      >
        {#snippet control()}
          <ValueInput
            id="cleanup-output-tokens"
            type="number"
            min={1}
            max={65536}
            step={1}
            value={options.maxOutputTokens === -1 ? "" : options.maxOutputTokens}
            oninput={updateLimit}
            aria-invalid={invalidLimit}
            aria-describedby="cleanup-output-help"
          />
        {/snippet}
      </ValueRow>
      <p id="cleanup-output-help" class="px-5 pb-4 text-xs text-muted-foreground">
        Choose enough tokens for the entire cleaned transcript. The timeout remains a separate
        limit.
      </p>
      {#if invalidLimit}<p role="alert" class="px-5 pb-4 text-xs text-destructive">
          Enter a whole number from 1 to 65,536, or turn off the output limit.
        </p>{/if}
    {/if}
  </div>
  <div class="border-t border-hairline">
    <SettingRow
      title="Disable reasoning"
      description={s1Mini
        ? capabilities?.cleanupDisableReasoning
          ? "S1-mini requires reasoning to be off. Freehand sends the reasoning-off override on every cleanup request."
          : "S1-mini requires reasoning to be off. Configure the server with thinking disabled. Choose llama.cpp for Freehand's qualified request override."
        : capabilities?.cleanupDisableReasoning
          ? "Request thinking-disabled cleanup. Requires a compatible llama.cpp build and model template. Off preserves the server's behavior."
          : "This override is available with the llama.cpp profile. Other profiles use their server's reasoning configuration."}
    >
      {#snippet control()}
        <div class="flex items-center gap-2">
          {#if s1Mini}<Badge variant="outline"
              >{reasoningOffRequired ? "Required off" : "Disable on server"}</Badge
            >{/if}
          {#if !s1Mini || reasoningOffRequired || options.disableReasoning}
            <Switch
              id="cleanup-disable-reasoning"
              checked={reasoningOffRequired || options.disableReasoning}
              onCheckedChange={(checked) => {
                options.disableReasoning = checked;
              }}
              disabled={reasoningOffRequired ||
                (!capabilities?.cleanupDisableReasoning && !options.disableReasoning)}
              aria-label="Disable cleanup reasoning"
            />
          {/if}
        </div>
      {/snippet}
    </SettingRow>
    {#if unsupportedReasoning}<p role="alert" class="px-5 pb-4 text-xs text-destructive">
        Turn off the reasoning override before saving this profile, or switch back to llama.cpp.
      </p>{/if}
  </div>
  <p class="border-t border-hairline px-5 py-4 text-xs leading-relaxed text-muted-foreground">
    Applies to cleanup after microphone and file transcription. Save before starting a new job.
    S1-mini keeps its fixed prompt and temperature zero. Its requirement to keep reasoning off is
    separate from the optional setting for custom cleanup.
  </p>
</details>
