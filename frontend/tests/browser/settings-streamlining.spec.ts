import { test, expect } from "./fixtures";
const section = (page: import("@playwright/test").Page, id: string) =>
  page.locator(`[data-settings-section="${id}"]`);

test("settings search finds detailed options and preserves drafts", async ({ page }) => {
  const search = page.getByRole("textbox", { name: "Find settings" });
  await search.fill("padding");
  await expect(section(page, "audio")).toBeVisible();
  await expect(section(page, "general")).toHaveCount(0);
  await search.press("Enter");
  await expect(section(page, "audio")).toBeFocused();
  await expect(search).toHaveValue("");
  await page.locator("#max-duration").fill("150");
  await search.fill("glow");
  await search.press("Enter");
  await expect(section(page, "overlay")).toBeFocused();
  await search.fill("unfindable-setting");
  await expect(page.getByText("No matching settings.")).toBeVisible();
  await search.press("Escape");
  await expect(search).toHaveValue("");
  await section(page, "audio").click();
  await expect(page.locator("#max-duration")).toHaveValue("150");
});

for (const width of [860, 520]) {
  test(`audio tuning stays available and validation reveals it at ${width}px`, async ({
    page,
    saves,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await section(page, "audio").click();
    const tuning = page
      .locator("details")
      .filter({ has: page.locator("summary", { hasText: "Speech detection tuning" }) });
    await expect(tuning).not.toHaveAttribute("open", "");
    await expect(page.getByRole("slider", { name: "Silence indicator delay" })).toBeHidden();
    await page.locator('label[for="silence-trimming"]').click();
    await expect(page.locator("#silence-trimming")).toHaveAttribute("aria-checked", "true");
    await tuning.locator("summary").click();
    const padding = page.getByRole("slider", { name: "Speech padding", exact: true });
    await padding.focus();
    await padding.press("ArrowRight");
    await expect(padding).toHaveAttribute("aria-valuenow", "350");
    await tuning.locator("summary").click();
    await section(page, "history").click();
    await page.getByRole("button", { name: "Save settings", exact: true }).click();
    await saves.complete(await saves.waitForStart(), "invalid-speech-padding");
    await expect(section(page, "audio")).toHaveAttribute("aria-current", "page");
    await expect(tuning).toHaveAttribute("open", "");
    await expect(padding).toBeFocused();
    await expect(padding).toHaveAttribute("aria-valuenow", "350");
    await tuning.locator("summary").click();
    await section(page, "general").click();
    await section(page, "audio").click();
    await page.screenshot({ path: info.outputPath(`audio-${width}.png`) });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
      true,
    );
  });
}

test("overlay has compact defaults and retains tuning when it does not apply", async ({
  page,
}, info) => {
  await page.setViewportSize({ width: 860, height: 740 });
  await page.goto("/tests/browser/app/?theme=dark");
  await section(page, "overlay").click();
  const tuning = page
    .locator("details")
    .filter({ has: page.locator("summary", { hasText: "Appearance and behavior" }) });
  await expect(page.getByRole("button", { name: "Preview overlay", exact: true })).toBeVisible();
  await expect(page.getByRole("slider", { name: "Overlay glow strength" })).toBeHidden();
  await page.screenshot({ path: info.outputPath("overlay-compact.png") });
  await tuning.locator("summary").click();
  const glow = page.getByRole("slider", { name: "Overlay glow strength" });
  await glow.focus();
  await glow.press("ArrowLeft");
  await expect(glow).toHaveAttribute("aria-valuenow", "95");
  const surface = page.getByRole("group", { name: "Overlay surface", exact: true });
  await surface.getByRole("radio", { name: "Minimal", exact: true }).click();
  await expect(glow).toHaveAttribute("aria-disabled", "true");
  await surface.getByRole("radio", { name: "Glass", exact: true }).click();
  await expect(glow).not.toHaveAttribute("aria-disabled", "true");
  await expect(glow).toHaveAttribute("aria-valuenow", "95");
  await tuning.locator("summary").click();
  await tuning.locator("summary").click();
  await expect(glow).toHaveAttribute("aria-valuenow", "95");
});
