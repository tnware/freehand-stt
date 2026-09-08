import type { SettingsSectionID } from "$lib/navigation";

/** Presentation target for a Go validation failure, not a second validation schema. */
export interface SettingsValidationIssue {
  field: string;
  section: SettingsSectionID;
  control: string | null;
  message: string;
}

const targets: Record<string, Pick<SettingsValidationIssue, "section" | "control">> = {
  vocabulary: { section: "vocabulary", control: "vocabulary-terms" },
  "voice-transcription": { section: "voice-transcription", control: "voice-connection" },
  maxDurationSeconds: { section: "audio", control: "max-duration" },
  microphoneID: { section: "audio", control: "microphone-select" },
  vadMode: { section: "audio", control: null },
  vadEnabled: { section: "audio", control: "vad-enabled" },
  vadActivitySilenceMilliseconds: {
    section: "audio",
    control: "vad-activity-silence",
  },
  speechPaddingMilliseconds: { section: "audio", control: "speech-padding" },
  autoStopSilenceMilliseconds: {
    section: "audio",
    control: "automatic-stop-silence",
  },
  autoStopMinimumSpeechMilliseconds: {
    section: "audio",
    control: "automatic-stop-minimum-speech",
  },
  segmentSeconds: { section: "audio", control: "segment-duration" },
  segmentSilenceMilliseconds: { section: "audio", control: "segment-silence" },
  transcriptionTimeoutSeconds: {
    section: "server",
    control: "transcription-timeout",
  },
  fileTranscriptionTimeoutSeconds: {
    section: "server",
    control: "file-transcription-timeout",
  },
  compatibilityProfile: { section: "server", control: "saved-connection-stt" },
  baseURL: { section: "server", control: "saved-connection-stt" },
  modelProfile: { section: "server", control: "transcription-model-profile" },
  language: { section: "server", control: "language" },
  postProcessing: { section: "processing", control: null },
  "postProcessing.compatibilityProfile": {
    section: "processing",
    control: "saved-connection-cleanup",
  },
  "postProcessing.baseURL": {
    section: "processing",
    control: "saved-connection-cleanup",
  },
  "postProcessing.model": { section: "processing", control: "cleanup-model" },
  "postProcessing.preset": {
    section: "processing",
    control: "cleanup-model-profile",
  },
  "postProcessing.timeoutSeconds": {
    section: "processing",
    control: "cleanup-timeout",
  },
  "postProcessing.systemPrompt": {
    section: "processing",
    control: "post-processing-custom-instruction",
  },
  "postProcessing.styling": { section: "processing", control: null },
  "postProcessing.structure": { section: "processing", control: null },
  "postProcessing.context": { section: "processing", control: null },
  "textToSpeech.compatibilityProfile": {
    section: "speech",
    control: "saved-connection-speech",
  },
  "textToSpeech.baseURL": {
    section: "speech",
    control: "saved-connection-speech",
  },
  "textToSpeech.modelProfile": {
    section: "speech",
    control: "speech-model-profile",
  },
  "textToSpeech.timeoutSeconds": { section: "speech", control: "tts-timeout" },
  "textToSpeech.voice": { section: "speech", control: "tts-voice" },
  "textToSpeech.speed": { section: "speech", control: "tts-speed" },
  appearanceMode: { section: "general", control: null },
  overlayEnabled: { section: "overlay", control: null },
  toggleShortcut: { section: "shortcuts", control: null },
};

export function settingsValidationIssue(failure: unknown): SettingsValidationIssue | null {
  if (!(failure instanceof Error)) return null;
  const detail = failure.cause;
  if (
    !detail ||
    typeof detail !== "object" ||
    !("kind" in detail) ||
    detail.kind !== "settings_validation" ||
    !("field" in detail) ||
    typeof detail.field !== "string" ||
    !("message" in detail) ||
    typeof detail.message !== "string" ||
    !Object.hasOwn(targets, detail.field)
  )
    return null;
  return {
    field: detail.field,
    ...targets[detail.field],
    message: detail.message,
  };
}

export const SETTINGS_VALIDATION = Symbol("settings-validation");
export interface SettingsValidationContext {
  readonly issue: SettingsValidationIssue | null;
  clear: () => void;
}
