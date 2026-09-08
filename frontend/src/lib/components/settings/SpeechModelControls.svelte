<script lang="ts">
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
    modelDetails,
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
    modelDetails?: Snippet;
  } = $props();
  const speech = $derived(settings.textToSpeech);
  const profile = $derived(
    settings.modelProfiles.speech?.find((p) => p.id === (speech.modelProfile || ID.Generic)),
  );
</script>

<RuntimeModelPicker
  id={compact ? "quick-speech-model" : "tts-model"}
  value={speech.model}
  {models}
  {draftModels}
  {compact}
  {immediate}
  savedModels={rememberedModels(settings, Purpose.Speech).map((e) => e.model)}
  busy={busy || modelsBusy}
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
  busy={voicesBusy}
  disabled={busy}
  {compact}
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
