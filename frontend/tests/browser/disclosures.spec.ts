import { test, expect } from "./fixtures";

for (const theme of ["light", "dark"] as const) {
  for (const width of [560, 1280]) {
    test(`disclosures keep keyboard access and contained content at ${width}px / ${theme}`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 820 });
      await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
      await page.goto(
        `/tests/browser/app/?main&runtime&runtime-ready&theme=${theme}`,
      );
      await page
        .getByRole("button", { name: "Voice settings", exact: true })
        .click();
      const inspector = page.locator('[data-pane="configuration"]');
      const summary = inspector
        .locator("summary")
        .filter({ hasText: "NeMo transcription controls" });
      const details = summary.locator("..");
      const punctuation = details.getByRole("switch", {
        name: "Automatic punctuation",
        exact: true,
      });
      await expect(punctuation).toBeHidden();
      await summary.focus();
      await summary.press("Enter");
      await expect(punctuation).toBeVisible();
      await expect(summary).toBeFocused();
      await expect(summary).toHaveCSS("outline-style", "solid");
      await expect(summary.locator(".disclosure-chevron")).toHaveCSS(
        "transition-property",
        "none",
      );
      await summary.press("Tab");
      await expect(punctuation).toBeFocused();
      await summary.focus();
      await summary.press("Space");
      await expect(punctuation).toBeHidden();
      await summary.press("Enter");
      expect(
        await inspector.evaluate((el) => el.scrollWidth <= el.clientWidth),
      ).toBe(true);
      await summary.scrollIntoViewIfNeeded();
      await page.screenshot({
        path: info.outputPath("settings-disclosure.png"),
      });
      await inspector
        .getByRole("button", { name: "Close secondary sidebar", exact: true })
        .click();
      await page
        .getByRole("navigation", { name: "Workspace", exact: true })
        .getByRole("button", { name: "Local runtime", exact: true })
        .click();
      for (const name of [
        "Runtime preferences",
        "Binary download source",
        "Manage runtime",
      ]) {
        const trigger = page.getByRole("button", {
          name: new RegExp(`^${name}`),
        });
        const id = await trigger.getAttribute("aria-controls");
        expect(id).toBeTruthy();
        const body = page.locator(`[id="${id}"]`);
        await expect(body).toBeHidden();
        await trigger.focus();
        await trigger.press("Enter");
        await expect(body).toBeVisible();
        await expect(trigger).toHaveAttribute("aria-expanded", "true");
        await expect(trigger.locator(".disclosure-chevron")).toHaveCSS(
          "transition-property",
          "none",
        );
        expect(
          await body.evaluate((el) => el.scrollWidth <= el.clientWidth),
        ).toBe(true);
        if (name === "Binary download source") {
          const nested = body
            .locator("summary")
            .filter({ hasText: "Binary download details" });
          await nested.focus();
          await nested.press("Enter");
          await expect(
            body.getByText("fixture-cpu.zip", { exact: true }),
          ).toBeVisible();
          await nested.scrollIntoViewIfNeeded();
          await page.screenshot({
            path: info.outputPath("runtime-disclosures.png"),
          });
        }
        await trigger.focus();
        await trigger.press("Space");
        await expect(body).toBeHidden();
      }
      expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
    });
  }
}
