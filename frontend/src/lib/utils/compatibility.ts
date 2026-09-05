import { ID } from "$bindings/compatibility";
import type { Settings } from "$lib/state";

export function transcriptionProfile(settings: Settings) {
  return settings.compatibilityProfiles.transcription?.find(
    (profile) => profile.id === (settings.compatibilityProfile || ID.Generic),
  );
}

export function usesServerLoadedModel(settings: Settings): boolean {
  return Boolean(transcriptionProfile(settings)?.capabilities.serverLoadedModel);
}
