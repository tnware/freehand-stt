import { test, expect } from "./fixtures";

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
    const fullText = rows.first().locator('[id$="-content"] > p');
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
  const text = result.locator(":scope > p");
  await expect(text).toContainText("The final paragraph is visible too");
  expect(await text.evaluate((element) => element.scrollHeight <= element.clientHeight)).toBe(true);
  for (const header of await page.locator(".history-disclosure").all()) {
    await expect(header).toHaveAttribute("aria-expanded", "false");
  }
});
