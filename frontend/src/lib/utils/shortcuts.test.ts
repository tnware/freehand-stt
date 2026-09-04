import { describe, expect, it } from "vitest";
import { ShortcutAction, type ShortcutPolicy } from "$bindings/hotkey";
import {
  shortcutKeyLabels,
  shortcutRequirement,
  shortcutSpokenLabel,
} from "$lib/utils/shortcuts";

describe("shortcut rendering", () => {
  it("normalizes persisted aliases to Windows-facing key labels", () => {
    expect(shortcutKeyLabels("CmdOrCtrl+Meta+Space")).toEqual([
      "Ctrl",
      "Win",
      "Space",
    ]);
  });

  it("ignores empty chord segments and creates consistent spoken text", () => {
    expect(shortcutSpokenLabel(" Ctrl + Shift + D ")).toBe("Ctrl plus Shift plus D");
    expect(shortcutKeyLabels(" ")).toEqual([]);
  });

  it("renders the backend's Windows-key normalization consistently", () => {
    expect(shortcutKeyLabels("Ctrl+Super+F13")).toEqual(["Ctrl", "Win", "F13"]);
  });

  it("describes action requirements from native policy metadata", () => {
    const policy: ShortcutPolicy = {
      action: ShortcutAction.HoldToTalk,
      required: false,
      modifiedPrimaryGroups: ["A-Z", "Space"],
      dedicatedPrimaryGroups: ["F13-F24"],
      modifierOnlyMinimum: 2,
      externalAvailabilityKnown: false,
    };
    expect(shortcutRequirement(policy)).toBe(
      "Optional: a modifier plus A-Z, Space; or F13-F24 on its own; or 2 modifiers on their own.",
    );
  });
});
