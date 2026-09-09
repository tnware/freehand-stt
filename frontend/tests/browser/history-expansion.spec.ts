import { test, expect } from "./fixtures";

for (const width of [560, 1156]) {
  test(`mouse wheel scrolls history entries at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 560 });
    await page.goto("/tests/browser/app/?view=workspace&theme=dark&history=expansion");
    if (width < 900) await page.getByRole("button", { name: "History · 2", exact: true }).click();
    const scroll = page.locator(".history-area .overflow-y-auto");
    await expect(scroll).toHaveCount(1);
    await expect
      .poll(() => scroll.evaluate((node) => node.scrollHeight - node.clientHeight))
      .toBeGreaterThan(100);
    await scroll.hover();
    const target = await scroll.evaluate((node) =>
      Math.min(200, node.scrollHeight - node.clientHeight),
    );
    await page.mouse.wheel(0, 200);
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(target);
    await page.getByRole("button", { name: "Update latest", exact: true }).click();
    // Native scroll anchoring may compensate for changed text/controls above
    // the reader, but updating the same entry must not jump back to the top.
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBeGreaterThan(100);
    await scroll.hover();
    await page.mouse.wheel(0, -1000);
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(0);
    await page.mouse.wheel(0, 10000);
    await expect
      .poll(() => scroll.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop))
      .toBeLessThan(2);
    await page.getByRole("button", { name: "Add transcript", exact: true }).click();
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(0);
  });
}

for (const width of [560, 1156]) {
  test(`history follows the newest transcript and preserves manual reading at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/tests/browser/app/?view=workspace&theme=dark&history=expansion");
    if (width < 900) await page.getByRole("button", { name: "History · 2", exact: true }).click();
    const rows = page.locator(".history-entry");
    const headers = page.locator(".history-disclosure");
    await expect(headers.first()).toHaveAttribute("aria-expanded", "true");
    await expect(headers.nth(1)).toHaveAttribute("aria-expanded", "false");
    const fullText = rows.first().getByRole("textbox");
    await expect(fullText).toContainText("The final paragraph is visible too");
    expect(await fullText.evaluate((element) => element.scrollHeight <= element.clientHeight)).toBe(
      true,
    );
    const preview = rows.nth(1).locator(".transcript-toggle > span");
    expect(await preview.evaluate((element) => element.scrollHeight > element.clientHeight)).toBe(
      true,
    );
    await page.screenshot({ path: info.outputPath(`history-latest-${width}.png`) });

    // Readers can open older results; a same-ID cleanup update must not reset them.
    await headers.nth(1).focus();
    await headers.nth(1).press("Enter");
    await expect(headers.nth(1)).toHaveAttribute("aria-expanded", "true");
    await page.getByRole("button", { name: "Update latest", exact: true }).click();
    await expect(headers.nth(1)).toHaveAttribute("aria-expanded", "true");
    await rows.first().getByRole("button", { name: "Compare raw and cleaned transcripts" }).click();
    await page.getByRole("button", { name: "Update latest", exact: true }).click();
    await expect(
      rows.first().getByRole("region", { name: "Raw transcript", exact: true }),
    ).toBeVisible();

    // A new arrival opens in full and closes every previous result and comparison.
    await page.getByRole("button", { name: "Add transcript", exact: true }).click();
    await expect(rows).toHaveCount(3);
    await expect(headers.first()).toHaveAttribute("aria-expanded", "true");
    await expect(headers.nth(1)).toHaveAttribute("aria-expanded", "false");
    await expect(headers.nth(2)).toHaveAttribute("aria-expanded", "false");
    await expect(page.getByRole("region", { name: "Raw transcript", exact: true })).toHaveCount(0);
    await headers.nth(1).click();
    await expect(
      rows.nth(1).getByRole("region", { name: "Raw transcript", exact: true }),
    ).toHaveCount(0);

    await rows.first().getByRole("button", { name: "Transcript actions", exact: true }).click();
    await page.getByRole("menuitem", { name: "Remove from history", exact: true }).click();
    await expect(rows).toHaveCount(2);
    await expect(headers.first()).toHaveAttribute("aria-expanded", "true");
    await expect(headers.nth(1)).toHaveAttribute("aria-expanded", "false");
    await page.getByRole("button", { name: "Clear history", exact: true }).click();
    await expect(rows).toHaveCount(0);
    await page.getByRole("button", { name: "Add transcript", exact: true }).click();
    await expect(headers.first()).toHaveAttribute("aria-expanded", "true");
  });
}

test("an unretained file result is fully readable above collapsed history", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&history=expansion");
  await page.getByRole("button", { name: "Show file result", exact: true }).click();
  const result = page.getByRole("article", { name: "Audio file transcript result", exact: true });
  await expect(result).toBeVisible();
  const text = result.getByRole("textbox", { name: "Audio file transcript", exact: true });
  await expect(text).toContainText("The final paragraph is visible too");
  expect(await text.evaluate((element) => element.scrollHeight <= element.clientHeight)).toBe(true);
  for (const header of await page.locator(".history-disclosure").all()) {
    await expect(header).toHaveAttribute("aria-expanded", "false");
  }
});
