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
    expect(sectionsInGroup("delivery").map((section) => section.id)).toEqual([
      "general",
      "speech",
    ]);
    expect(sectionsInGroup("server").map((section) => section.label)).toEqual([
      "Transcription",
      "Post-processing",
    ]);
    expect(sectionsInGroup("data").map((section) => section.id)).toEqual(["history"]);
  });

  it("keeps section identifiers unique", () => {
    const ids = SETTINGS_SECTIONS.map((section) => section.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
