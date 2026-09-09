import { test, expect } from "./fixtures";

test("capture duration freezes when transcription clears the recording timestamp", async ({
  page,
}) => {
  await page.clock.install({ time: new Date("2026-09-08T18:00:00Z") });
  await page.setViewportSize({ width: 560, height: 550 });
  await page.goto("/tests/browser/app/?view=workspace&theme=dark&capture-clock");
  await page.getByRole("button", { name: "Simulate recording", exact: true }).click();
  const timer = page.locator('[aria-label="Recording duration"]');
  await expect(timer).toHaveText("00:04");
  await page.getByRole("button", { name: "Simulate transcription", exact: true }).click();
  await expect(timer).toHaveText("00:04");
  await page.clock.fastForward(60000);
  await expect(timer).toHaveText("00:04");
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});

for (const width of [560, 1100]) {
  test(`jump to latest stays outside transcript text at ${width}px`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 300 });
    await page.goto("/tests/browser/app/?view=transcript&theme=dark");
    const scroll = page.getByRole("region", { name: "Current result" }).locator(".overflow-y-auto");
    const gap = () =>
      scroll.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop);
    await expect.poll(gap).toBeLessThan(2);
    await scroll.hover();
    await page.mouse.wheel(0, -300);
    const jump = page.getByRole("button", { name: "Jump to latest", exact: true });
    await expect(jump).toBeVisible();
    const scrollBox = (await scroll.boundingBox())!;
    const jumpBox = (await jump.boundingBox())!;
    expect(jumpBox.y).toBeGreaterThanOrEqual(scrollBox.y + scrollBox.height);
    const top = await scroll.evaluate((node) => node.scrollTop);
    await page.getByRole("button", { name: "Append text", exact: true }).click();
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(top);
    await page.getByRole("button", { name: "Finalize", exact: true }).click();
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(top);
    await page.screenshot({ path: info.outputPath(`transcript-jump-${width}.png`) });
    await jump.click();
    await expect(jump).toHaveCount(0);
    await expect.poll(gap).toBeLessThan(2);
  });
}
