import { test, expect } from "./fixtures";
import type { Locator, Page } from "@playwright/test";

const selected = (page: Page) => page.evaluate(() => getSelection()?.toString() ?? "");
async function selectExcerpt(text: Locator) {
  await text.focus();
  return text.evaluate((node) => {
    const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT);
    const first = walker.nextNode()!;
    const range = document.createRange();
    range.setStart(first, 5);
    range.setEnd(first, 24);
    getSelection()!.removeAllRanges();
    getSelection()!.addRange(range);
    return range.toString();
  });
}

for (const width of [560, 1156]) {
  test(`transcript selection survives updates and uses native copy at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 560 });
    await page.goto("/tests/browser/app/?view=transcript");
    const text = page.getByRole("textbox", { name: "Live transcript", exact: true });
    const scroll = page.getByRole("region", { name: "Current result" }).locator(".overflow-y-auto");
    await text.focus();
    await text.press("Home");
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(0);
    await text.press("PageDown");
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBeGreaterThan(100);
    await text.press("Home");
    await text.press("Control+a");
    expect(await selected(page)).toBe(await text.textContent());
    const excerpt = await selectExcerpt(text);
    const before = await text.textContent();
    const top = await scroll.evaluate((node) => node.scrollTop);
    // Dispatch an external update without moving focus or changing the selection.
    await page
      .getByRole("button", { name: "Append text", exact: true })
      .evaluate((node: HTMLButtonElement) => node.click());
    await expect(text).toHaveText(before!);
    expect(await selected(page)).toBe(excerpt);
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(top);
    // Observe the browser's actual copy event without replacing the user's OS clipboard.
    await page.evaluate(() =>
      document.addEventListener(
        "copy",
        (event) => {
          document.documentElement.dataset.copiedExcerpt = getSelection()?.toString();
          event.preventDefault();
        },
        { once: true },
      ),
    );
    await text.press("Control+c");
    await expect(page.locator("html")).toHaveAttribute("data-copied-excerpt", excerpt);
    await page
      .getByRole("button", { name: "Finalize", exact: true })
      .evaluate((node: HTMLButtonElement) => node.click());
    const final = page.getByRole("textbox", { name: "Current transcript", exact: true });
    await expect(final).toHaveText(before!);
    expect(await selected(page)).toBe(excerpt);
    await page.evaluate(() => getSelection()?.removeAllRanges());
    await expect(final).toContainText("Line 40:");
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(top);
    await final.press("End");
    await expect
      .poll(() => scroll.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop))
      .toBeLessThan(2);
    await final.press("PageUp");
    await expect
      .poll(() => scroll.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop))
      .toBeGreaterThan(100);
    await final.press("Control+a");
    await page
      .getByRole("button", { name: "New recording", exact: true })
      .evaluate((node: HTMLButtonElement) => node.click());
    await expect(text).not.toContainText("Line 40:");
    expect(await selected(page)).toBe("");
  });

  test(`history and comparison text have scoped selection and keyboard scrolling at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 560 });
    await page.goto("/tests/browser/app/?view=workspace&history=expansion");
    if (width < 900) await page.getByRole("button", { name: "History · 2", exact: true }).click();
    const row = page.locator(".history-entry").first();
    const text = row.getByRole("textbox");
    await text.focus();
    await text.press("Control+a");
    const original = await text.textContent();
    expect(await selected(page)).toBe(original);
    const scroll = page.locator(".history-area .overflow-y-auto");
    await text.press("End");
    await expect
      .poll(() => scroll.evaluate((node) => node.scrollHeight - node.clientHeight - node.scrollTop))
      .toBeLessThan(2);
    await text.press("Home");
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBe(0);
    await page
      .getByRole("button", { name: "Update latest", exact: true })
      .evaluate((node: HTMLButtonElement) => node.click());
    await expect(text).toHaveText(original!);
    expect(await selected(page)).toBe(original);
    await page.evaluate(() => getSelection()?.removeAllRanges());
    await expect(text).toContainText("Cleaned.");
    await row
      .getByRole("button", { name: "Compare raw and cleaned transcripts", exact: true })
      .click();
    const raw = row.getByRole("textbox", { name: "Raw transcript text", exact: true });
    const cleaned = row.getByRole("textbox", { name: "Cleaned transcript text", exact: true });
    await raw.focus();
    await raw.press("Control+a");
    expect(await selected(page)).toBe(original);
    await cleaned.focus();
    await cleaned.press("Control+a");
    expect(await selected(page)).toBe("Cleaned. " + original);
    await expect(cleaned.locator(".diff-added")).toContainText("Cleaned.");
  });
}
