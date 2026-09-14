import { test, expect } from "./fixtures";

for (const theme of ["light", "dark"]) {
  for (const width of [860, 520]) {
    test(`settings actions remain readable and reachable in ${theme} at ${width}px`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({
        colorScheme: theme === "dark" ? "dark" : "light",
      });
      // Native shortcut policy is metadata; no keyboard capture is performed.
      await page.route("**/wails/runtime", (route) =>
        route.fulfill({
          json: ["toggle", "show", "hold"].map((action) => ({
            action,
            required: false,
            modifiedPrimaryGroups: ["A-Z", "0-9", "Space", "F1-F11", "F13-F24"],
            dedicatedPrimaryGroups: ["F13-F24"],
            modifierOnlyMinimum: action === "hold" ? 2 : 0,
            externalAvailabilityKnown: false,
          })),
        }),
      );
      await page.goto(`/tests/browser/app/?workflows&pickers&theme=${theme}`);
      await page.locator('[data-settings-section="general"]').click();
      await page.locator("#show-window-on-launch").click();
      if (theme === "dark")
        await expect(page.locator("html")).toHaveClass(/dark/);
      else await expect(page.locator("html")).not.toHaveClass(/dark/);
      const save = page.getByRole("button", {
        name: "Save and return",
        exact: true,
      });
      await expect(save).toBeEnabled();
      await expect(save).toBeInViewport();

      // Measure the rendered action, including its hover state. This catches
      // unreadable foreground/background combinations without freezing a palette.
      for (const hover of [false, true]) {
        if (hover) await save.hover();
        await expect
          .poll(
            () =>
              save.evaluate((element) => {
                const style = getComputedStyle(element);
                const context = document
                  .createElement("canvas")
                  .getContext("2d")!;
                const luminance = (color: string) => {
                  context.fillStyle = color;
                  context.fillRect(0, 0, 1, 1);
                  const channels = [...context.getImageData(0, 0, 1, 1).data]
                    .slice(0, 3)
                    .map((value) => {
                      const channel = value / 255;
                      return channel <= 0.04045
                        ? channel / 12.92
                        : ((channel + 0.055) / 1.055) ** 2.4;
                    });
                  return (
                    channels[0] * 0.2126 +
                    channels[1] * 0.7152 +
                    channels[2] * 0.0722
                  );
                };
                const foreground = luminance(style.color);
                const background = luminance(style.backgroundColor);
                return (
                  (Math.max(foreground, background) + 0.05) /
                  (Math.min(foreground, background) + 0.05)
                );
              }),
            { message: "enabled primary action has normal-text contrast" },
          )
          .toBeGreaterThanOrEqual(4.5);
      }

      for (const section of ["server", "shortcuts"]) {
        await page.locator(`[data-settings-section="${section}"]`).click();
        if (section === "shortcuts")
          await expect(
            page.getByRole("group", { name: "Toggle recording", exact: true }),
          ).toBeVisible();
        await expect(save).toBeInViewport();
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
        ).toBe(true);
        await page.screenshot({
          path: info.outputPath(`${section}-${theme}-${width}.png`),
        });
      }

      const allowedKeys = page
        .getByText("Allowed keys", { exact: true })
        .first();
      await allowedKeys.focus();
      await allowedKeys.press("Enter");
      await expect(allowedKeys.locator("..")).toHaveAttribute("open", "");
      const recorder = allowedKeys.locator("../..");
      await expect(
        recorder.getByRole("button", { name: "Record", exact: true }),
      ).toBeEnabled();
    });
  }
}
