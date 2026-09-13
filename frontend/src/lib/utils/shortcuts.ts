import {
  ShortcutAction,
  type ShortcutAssignments,
  type ShortcutPolicy,
} from "$bindings/hotkey";
import { platformPresentation } from "$lib/platform";
import { State, type Settings, type Status } from "$lib/state";

export { ShortcutAction };

export const shortcutLabel = (action: ShortcutAction): string => {
  if (action === ShortcutAction.ToggleRecording) return "Toggle recording";
  if (action === ShortcutAction.ShowFreehand) return "Show Freehand";
  return "Hold to talk";
};

export const shortcutDescription = (action: ShortcutAction): string => {
  if (action === ShortcutAction.ToggleRecording) {
    return "Press once to start recording and press again to stop.";
  }
  if (action === ShortcutAction.ShowFreehand) {
    return "Reveal and focus Freehand from anywhere.";
  }
  return "Hold the chord to record, then release it to finish and transcribe.";
};

const KEY_LABELS: Record<string, string> = {
  alt: "Alt",
  cmd: "Win",
  command: "Win",
  cmdorctrl: "Ctrl",
  control: "Ctrl",
  ctrl: "Ctrl",
  meta: "Win",
  shift: "Shift",
  space: "Space",
  super: "Win",
  win: "Win",
};

/** Spoken names only: persisted native chords remain unchanged. */
const shortcutKeyNames = (value: string, platform = "windows"): string[] =>
  value
    .split("+")
    .map((part) => part.trim())
    .filter(Boolean)
    .map((part) => {
      const key = part.toLowerCase();
      const native = platformPresentation(platform);
      if (["cmd", "command", "meta", "super", "win"].includes(key))
        return native.command;
      if (key === "cmdorctrl") return native.primaryModifier;
      if (key === "alt" || key === "option") return native.option;
      if (native.mac && (key === "ctrl" || key === "control")) return "Control";
      return KEY_LABELS[key] ?? part;
    });

const MAC_GLYPHS: Record<string, string> = {
  Command: "⌘",
  Option: "⌥",
  Control: "⌃",
  Shift: "⇧",
};

/** Native visual labels, separate from the accessible spoken names. */
export const shortcutKeyLabels = (
  value: string,
  platform = "windows",
): string[] =>
  shortcutKeyNames(value, platform).map((name) =>
    platformPresentation(platform).mac ? (MAC_GLYPHS[name] ?? name) : name,
  );

/** Screen-reader wording shared by every visual shortcut renderer. */
export const shortcutSpokenLabel = (
  value: string,
  platform = "windows",
): string => shortcutKeyNames(value, platform).join(" plus ");

export const shortcutValue = (
  settings: Settings,
  action: ShortcutAction,
): string => {
  if (action === ShortcutAction.ToggleRecording) return settings.toggleShortcut;
  if (action === ShortcutAction.ShowFreehand) return settings.showShortcut;
  return settings.holdShortcut ?? "";
};

export const setShortcutValue = (
  settings: Settings,
  action: ShortcutAction,
  value: string,
): void => {
  if (action === ShortcutAction.ToggleRecording)
    settings.toggleShortcut = value;
  else if (action === ShortcutAction.ShowFreehand)
    settings.showShortcut = value;
  else settings.holdShortcut = value;
};

export const shortcutAssignments = (
  settings: Settings,
): ShortcutAssignments => ({
  toggleRecording: settings.toggleShortcut,
  showFreehand: settings.showShortcut,
  holdToTalk: settings.holdShortcut ?? "",
});

const groupList = (groups: string[] | null): string => groups?.join(", ") ?? "";

/** Human copy derived from backend capability metadata, not a second policy. */
export const shortcutRequirement = (policy: ShortcutPolicy): string => {
  const forms = [
    `a modifier plus ${groupList(policy.modifiedPrimaryGroups)}`,
    `${groupList(policy.dedicatedPrimaryGroups)} on its own`,
  ];
  if (policy.modifierOnlyMinimum > 0) {
    forms.push(`${policy.modifierOnlyMinimum} modifiers on their own`);
  }
  return `${policy.required ? "Required" : "Optional"}: ${forms.join("; or ")}.`;
};

export const isRecommendedShortcut = (
  policy: ShortcutPolicy,
  value: string,
): boolean =>
  value === policy.defaultShortcut ||
  (policy.defaultAliases ?? []).includes(value);

/**
 * Recording a chord suspends the global shortcuts, so it is only offered when
 * no dictation is in flight.
 */
export const shortcutEditingAllowed = (
  status: Status,
  busy: boolean,
): boolean => !busy && [State.Idle, State.Failed].includes(status.state);
