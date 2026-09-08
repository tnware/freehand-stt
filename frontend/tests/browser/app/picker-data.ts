import { ID, type Profile } from "$bindings/modelprofile";
import type { Settings } from "$lib/state";

// Synthetic contracts for exercising shared controls, never inferred from a server inventory.
export function configurePickerFixture(settings: Settings) {
  const generic: Profile = {
    id: ID.Generic,
    name: "Generic",
    description: "Use the standard transcription options supported by this connection.",
    reasoningOffRequired: false,
    capabilities: {
      realtime: false,
      serverLoadedModel: false,
      vllmTranscriptionEvents: false,
      cleanupOutputLimit: false,
      cleanupDisableReasoning: false,
      fileStreaming: false,
      typedTranscriptionEvents: false,
      legacyTranscriptionSegments: false,
      languageHint: true,
      voiceDiscovery: false,
      speechInstructions: false,
      speechLanguage: false,
      speechSpeed: true,
      transcriptionPrompt: true,
      transcriptionHotwords: false,
      transcriptionTemperature: true,
    },
  };
  const languages = [
    ["en", "English"],
    ["zh", "Chinese"],
    ["fr", "French"],
    ["de", "German"],
    ["it", "Italian"],
    ["ja", "Japanese"],
    ["ko", "Korean"],
    ["pt", "Portuguese"],
    ["ru", "Russian"],
    ["es", "Spanish"],
    ["ar", "Arabic"],
    ["hi", "Hindi"],
    ["tr", "Turkish"],
    ["vi", "Vietnamese"],
    ["id", "Indonesian"],
    ["th", "Thai"],
    ["nl", "Dutch"],
    ["pl", "Polish"],
    ["sv", "Swedish"],
    ["fi", "Finnish"],
  ].map(([code, label]) => ({ code, label }));
  const qwen: Profile = {
    ...generic,
    id: ID.Qwen3ASR,
    name: "Qwen3-ASR",
    description:
      "Transcription with supported languages and context hints. Realtime uses automatic language detection.",
    languages,
    capabilities: { ...generic.capabilities, realtime: true },
  };
  settings.transcriptionLanguages = languages;
  settings.modelProfiles.voiceTranscription = [generic, qwen];
  settings.modelProfiles.transcription = [generic, qwen];
}
