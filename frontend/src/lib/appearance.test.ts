import { describe, expect, it } from "vitest";
import { AppearanceMode, type Settings } from "$lib/state";
import { activeAppearanceMode, appearanceRestartRequired } from "$lib/appearance";

const appearance = (
  overrides: Partial<
    Pick<Settings, "useMica" | "micaActive" | "appearanceMode" | "appearanceModeActive">
  > = {},
) => ({
  useMica: false,
  micaActive: false,
  appearanceMode: AppearanceMode.AppearanceModeSystem,
  appearanceModeActive: AppearanceMode.AppearanceModeSystem,
  ...overrides,
});

describe("appearance", () => {
  it("maps the backend's applied mode to mode-watcher", () => {
    expect(activeAppearanceMode(appearance())).toBe("system");
    expect(
      activeAppearanceMode(
        appearance({ appearanceModeActive: AppearanceMode.AppearanceModeLight }),
      ),
    ).toBe("light");
    expect(
      activeAppearanceMode(
        appearance({ appearanceModeActive: AppearanceMode.AppearanceModeDark }),
      ),
    ).toBe("dark");
  });

  it("requires restart for material changes and active solid-mode changes", () => {
    expect(appearanceRestartRequired(appearance())).toBe(false);
    expect(appearanceRestartRequired(appearance({ useMica: true }))).toBe(true);
    expect(
      appearanceRestartRequired(
        appearance({ appearanceMode: AppearanceMode.AppearanceModeDark }),
      ),
    ).toBe(true);
  });

  it("preserves a different solid preference without claiming Mica needs restart", () => {
    expect(
      appearanceRestartRequired(
        appearance({
          useMica: true,
          micaActive: true,
          appearanceMode: AppearanceMode.AppearanceModeDark,
        }),
      ),
    ).toBe(false);
  });
});
