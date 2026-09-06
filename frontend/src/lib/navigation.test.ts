import { describe, expect, it } from "vitest";
import { SETTINGS_SECTIONS, sectionByID, sectionsInGroup } from "$lib/navigation";

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
      "connections",
      "history",
    ]);
    expect(sectionsInGroup("features").map((section) => section.label)).toEqual([
      "Transcription",
      "Cleanup",
      "Text to speech",
    ]);
    expect(SETTINGS_SECTIONS.map((section) => section.id)).toEqual(
      ["capture", "features", "application"].flatMap((group) =>
        sectionsInGroup(group as Parameters<typeof sectionsInGroup>[0]).map(
          (section) => section.id,
        ),
      ),
    );
  });

  it("keeps section identifiers unique", () => {
    const ids = SETTINGS_SECTIONS.map((section) => section.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
