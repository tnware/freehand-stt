import { describe, expect, it } from "vitest";
import { platformPresentation, windowMaterial } from "$lib/platform";

describe("native platform presentation", () => {
  it("uses macOS wording and hides unsupported Mica", () => {
    expect(platformPresentation("darwin")).toMatchObject({
      name: "macOS",
      credentialStore: "macOS Keychain",
      startTitle: "Start at login",
      followSystem: "Follow macOS",
      supportsMica: false,
    });
    expect(windowMaterial({ platform: "darwin", micaActive: true })).toBe(
      "solid",
    );
  });
  it("preserves Windows presentation and active material", () => {
    expect(platformPresentation("windows")).toMatchObject({
      startTitle: "Start with Windows",
      supportsMica: true,
    });
    expect(windowMaterial({ platform: "windows", micaActive: true })).toBe(
      "mica",
    );
    expect(windowMaterial(null)).toBe("solid");
  });
});
