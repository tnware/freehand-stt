<script lang="ts">
  import type { Settings } from "$lib/state";
  import { Purpose } from "$bindings/savedconnection";
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";

  let {
    settings,
    purpose,
    runtime,
    draft = false,
  }: {
    settings: Settings;
    purpose: Purpose;
    runtime?: ManagedRuntimeState;
    draft?: boolean;
  } = $props();
  const config = $derived(
    purpose === Purpose.Voice
      ? settings.voiceTranscription
      : purpose === Purpose.Cleanup
        ? settings.postProcessing
        : purpose === Purpose.Speech
          ? settings.textToSpeech
          : settings,
  );
  const connection = $derived(
    settings.savedConnections.entries?.find(
      (entry) => entry.id === settings.savedConnections.selected?.[purpose],
    ),
  );
  const instance = $derived(runtime?.statusFor(config.managedInstanceID ?? ""));
  const managed = $derived(!!config.managedInstanceID);
  const profiles = $derived(
    purpose === Purpose.Cleanup
      ? settings.compatibilityProfiles.postProcessing
      : purpose === Purpose.Speech
        ? settings.compatibilityProfiles.speech
        : settings.compatibilityProfiles.transcription,
  );
  const backend = $derived(
    profiles?.find((profile) => profile.id === config.compatibilityProfile),
  );
  const modelID = $derived(
    managed
      ? ((purpose === Purpose.Speech
          ? instance?.instance.speechModel
          : instance?.instance.model) ?? "")
      : config.model,
  );
  const modelName = $derived(
    managed
      ? instance?.status.models?.find((model) => model.id === modelID)?.name ||
          modelID
      : modelID,
  );
  const serverLoaded = $derived(
    !managed &&
      !!backend?.capabilities.serverLoadedModel &&
      !(purpose === Purpose.Voice && backend?.capabilities.realtime),
  );
  const inactive = $derived(
    (purpose === Purpose.Cleanup && !settings.postProcessing.enabled) ||
      (purpose === Purpose.Speech && !settings.textToSpeech.enabled),
  );
  const engine = $derived(
    managed
      ? runtime?.providers.find(
          (provider) => provider.id === instance?.instance.provider,
        )?.name || "Local runtime"
      : backend?.label || "Compatible server",
  );
</script>

<div
  class="selection-summary"
  role="group"
  aria-label={draft || inactive
    ? "Selected engine and model"
    : "Active engine and model"}
>
  <div class="flex min-w-0 items-center gap-2">
    <ProviderIcon
      profile={instance?.instance.provider || config.compatibilityProfile}
      size={16}
    />
    <p class="min-w-0 flex-1 text-xs font-medium text-secondary-foreground">
      {draft || inactive ? "Selected engine" : "Active engine"}
    </p>
    <span class="text-[10px] text-muted-foreground"
      >{managed ? "Local" : "Connection"}</span
    >
  </div>
  <p class="break-words text-[13px] font-semibold">
    {connection ? engine : "Choose a connection"}
  </p>
  <p class="break-all font-mono text-xs text-muted-foreground">
    {connection
      ? serverLoaded
        ? "Model loaded on server"
        : modelName || "Choose a model"
      : "Then choose the model for this workflow."}
  </p>
  {#if inactive}<p class="text-xs text-muted-foreground">
      This workflow is off. Enable it to use this selection.
    </p>{/if}
  {#if managed && instance && instance.status.state !== "running"}<p
      class="text-xs text-muted-foreground"
    >
      Selected locally · runtime not running
    </p>{/if}
</div>

<style>
  .selection-summary {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 0.25rem;
    padding: 0.625rem 0.75rem;
    border: 1px solid var(--hairline);
    border-radius: var(--radius-md);
    background: var(--well);
  }
</style>
