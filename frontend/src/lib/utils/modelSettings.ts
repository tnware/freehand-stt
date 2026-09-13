import { Purpose } from "$bindings/savedconnection";
import { ID } from "$bindings/modelprofile";
import { PostProcessingPreset } from "$bindings/config";
import type { Options, TranscriptionOptions } from "$bindings/modelsettings";
import type { TranscriptionOptions as InferenceTranscriptionOptions } from "$bindings/config";
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
function transcriptionOptions(
  options: InferenceTranscriptionOptions,
): TranscriptionOptions {
  return {
    prompt: options.prompt,
    temperatureOverride: options.temperatureOverride,
    temperature: options.temperature,
  };
}

export function modelOptions(settings: Settings, purpose: Purpose): Options {
  const base: Options = {
    speech: { language: "", instructions: "" },
    profile: ID.Generic,
    transcription: {
      prompt: "",
      temperatureOverride: false,
      temperature: 0,
    },
    cleanup: {
      limitOutputTokens: false,
      maxOutputTokens: 0,
      disableReasoning: false,
    },
    voice: "",
  };
  if (purpose === Purpose.Voice)
    return {
      ...base,
      profile: settings.voiceTranscription.modelProfile,
      transcription: transcriptionOptions(
        settings.voiceTranscription.transcriptionOptions,
      ),
    };
  if (purpose === Purpose.Transcription)
    return {
      ...base,
      profile: settings.modelProfile || ID.Generic,
      transcription: transcriptionOptions(settings.transcriptionOptions),
    };
  if (purpose === Purpose.Cleanup) {
    const p = settings.postProcessing;
    return {
      ...base,
      profile:
        p.preset === PostProcessingPreset.PostProcessingPresetS1Mini
          ? ID.S1Mini
          : ID.Generic,
      cleanup: { ...p.generationOptions },
    };
  }
  return {
    ...base,
    profile: settings.textToSpeech.modelProfile || ID.Generic,
    speech: { ...settings.textToSpeech.options },
    voice: settings.textToSpeech.voice,
  };
}
export function savedModelOptions(
  settings: Settings,
  purpose: Purpose,
  model: string,
): Options | undefined {
  return (
    rememberedModels(settings, purpose).find((e) => e.model === model)
      ?.options ?? settings.rememberedModels?.defaults?.[purpose]
  );
}
export function applyModel(
  settings: Settings,
  purpose: Purpose,
  model: string,
): boolean {
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
  // The remembered DTO represents model behavior only; task state stays in place.
  const o = {
    ...options,
    transcription: { ...options.transcription },
    cleanup: { ...options.cleanup },
  };
  if (purpose === Purpose.Voice) {
    Object.assign(settings.voiceTranscription, {
      model,
      modelProfile: o.profile,
      transcriptionOptions: {
        ...settings.voiceTranscription.transcriptionOptions,
        ...o.transcription,
      },
      realtime:
        settings.voiceTranscription.realtime &&
        !!settings.modelProfiles.voiceTranscription?.find(
          (p) => p.id === o.profile,
        )?.capabilities.realtime,
    });
  } else if (purpose === Purpose.Transcription) {
    settings.model = model;
    settings.modelProfile = o.profile;
    settings.transcriptionOptions = {
      ...settings.transcriptionOptions,
      ...o.transcription,
    };
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
      options: { ...o.speech },
    });
  }
  return true;
}
