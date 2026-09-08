import { Purpose } from "$bindings/savedconnection";
import { ID } from "$bindings/modelprofile";
import { PostProcessingPreset } from "$bindings/config";
import type { Options } from "$bindings/modelsettings";
import type { Settings } from "$lib/state";

export function modelFor(settings: Settings, purpose: Purpose): string {
  return purpose === Purpose.Voice
    ? settings.voiceTranscription.model
    : purpose === Purpose.Transcription
      ? settings.model
      : purpose === Purpose.Cleanup
        ? settings.postProcessing.model
        : settings.textToSpeech.model;
}
export function rememberedModels(settings: Settings, purpose: Purpose) {
  const connection = settings.savedConnections.selected?.[purpose];
  return (settings.rememberedModels?.entries ?? []).filter(
    (e) => e.connectionID === connection && e.purpose === purpose,
  );
}
export function modelOptions(settings: Settings, purpose: Purpose): Options {
  const base: Options = {
    realtime: { vocabulary: "", boost: 0 },
    profile: ID.Generic,
    language: "",
    transcription: { prompt: "", hotwords: "", temperatureOverride: false, temperature: 0 },
    cleanup: { limitOutputTokens: false, maxOutputTokens: 0, disableReasoning: false },
    systemPrompt: "",
    styling: "",
    structure: "",
    context: "",
    voice: "",
    speed: 0,
  };
  if (purpose === Purpose.Voice)
    return {
      ...base,
      profile: settings.voiceTranscription.modelProfile,
      language: settings.voiceTranscription.language,
      realtime: { ...settings.voiceTranscription.options },
      transcription: { ...settings.voiceTranscription.transcriptionOptions },
    };
  if (purpose === Purpose.Transcription)
    return {
      ...base,
      profile: settings.modelProfile || ID.Generic,
      language: settings.language ?? "",
      transcription: { ...settings.transcriptionOptions },
    };
  if (purpose === Purpose.Cleanup) {
    const p = settings.postProcessing;
    return {
      ...base,
      profile:
        p.preset === PostProcessingPreset.PostProcessingPresetS1Mini ? ID.S1Mini : ID.Generic,
      cleanup: { ...p.generationOptions },
      systemPrompt: p.systemPrompt,
      styling: p.styling,
      structure: p.structure,
      context: p.context,
    };
  }
  return {
    ...base,
    profile: settings.textToSpeech.modelProfile || ID.Generic,
    voice: settings.textToSpeech.voice,
    speed: settings.textToSpeech.speed,
  };
}
export function savedModelOptions(
  settings: Settings,
  purpose: Purpose,
  model: string,
): Options | undefined {
  return (
    rememberedModels(settings, purpose).find((e) => e.model === model)?.options ??
    settings.rememberedModels?.defaults?.[purpose]
  );
}
export function applyModel(settings: Settings, purpose: Purpose, model: string): boolean {
  const options = savedModelOptions(settings, purpose, model);
  if (!options) return false;
  applyModelOptions(settings, purpose, model, options);
  return true;
}
export function applyModelOptions(
  settings: Settings,
  purpose: Purpose,
  model: string,
  options: Options,
) {
  // Restore engine options while preserving task intent. Historical task fields
  // remain in the saved snapshot format, but do not control model selection.
  const o = {
    ...options,
    transcription: { ...options.transcription },
    cleanup: { ...options.cleanup },
  };
  if (purpose === Purpose.Voice) {
    Object.assign(settings.voiceTranscription, {
      model,
      modelProfile: o.profile,
      options: { ...o.realtime },
      transcriptionOptions: o.transcription,
      realtime:
        settings.voiceTranscription.realtime &&
        !!settings.modelProfiles.voiceTranscription?.find((p) => p.id === o.profile)?.capabilities
          .realtime,
    });
  } else if (purpose === Purpose.Transcription) {
    settings.model = model;
    settings.modelProfile = o.profile;
    settings.transcriptionOptions = o.transcription;
  } else if (purpose === Purpose.Cleanup) {
    Object.assign(settings.postProcessing, {
      model,
      preset:
        o.profile === ID.S1Mini
          ? PostProcessingPreset.PostProcessingPresetS1Mini
          : PostProcessingPreset.PostProcessingPresetGeneric,
      generationOptions: o.cleanup,
    });
  } else {
    Object.assign(settings.textToSpeech, {
      model,
      modelProfile: o.profile,
      voice: o.voice,
    });
  }
  return true;
}
