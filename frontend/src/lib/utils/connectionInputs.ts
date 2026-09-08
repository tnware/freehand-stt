import { Purpose } from "$bindings/savedconnection";
import type { Settings } from "$lib/state";
import { modelOptions, modelFor } from "$lib/utils/modelSettings";
// Non-secret comparison input. It is never persisted, logged, or sent to the server.
export function connectionInputKey(settings: Settings, purpose: Purpose): string {
  const feature =
    purpose === Purpose.Voice
      ? settings.voiceTranscription
      : purpose === Purpose.Transcription
        ? settings
        : purpose === Purpose.Cleanup
          ? settings.postProcessing
          : settings.textToSpeech;
  return JSON.stringify({
    connection: settings.savedConnections.selected?.[purpose],
    baseURL: feature.baseURL,
    profile: feature.compatibilityProfile,
    allowHTTP: feature.allowInsecureHTTP,
    model: modelFor(settings, purpose),
    options: modelOptions(settings, purpose),
    authentication:
      purpose === Purpose.Cleanup
        ? settings.postProcessingCredentialConfigured
        : purpose === Purpose.Transcription
          ? settings.authenticationMode
          : purpose === Purpose.Voice
            ? settings.voiceTranscription.authenticationMode
            : settings.textToSpeech.authenticationMode,
    health:
      purpose === Purpose.Transcription
        ? settings.healthPath
        : purpose === Purpose.Voice
          ? settings.voiceTranscription.healthPath
          : undefined,
    headers:
      purpose === Purpose.Transcription
        ? settings.headers
        : purpose === Purpose.Voice
          ? settings.voiceTranscription.headers
          : undefined,
  });
}
