import { test, expect } from "./fixtures";

for (const scale of [1, 1.25, 1.5]) {
  test(`settings keep scrolling inside their panes at ${scale * 100}% scale`, async ({ page }) => {
    await page.setViewportSize({ width: 860, height: 640 });
    await page.goto("/tests/browser/app/?workflows&theme=dark");
    await page.evaluate((zoom) => {
      document.documentElement.style.zoom = String(zoom);
    }, scale);
    const pane = page.locator('section[aria-labelledby="settings-section-title"]').locator("..");
    const save = page.getByRole("button", { name: "Save settings", exact: true });
    for (const workflow of ["voice-transcription", "server", "processing", "speech"]) {
      await page.locator(`[data-settings-section="${workflow}"]`).click();
      const before = await save.boundingBox();
      await pane.evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await expect.poll(() => pane.evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
      await expect(save).toBeInViewport();
      expect(await save.boundingBox()).toEqual(before);
      expect(
        await page.evaluate(() => ({
          height: document.documentElement.scrollHeight,
          viewport: innerHeight,
          scroll: document.scrollingElement?.scrollTop,
        })),
      ).toEqual({ height: 640, viewport: 640, scroll: 0 });
    }
    await page.locator('[data-settings-section="voice-transcription"]').click();
    const modelCard = page.locator(".settings-group").filter({ has: page.locator("#voice-model") });
    await expect(modelCard).toHaveCount(1);
    await expect(modelCard.locator("#voice-language")).toBeVisible();
    await expect(modelCard.locator("#voice-prompt")).toBeVisible();
  });
}
