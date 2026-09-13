import type { Settings } from "$lib/state";

/** Native Go platform metadata is authoritative; never infer from the WebView. */
export function platformPresentation(platform = "windows") {
  const mac = platform === "darwin";
  return {
    mac,
    supportsMica: !mac,
    followSystem: mac ? "Follow macOS" : "Follow Windows",
    startTitle: mac ? "Start at login" : "Start with Windows",
    startDescription: mac
      ? "Launch quietly in the menu bar when you log in."
      : "Launch quietly in the tray when you sign in.",
    launchDescription: mac
      ? "Open this window on a normal manual launch. Login launches always remain menu-bar-only."
      : "Open this window on a normal manual launch. Windows sign-in launches always remain tray-only.",
    name: mac ? "macOS" : "Windows",
    credentialStore: mac ? "macOS Keychain" : "Windows Credential Manager",
    command: mac ? "Command" : "Win",
    primaryModifier: mac ? "Command" : "Ctrl",
    option: mac ? "Option" : "Alt",
  };
}

export function windowMaterial(
  settings: Pick<Settings, "platform" | "micaActive"> | null | undefined,
): "mica" | "solid" {
  return settings &&
    platformPresentation(settings.platform).supportsMica &&
    settings.micaActive
    ? "mica"
    : "solid";
}
