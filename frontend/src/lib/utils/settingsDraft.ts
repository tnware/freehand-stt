import { Purpose } from "$bindings/savedconnection";
import type { Settings } from "$lib/state";
import { applyModel } from "./modelSettings";

export type QuickSettingsPatch = Partial<
  Pick<
    Settings,
    | "microphoneID"
    | "vadEnabled"
    | "silenceTrimming"
    | "autoStopEnabled"
    | "silenceSplitting"
    | "maxDurationSeconds"
    | "autoInsert"
    | "historyEnabled"
    | "overlayEnabled"
    | "language"
  >
> & {
  voiceTranscription?: Partial<Settings["voiceTranscription"]>;
  model?: string;
  textToSpeech?: Partial<
    Pick<Settings["textToSpeech"], "model" | "voice" | "speed" | "options">
  >;
  postProcessing?: Partial<
    Pick<
      Settings["postProcessing"],
      "enabled" | "model" | "preset" | "styling" | "structure" | "context"
    >
  >;
};

/** Keeps the editable draft independent from the backend-confirmed snapshot. */
export const copySettings = (settings: Settings): Settings => ({
  ...settings,
  vocabulary: { ...settings.vocabulary },
  voiceTranscription: {
    ...settings.voiceTranscription,
    headers: { ...settings.voiceTranscription.headers },
    transcriptionOptions: {
      ...settings.voiceTranscription.transcriptionOptions,
    },
  },
  savedConnections: {
    selected: { ...settings.savedConnections.selected },
    entries: (settings.savedConnections.entries ?? []).map((c) => ({
      ...c,
      uses: [...(c.uses ?? [])],
      details: { ...c.details, headers: { ...c.details.headers } },
    })),
  },
  transcriptionOptions: { ...settings.transcriptionOptions },
  headers:
    settings.headers == null ? settings.headers : { ...settings.headers },
  postProcessing: {
    ...settings.postProcessing,
    generationOptions: { ...settings.postProcessing.generationOptions },
  },
  textToSpeech: {
    ...settings.textToSpeech,
    options: { ...settings.textToSpeech.options },
  },
  microphoneID: settings.microphoneID ?? "",
});

/** Builds a quick-control save without changing the confirmed settings snapshot. */
export function quickSettingsDraft(
  settings: Settings,
  patch: QuickSettingsPatch,
): Settings {
  const next = copySettings(settings);
  if (patch.voiceTranscription) {
    if (
      patch.voiceTranscription.model !== undefined &&
      !applyModel(next, Purpose.Voice, patch.voiceTranscription.model.trim())
    )
      throw new Error("Reload settings before selecting a Voice model.");
    next.voiceTranscription = {
      ...next.voiceTranscription,
      ...patch.voiceTranscription,
    };
  }
  if (patch.textToSpeech) {
    const { model, ...options } = patch.textToSpeech;
    if (model !== undefined && !applyModel(next, Purpose.Speech, model.trim()))
      throw new Error("Reload settings before selecting a speech model.");
    next.textToSpeech = { ...next.textToSpeech, ...options };
  }
  if (
    patch.model !== undefined &&
    !applyModel(next, Purpose.Transcription, patch.model.trim())
  )
    throw new Error("Reload settings before selecting a model.");
  if (patch.microphoneID !== undefined) next.microphoneID = patch.microphoneID;
  if (patch.language !== undefined) next.language = patch.language;
  if (patch.vadEnabled !== undefined) next.vadEnabled = patch.vadEnabled;
  if (patch.silenceTrimming !== undefined)
    next.silenceTrimming = patch.silenceTrimming;
  if (patch.autoStopEnabled !== undefined)
    next.autoStopEnabled = patch.autoStopEnabled;
  if (patch.silenceSplitting !== undefined) {
    next.silenceSplitting = patch.silenceSplitting;
  }
  if (patch.maxDurationSeconds !== undefined) {
    next.maxDurationSeconds = patch.maxDurationSeconds;
  }
  if (patch.autoInsert !== undefined) next.autoInsert = patch.autoInsert;
  if (patch.historyEnabled !== undefined)
    next.historyEnabled = patch.historyEnabled;
  if (patch.overlayEnabled !== undefined)
    next.overlayEnabled = patch.overlayEnabled;
  if (
    patch.postProcessing?.model !== undefined &&
    !applyModel(next, Purpose.Cleanup, patch.postProcessing.model.trim())
  )
    throw new Error("Reload settings before selecting a model.");
  if (patch.postProcessing) {
    next.postProcessing = {
      ...next.postProcessing,
      ...patch.postProcessing,
    };
  }
  return next;
}
