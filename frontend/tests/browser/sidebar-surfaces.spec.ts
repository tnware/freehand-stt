import { test, expect } from "./fixtures";

for (const theme of ["light", "dark"] as const) {
  for (const width of [1280, 560]) {
    test(`managed workflow surfaces stay consistent across both sidebars at ${width}px in ${theme}`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 900 });
      await page.emulateMedia({ colorScheme: theme });
      await page.goto(
        `/tests/browser/app/?main&runtime&runtime-ready&runtime-combined&theme=${theme}`,
      );
      const primary = page.locator("#workbench-primary-sidebar");
      const inspector = page.locator('[data-pane="configuration"]');
      const toggle = page.getByRole("button", {
        name: "Toggle primary sidebar",
        exact: true,
      });
      for (const [workflow, options, modelLabel] of [
        ["Voice transcription", "Voice settings", "Selected model"],
        ["Audio file", "Audio file settings", "Selected model"],
        ["Text to speech", "Text to speech settings", "Selected speech model"],
      ]) {
        await page
          .getByRole("navigation", { name: "Workspace", exact: true })
          .getByRole("button", { name: workflow, exact: true })
          .click();
        if (!(await primary.isVisible())) await toggle.click();
        const local = primary.getByRole("group", {
          name: "Local runtime Local speech",
          exact: true,
        });
        await expect(local).toBeVisible();
        const surface = local.locator(".runtime-surface");
        const model = local.getByRole("button", {
          name: modelLabel,
          exact: true,
        });
        await expect(model).toHaveCSS("height", "32px");
        await expect(local.locator("label")).toBeInViewport();
        await expect(
          local.getByRole("status", {
            name: "Local runtime status",
            exact: true,
          }),
        ).toContainText("Stopped");
        await expect(
          local.getByRole("button", { name: "Set up runtime", exact: true }),
        ).toBeEnabled();
        const visual = await surface.evaluate((el) => {
          const style = getComputedStyle(el);
          return {
            background: style.backgroundColor,
            border: style.borderColor,
            padding: style.padding,
          };
        });
        expect(
          await primary.evaluate((el) => el.scrollWidth <= el.clientWidth),
        ).toBe(true);
        await page.screenshot({
          path: info.outputPath(`${workflow}-primary.png`),
        });
        if (width < 700) await toggle.click();
        await page.getByRole("button", { name: options, exact: true }).click();
        const detail = inspector.getByRole("group", {
          name: "Local runtime Local speech",
          exact: true,
        });
        await expect(detail).toBeVisible();
        await expect(
          inspector.getByRole("region", {
            name: "Active connection",
            exact: true,
          }),
        ).toContainText("Local · CPU · Stopped");
        await expect(
          detail.getByRole("button", { name: modelLabel, exact: true }),
        ).toHaveCSS("height", "32px");
        expect(
          await detail.locator(".runtime-surface").evaluate((el) => {
            const style = getComputedStyle(el);
            return {
              background: style.backgroundColor,
              border: style.borderColor,
              padding: style.padding,
            };
          }),
        ).toEqual(visual);
        expect(
          await inspector.evaluate((el) => el.scrollWidth <= el.clientWidth),
        ).toBe(true);
        await detail
          .getByRole("button", { name: "View output", exact: true })
          .scrollIntoViewIfNeeded();
        await page.screenshot({
          path: info.outputPath(`${workflow}-inspector.png`),
        });
        await inspector
          .getByRole("button", { name: "Close secondary sidebar", exact: true })
          .click();
      }
      expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
    });
  }
}
