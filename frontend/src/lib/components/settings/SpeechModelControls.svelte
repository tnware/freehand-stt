<script lang="ts">
  import type { ManagedRuntimeState } from "$lib/stores/managed-runtime.svelte";
  import ManagedRuntimeControls from "$lib/components/home/ManagedRuntimeControls.svelte";
  import LanguagePicker from "./LanguagePicker.svelte";
  import { ID } from "$bindings/modelprofile";
  import { Role } from "$bindings/compatibility";
  import {
    speechLanguageOptions,
    speechLanguageValue,
  } from "$lib/utils/speechLanguages";
  import type { Snippet } from "svelte";
  import type { Settings } from "$lib/state";
  import type { VoicesResult } from "$bindings/inference";
  import { Purpose } from "$bindings/savedconnection";
  import { rememberedModels } from "$lib/utils/modelSettings";
  import RuntimeModelPicker from "./RuntimeModelPicker.svelte";
  import VoicePicker from "./VoicePicker.svelte";
  import SpeechSpeedControl from "./SpeechSpeedControl.svelte";

  let {
    settings,
    runtime,
    runtimeWorkBusy = false,
    onManageRuntime = () => {},
    models = [],
    draftModels = [],
    voices = null,
    busy = false,
    modelsBusy = false,
    voicesBusy = false,
    compact = false,
    immediate = false,
    showAdvanced = true,
    onEnter,
    metadataStatus = "idle",
    onChooseModel,
    onForgetModel,
    onDiscoverModels,
    onDiscoverVoices,
    onVoice,
    onSpeed,
    onOptions,
    modelDetails,
    voiceActions,
  }: {
    settings: Settings;
    runtime?: ManagedRuntimeState;
    runtimeWorkBusy?: boolean;
    onManageRuntime?: () => void;
    models?: string[];
    draftModels?: string[];
    onEnter?: () => void;
    metadataStatus?: "idle" | "loading" | "ready" | "empty" | "failed";
    voices?: VoicesResult | null;
    busy?: boolean;
    modelsBusy?: boolean;
    voicesBusy?: boolean;
    compact?: boolean;
    immediate?: boolean;
    showAdvanced?: boolean;
    onChooseModel: (model: string) => boolean | Promise<boolean>;
    onForgetModel?: () => void;
    onDiscoverModels: () => void;
    onDiscoverVoices: () => void;
    onVoice: (voice: string) => boolean | Promise<boolean>;
    onSpeed: (speed: number) => boolean | Promise<boolean>;
    onOptions: (
      options: Settings["textToSpeech"]["options"],
    ) => boolean | Promise<boolean>;
    modelDetails?: Snippet;
    voiceActions?: Snippet;
  } = $props();
  const uid = $props.id();
  const controlID = (name: string) =>
    compact ? `${uid}-speech-${name}` : `tts-${name}`;
  const speech = $derived(settings.textToSpeech);
  const profile = $derived(
    settings.modelProfiles.speech?.find(
      (p) => p.id === (speech.modelProfile || ID.Generic),
    ),
  );
  const magpie = $derived(profile?.id === ID.MagpieTTS);
  const languages = $derived(speechLanguageOptions(profile, voices));
  const allowedVoices = $derived(
    magpie
      ? [
          "default",
          ...(!voices?.errorKind && voices?.voices?.length
            ? voices.voices.map((voice) => voice.id)
            : (profile?.voices ?? [])),
        ].filter((voice, index, all) => all.indexOf(voice) === index)
      : (profile?.voices ?? []),
  );
</script>

{#if speech.managedInstanceID}
  {#if runtime}
    <ManagedRuntimeControls
      sidebar={compact && !showAdvanced}
      workBusy={runtimeWorkBusy}
      {runtime}
      role={Role.Speech}
      instanceID={speech.managedInstanceID}
      disabled={busy}
      onManage={onManageRuntime}
    />
  {/if}
{:else}
  <RuntimeModelPicker
    sidebar={compact && !showAdvanced}
    showProfileName={!modelDetails}
    id={controlID("model")}
    value={speech.model}
    profileName={profile?.name ?? speech.modelProfile}
    {models}
    {draftModels}
    {compact}
    {immediate}
    savedModels={rememberedModels(settings, Purpose.Speech).map((e) => e.model)}
    disabled={busy}
    busy={modelsBusy}
    onEnter={() => {
      if (!busy) onEnter?.();
    }}
    {metadataStatus}
    onChoose={(model) => (busy ? false : onChooseModel(model))}
    onForget={onForgetModel
      ? () => {
          if (!busy) onForgetModel?.();
        }
      : undefined}
    onDiscover={() => {
      if (!busy && !modelsBusy) onDiscoverModels();
    }}
  />
  {@render modelDetails?.()}
{/if}
<VoicePicker
  id={controlID("voice")}
  value={magpie && (!speech.voice || speech.voice === "alloy")
    ? "default"
    : speech.voice}
  supported={!!profile?.capabilities.voiceDiscovery}
  result={voices}
  {allowedVoices}
  busy={voicesBusy}
  disabled={busy}
  {compact}
  actions={voiceActions}
  onChoose={(voice) => (busy ? false : onVoice(voice))}
  onDiscover={() => {
    if (!busy && !voicesBusy) onDiscoverVoices();
  }}
/>
{#if magpie}<p class="text-xs leading-relaxed text-muted-foreground">
    Default uses the server's default voice. The listed voices share the same
    supported languages.
  </p>{/if}
<SpeechSpeedControl
  id={controlID("speed")}
  value={speech.speed}
  supported={!!profile?.capabilities.speechSpeed}
  disabled={busy}
  {compact}
  onChange={(speed) => (busy ? false : onSpeed(speed))}
/>

{#if showAdvanced && (profile?.capabilities.speechLanguage || profile?.capabilities.speechInstructions)}
  <div class={compact ? "space-y-4" : "space-y-4 p-5"}>
    {#if profile.capabilities.speechLanguage}
      <div class="space-y-1.5">
        <label for={controlID("language")} class="content-value"
          >Speech language</label
        >
        <LanguagePicker
          id={controlID("language")}
          {languages}
          restricted
          disabled={busy}
          bind:value={
            () => speechLanguageValue(profile, speech.options.language),
            (language) => {
              if (!busy) void onOptions({ ...speech.options, language });
            }
          }
        />
        {#if magpie && !voices?.languages?.length}<p
            class="text-xs leading-relaxed text-muted-foreground"
          >
            Japanese and Mandarin require optional server language support.
            Refresh voices to discover enabled languages.
          </p>{/if}
      </div>
    {/if}
    {#if profile.capabilities.speechInstructions}
      <div class="space-y-1.5">
        <label for={controlID("instructions")} class="content-value"
          >Voice style</label
        >
        <textarea
          id={controlID("instructions")}
          rows="3"
          maxlength="500"
          class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          placeholder="Speak warmly and clearly, with a relaxed pace."
          disabled={busy}
          value={speech.options.instructions}
          oninput={(event) => {
            if (!busy && !immediate)
              void onOptions({
                ...speech.options,
                instructions: event.currentTarget.value,
              });
          }}
          onchange={(event) => {
            if (!busy && immediate)
              void onOptions({
                ...speech.options,
                instructions: event.currentTarget.value,
              });
          }}></textarea>
        <p class="text-xs leading-relaxed text-muted-foreground">
          Describe tone, emotion, or delivery. Leave empty for the selected
          voice’s usual style.
        </p>
      </div>
    {/if}
  </div>
{/if}
