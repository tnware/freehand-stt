import { test, expect } from "./fixtures";

for (const theme of ["light", "dark"] as const) {
  for (const { pane, height } of [
    { pane: "narrow", height: 620 },
    { pane: "wide", height: 620 },
    { pane: "narrow", height: 360 },
  ]) {
    test(`long content fits a ${pane} pane in ${theme} at ${height}px high`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width: 900, height });
      await page.emulateMedia({ colorScheme: theme });
      await page.goto(
        `/tests/browser/app/?view=long-content&pane=${pane}&theme=${theme}`,
      );
      const fits = (node: Element) => node.scrollWidth <= node.clientWidth + 1;
      for (const name of [
        "Long connection diagnostics",
        "Long built-in details",
        "Long model catalog",
        "Long filename",
      ]) {
        const region = page.getByRole("region", { name, exact: true });
        await expect.soft(region, name).toBeVisible();
        expect
          .soft(
            await region.evaluate(fits),
            `${name} has no horizontal overflow`,
          )
          .toBe(true);
      }
      const ownership = page.getByText("Connection ownership & safety", {
        exact: true,
      });
      await ownership.click();
      expect
        .soft(
          await page
            .getByRole("region", { name: "Long built-in details" })
            .evaluate(fits),
        )
        .toBe(true);
      const file = page.getByRole("region", { name: "Long filename" });
      await file.getByRole("button", { name: "Retry", exact: true }).click();
      await expect(
        page.getByRole("status").filter({ hasText: /^Retry requested$/ }),
      ).toHaveCount(1);
      const trigger = page.getByRole("button", {
        name: "Long error message details",
        exact: true,
      });
      await trigger.focus();
      await trigger.press("Enter");
      const dialog = page.getByRole("dialog", {
        name: "Long error message details",
        exact: true,
      });
      await expect(dialog).toBeVisible();
      expect.soft(await dialog.evaluate(fits)).toBe(true);
      const message = dialog.getByRole("region", { name: "Details message" });
      await expect(message).toBeFocused();
      expect(await message.evaluate((node) => node.scrollTop)).toBe(0);
      await message.press("End");
      await expect
        .poll(() => message.evaluate((node) => node.scrollTop))
        .toBeGreaterThan(0);
      // Recovery stays visible while the complete message scrolls independently.
      const action = dialog.getByRole("button", { name: "Open settings" });
      await expect(action).toBeInViewport({ ratio: 1 });
      await message.press("Tab");
      await expect(action).toBeFocused();
      await expect(action).toBeInViewport({ ratio: 1 });
      await page.screenshot({
        path: info.outputPath(`long-content-${pane}-${theme}-${height}.png`),
      });
      await page.keyboard.press("Escape");
      await expect(trigger).toBeFocused();
    });
  }
}
