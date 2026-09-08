import { test, expect } from "./fixtures";
for (const width of [560, 1156]) {
  test(`history keeps common actions compact and secondary actions accessible at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await page.goto("/tests/browser/app/?view=workspace&theme=dark&history=review");
    if (width < 900) await page.getByRole("button", { name: "History · 2", exact: true }).click();
    const entries = page.locator(".history-entry");
    const first = entries.first();
    const footer = first.locator(".history-footer");
    await expect(
      first.getByRole("button", { name: "Copy cleaned transcript", exact: true }),
    ).toBeVisible();
    await expect(
      first.getByRole("button", { name: "Listen to transcript", exact: true }),
    ).toBeVisible();
    await expect(page.getByText("Cleanup could not finish", { exact: false })).toBeVisible();
    const metadata = await footer.locator(":scope > div").first().boundingBox();
    const actions = await footer.locator(".history-actions").boundingBox();
    expect(
      Math.abs(metadata!.y + metadata!.height / 2 - actions!.y - actions!.height / 2),
    ).toBeLessThan(2);
    await first.getByRole("button", { name: "Compare raw and cleaned transcripts" }).click();
    await expect(first.getByRole("region", { name: "Raw transcript", exact: true })).toBeVisible();
    await first.getByRole("button", { name: "Copy raw transcript", exact: true }).click();
    await expect(
      first.getByRole("button", { name: "Raw transcript copied", exact: true }),
    ).toBeVisible();
    const menu = first.getByRole("button", { name: "Transcript actions", exact: true });
    await menu.focus();
    await menu.press("Enter");
    await expect(
      page.getByRole("menuitem", { name: "Transcription details", exact: true }),
    ).toBeEnabled();
    await expect(
      page.getByRole("menuitem", { name: "Remove from history", exact: true }),
    ).toBeInViewport();
    await page.screenshot({ path: info.outputPath(`history-menu-${width}.png`) });
    await page.keyboard.press("Escape");
    await expect(menu).toBeFocused();
    await menu.click();
    await page.getByRole("menuitem", { name: "Remove from history", exact: true }).click();
    await expect(entries).toHaveCount(1);
    await page.screenshot({ path: info.outputPath(`history-warning-${width}.png`) });
  });

  test(`file error details are readable without shifting the workspace at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await page.goto("/tests/browser/app/?view=workspace&theme=dark&file-error");
    await page.getByRole("tab", { name: "Audio file", exact: true }).click();
    const transport = page.locator(".transport");
    const before = await transport.boundingBox();
    const trigger = page.getByRole("button", {
      name: "File transcription error details",
      exact: true,
    });
    await trigger.focus();
    await trigger.press("Enter");
    const details = page.getByRole("dialog", {
      name: "File transcription error details",
      exact: true,
    });
    await expect(details).toBeInViewport();
    await expect(details).toContainText("then retry this file.");
    expect(await transport.boundingBox()).toEqual(before);
    await page.screenshot({ path: info.outputPath(`file-error-${width}.png`) });
    await page.keyboard.press("Escape");
    await expect(trigger).toBeFocused();
    await page.getByRole("button", { name: "Retry", exact: true }).click();
    await expect(trigger).toHaveCount(0);
    await expect(transport).toHaveAttribute("data-state", "uploading");
    expect(await transport.boundingBox()).toEqual(before);
  });
}
