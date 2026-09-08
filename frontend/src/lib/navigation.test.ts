import { describe, expect, it } from "vitest";
import { SETTINGS_GROUPS, SETTINGS_SECTIONS, sectionByID, sectionsInGroup } from "$lib/navigation";

describe("settings navigation", () => {
  it("organizes sections by user task", () => {
    const overlay = sectionByID("overlay");

    expect(overlay.label).toBe("Overlay");
    expect(sectionsInGroup("capture").map((section) => section.id)).toEqual([
      "shortcuts",
      "audio",
      "overlay",
    ]);
    expect(sectionsInGroup("application").map((section) => section.id)).toEqual([
      "general",
      "history",
    ]);
    expect(sectionsInGroup("workflows").map((section) => section.label)).toEqual([
      "Voice transcription",
      "Audio-file transcription",
      "Cleanup",
      "Text to speech",
    ]);
    expect(sectionsInGroup("shared").map((section) => section.id)).toEqual([
      "connections",
      "vocabulary",
    ]);
    expect(SETTINGS_SECTIONS.map((section) => section.id)).toEqual(
      SETTINGS_GROUPS.flatMap((group) => sectionsInGroup(group).map((section) => section.id)),
    );
  });

  it("keeps section identifiers unique", () => {
    const ids = SETTINGS_SECTIONS.map((section) => section.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
