<script lang="ts">
  import LanguagePicker from "./LanguagePicker.svelte";
  import { ID } from "$bindings/modelprofile";
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
    models = [],
    draftModels = [],
    voices = null,
    busy = false,
    modelsBusy = false,
    voicesBusy = false,
    compact = false,
    immediate = false,
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
    models?: string[];
    draftModels?: string[];
    voices?: VoicesResult | null;
    busy?: boolean;
    modelsBusy?: boolean;
    voicesBusy?: boolean;
    compact?: boolean;
    immediate?: boolean;
    onChooseModel: (model: string) => boolean | Promise<boolean>;
    onForgetModel?: () => void;
    onDiscoverModels: () => void;
    onDiscoverVoices: () => void;
    onVoice: (voice: string) => boolean | Promise<boolean>;
    onSpeed: (speed: number) => boolean | Promise<boolean>;
    onOptions: (options: Settings["textToSpeech"]["options"]) => boolean | Promise<boolean>;
    modelDetails?: Snippet;
    voiceActions?: Snippet;
  } = $props();
  const speech = $derived(settings.textToSpeech);
  const profile = $derived(
    settings.modelProfiles.speech?.find((p) => p.id === (speech.modelProfile || ID.Generic)),
  );
</script>

<RuntimeModelPicker
  showProfileName={!modelDetails}
  id={compact ? "quick-speech-model" : "tts-model"}
  value={speech.model}
  profileName={profile?.name ?? speech.modelProfile}
  {models}
  {draftModels}
  {compact}
  {immediate}
  savedModels={rememberedModels(settings, Purpose.Speech).map((e) => e.model)}
  disabled={busy}
  busy={modelsBusy}
  onChoose={onChooseModel}
  onForget={onForgetModel}
  onDiscover={onDiscoverModels}
/>
{@render modelDetails?.()}
<VoicePicker
  id={compact ? "quick-speech-voice" : "tts-voice"}
  value={speech.voice}
  supported={!!profile?.capabilities.voiceDiscovery}
  result={voices}
  allowedVoices={profile?.voices ?? []}
  busy={voicesBusy}
  disabled={busy}
  {compact}
  actions={voiceActions}
  onChoose={onVoice}
  onDiscover={onDiscoverVoices}
/>
<SpeechSpeedControl
  id={compact ? "quick-speech-speed" : "tts-speed"}
  value={speech.speed}
  supported={!!profile?.capabilities.speechSpeed}
  disabled={busy}
  {compact}
  onChange={onSpeed}
/>

{#if profile?.capabilities.speechLanguage || profile?.capabilities.speechInstructions}
  <div class={compact ? "space-y-4" : "space-y-4 p-5"}>
    {#if profile.capabilities.speechLanguage}
      <div class="space-y-1.5">
        <label for={compact ? "quick-speech-language" : "tts-language"} class="text-sm font-medium"
          >Speech language</label
        >
        <LanguagePicker
          id={compact ? "quick-speech-language" : "tts-language"}
          languages={profile.languages ?? []}
          restricted
          disabled={busy}
          bind:value={
            () => speech.options.language || "auto",
            (language) => {
              void onOptions({ ...speech.options, language });
            }
          }
        />
      </div>
    {/if}
    {#if profile.capabilities.speechInstructions}
      <div class="space-y-1.5">
        <label
          for={compact ? "quick-speech-instructions" : "tts-instructions"}
          class="text-sm font-medium">Voice style</label
        >
        <textarea
          id={compact ? "quick-speech-instructions" : "tts-instructions"}
          rows="3"
          maxlength="500"
          class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          placeholder="Speak warmly and clearly, with a relaxed pace."
          disabled={busy}
          value={speech.options.instructions}
          oninput={(event) => {
            if (!immediate)
              void onOptions({ ...speech.options, instructions: event.currentTarget.value });
          }}
          onchange={(event) => {
            if (immediate)
              void onOptions({ ...speech.options, instructions: event.currentTarget.value });
          }}
        ></textarea>
        <p class="text-xs leading-relaxed text-muted-foreground">
          Describe tone, emotion, or delivery. Leave empty for the selected voice’s usual style.
        </p>
      </div>
    {/if}
  </div>
{/if}
