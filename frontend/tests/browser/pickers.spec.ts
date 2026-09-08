import { test, expect } from "./fixtures";
const section = (page: import("@playwright/test").Page, id: string) =>
  page.locator(`[data-settings-section="${id}"]`);

for (const width of [520, 860]) {
  test(`shared profile and language controls preserve drafts at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await page.goto("/tests/browser/app/?workflows&pickers&theme=dark");
    await section(page, "voice-transcription").click();
    const model = page.locator("#voice-model");
    const profile = page.locator("#voice-profile");
    const language = page.locator("#voice-language");
    expect((await model.boundingBox())!.height).toBe((await language.boundingBox())!.height);
    const original = await profile.boundingBox();
    const help = page.getByRole("button", { name: "About this model profile", exact: true });
    await help.focus();
    await help.press("Enter");
    await expect(page.getByRole("dialog", { name: "About this model profile" })).toContainText(
      "standard transcription options",
    );
    expect(await profile.boundingBox()).toEqual(original);
    await page.keyboard.press("Escape");
    await expect(help).toBeFocused();
    await profile.click();
    await expect(page.getByRole("option", { name: /Qwen3-ASR/ })).toBeInViewport();
    await page.screenshot({ path: info.outputPath(`profiles-${width}.png`) });
    await page.getByRole("option", { name: /Qwen3-ASR/ }).click();
    await expect(page.locator("#voice-realtime")).toBeVisible();
    await language.fill("fr");
    await page.getByRole("option", { name: "French", exact: true }).click();
    await expect(language).toHaveValue("French");
    await language.fill("unlisted");
    await expect(page.getByRole("option", { name: /Custom server/ })).toHaveCount(0);
    await expect(page.getByText("No matching language in this model profile.")).toBeVisible();
    await page.keyboard.press("Escape");
    await page.getByRole("button", { name: "Show languages" }).click();
    await page.keyboard.press("End");
    await page.keyboard.press("Enter");
    await expect(language).toHaveValue("Finnish");
    await section(page, "general").click();
    await section(page, "voice-transcription").click();
    await expect(profile).toContainText("Qwen3-ASR");
    await expect(language).toHaveValue("Finnish");
    await page.screenshot({ path: info.outputPath(`voice-controls-${width}.png`) });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
      true,
    );
  });
}

test("unlisted language search still offers custom server values", async ({ page }) => {
  await page.goto("/tests/browser/app/?workflows&pickers");
  const language = page.locator("#language");
  await language.fill("server-specific");
  await page.getByRole("option", { name: "Custom server value…", exact: true }).click();
  await page.locator("#language-custom").fill("custom-en");
  await expect(page.getByText("Custom values are sent to the server unchanged.")).toBeVisible();
  await section(page, "general").click();
  await section(page, "server").click();
  await expect(page.locator("#language-custom")).toHaveValue("custom-en");
});

test("quick Voice uses the same profile controls and preserves immediate-save behavior", async ({
  page,
  saves,
}) => {
  await page.setViewportSize({ width: 900, height: 740 });
  await page.goto("/tests/browser/app/?view=workspace&pickers&theme=dark");
  await page.getByRole("button", { name: "Transcription settings", exact: true }).click();
  const quick = page.getByRole("dialog", { name: "Transcription settings", exact: true });
  await quick.getByRole("button", { name: "About model settings", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "About model settings" })).toContainText(
    "Changes apply immediately",
  );
  await page.keyboard.press("Escape");
  await expect(quick).toBeVisible();
  await quick.locator("#voice-profile").click();
  await page.getByRole("option", { name: /Qwen3-ASR/ }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(quick.locator("#voice-profile")).toContainText("Qwen3-ASR");
  await quick.locator("#voice-language").fill("fr");
  await page.getByRole("option", { name: "French", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "failure");
  await expect(quick.locator("#voice-language")).toHaveValue("English");
  await expect(quick.getByRole("status").filter({ hasText: "Could not save" })).toBeVisible();
});
