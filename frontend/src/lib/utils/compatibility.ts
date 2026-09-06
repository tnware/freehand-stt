import { ID } from "$bindings/compatibility";
import type { Settings } from "$lib/state";

export function transcriptionProfile(settings: Settings) {
  return settings.modelProfiles.transcription?.find(
    (profile) => profile.id === (settings.modelProfile || ID.Generic),
  );
}

export function usesServerLoadedModel(settings: Settings): boolean {
  return Boolean(settings.compatibilityProfiles.transcription?.find(
    (profile) => profile.id === (settings.compatibilityProfile || ID.Generic),
  )?.capabilities.serverLoadedModel);
}
