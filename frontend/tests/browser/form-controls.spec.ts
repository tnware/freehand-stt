import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

for (const theme of ["dark", "light"] as const) {
  for (const width of [520, 1280]) {
    test(`fields share geometry and focus, and switches stay beside labels at ${width}px in ${theme}`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
      await page.goto(`/tests/browser/app/?workflows&pickers&theme=${theme}`);
      await openSection(page, "audio");
      const input = page.locator("#max-duration");
      await page.keyboard.press("Tab");
      await input.focus();
      const appearance = await input.evaluate((el) => {
        const s = getComputedStyle(el);
        return {
          height: s.height,
          border: s.borderColor,
          radius: s.borderRadius,
          shadow: s.boxShadow,
          padding: s.paddingLeft,
        };
      });
      expect(appearance.height).toBe("32px");
      expect(appearance.shadow).not.toBe("none");
      await openSection(page, "voice-transcription");
      const pane = page.locator('[data-pane="configuration"]');
      for (const id of [
        "voice-model",
        "voice-language",
        "voice-profile",
        "saved-connection-voice",
      ]) {
        const field = pane.locator(`[id="${id}"]`);
        await page.keyboard.press("Tab");
        await field.focus();
        await expect(field).toBeFocused();
        const actual = await field.evaluate((el) => {
          const s = getComputedStyle(el);
          return {
            height: s.height,
            border: s.borderColor,
            radius: s.borderRadius,
            shadow: s.boxShadow,
            padding: s.paddingLeft,
          };
        });
        expect(actual.height).toBe(appearance.height);
        expect(actual.border).toBe(appearance.border);
        expect(actual.radius).toBe(appearance.radius);
        expect(actual.shadow).toBe(appearance.shadow);
        // Connection artwork reserves its own leading space.
        if (id !== "saved-connection-voice")
          expect(actual.padding).toBe(appearance.padding);
      }
      const trigger = pane.getByRole("button", {
        name: "Show languages",
        exact: true,
      });
      await trigger.focus();
      await expect(trigger).toBeFocused();
      expect(
        await trigger.evaluate((el) => getComputedStyle(el).boxShadow),
      ).not.toBe("none");
      await trigger.press("Enter");
      await expect(page.getByRole("listbox")).toBeVisible();
      await page.keyboard.press("Escape");
      await page.screenshot({
        path: info.outputPath(`fields-${width}-${theme}.png`),
      });
      await openSection(page, "general");
      for (const id of [
        "start-with-windows",
        "show-window-on-launch",
        "check-for-updates",
      ]) {
        const control = page.locator(`[id="${id}"]`);
        const label = page.locator(`label[for="${id}"]`);
        await control.scrollIntoViewIfNeeded();
        const controlBox = (await control.boundingBox())!;
        const labelBox = (await label.boundingBox())!;
        expect(Math.abs(controlBox.y - labelBox.y)).toBeLessThanOrEqual(3);
        expect(controlBox.x).toBeGreaterThanOrEqual(
          labelBox.x + labelBox.width,
        );
      }
      await page.locator("#start-with-windows").scrollIntoViewIfNeeded();
      await page.screenshot({
        path: info.outputPath(`switches-${width}-${theme}.png`),
      });
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    });
  }
}
