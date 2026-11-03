import { ID as Backend } from "$bindings/compatibility";
import { ID as ModelProfile } from "$bindings/modelprofile";
import type { Settings } from "$lib/state";

/** Presentation only; the recorder checks its immutable settings independently. */
export function liveInsertionUnavailableReason(settings: Settings): string {
  if (!settings.autoInsert)
    return "Choose Direct input to type completed phrases while recording.";
  if (settings.postProcessing.enabled)
    return "Turn off transcript cleanup to type completed phrases while recording. With cleanup on, text is delivered after Stop.";
  const voice = settings.voiceTranscription;
  if (
    voice.compatibilityProfile === Backend.VLLM ||
    voice.compatibilityProfile === Backend.VLLMOmni
  )
    return "vLLM finalizes the transcript after Stop. Live previews are not typed.";
  if (
    voice.compatibilityProfile !== Backend.NeMoSpeechV1 ||
    voice.modelProfile !== ModelProfile.Nemotron35
  )
    return "Requires NeMo-Speech.cpp with the Nemotron 3.5 streaming profile.";
  if (!voice.realtime)
    return "Enable Realtime transcription in Voice settings first.";
  return "";
}

/** Describe the effective delivery mode without treating previews as final text. */
export function realtimeDeliverySummary(settings: Settings): string {
  if (!settings.autoInsert)
    return "Stop to finish. The transcript stays ready for you to copy.";
  if (settings.liveInsert && !liveInsertionUnavailableReason(settings))
    return "Completed phrases are typed while recording. Cancel does not remove text already typed.";
  return settings.postProcessing.enabled
    ? "Stop to finish and clean up the transcript before delivery."
    : "Stop to finish and deliver the transcript.";
}
