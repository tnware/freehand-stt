import { platformPresentation } from "$lib/platform";
import { AppearanceMode, type Settings } from "$lib/state";

export type RendererAppearanceMode = "system" | "light" | "dark";

export function activeAppearanceMode(
  settings: Pick<Settings, "appearanceModeActive"> | null | undefined,
): RendererAppearanceMode {
  switch (settings?.appearanceModeActive) {
    case AppearanceMode.AppearanceModeLight:
      return "light";
    case AppearanceMode.AppearanceModeDark:
      return "dark";
    default:
      return "system";
  }
}

export function appearanceRestartRequired(
  settings: Pick<
    Settings,
    | "platform"
    | "useMica"
    | "micaActive"
    | "appearanceMode"
    | "appearanceModeActive"
  >,
): boolean {
  if (platformPresentation(settings.platform).mac) {
    return settings.appearanceMode !== settings.appearanceModeActive;
  }
  return (
    settings.useMica !== settings.micaActive ||
    (!settings.useMica &&
      settings.appearanceMode !== settings.appearanceModeActive)
  );
}
