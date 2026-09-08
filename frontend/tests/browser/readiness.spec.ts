import { test, expect } from "./fixtures";

for (const width of [520, 1000]) {
  test(`first run stays compact and completes only after a saved check at ${width}px`, async ({
    page,
    saves,
  }, info) => {
    await page.setViewportSize({ width, height: width === 520 ? 620 : 740 });
    await page.goto("/tests/browser/app/?view=workspace&setup=first&pickers&theme=dark");
    const setup = page.getByRole("region", { name: "First-run setup", exact: true });
    await expect(setup).toBeVisible();
    await expect(page.getByRole("button", { name: "Finish setup", exact: true })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Back to workspace" })).toHaveCount(0);
    await expect(page.locator("#voice-language")).not.toBeVisible();
    await expect(page.getByText("4 checks ready", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Check connection", exact: true }),
    ).toBeInViewport();
    await page.screenshot({ path: info.outputPath(`setup-${width}.png`) });
    const completed = page.locator("summary").filter({ hasText: "4 checks ready" });
    await completed.focus();
    await completed.press("Enter");
    await expect(page.getByRole("button", { name: "Review Microphone settings" })).toBeVisible();
    await completed.press("Enter");
    await page.locator("summary").filter({ hasText: "Transcription options" }).click();
    await expect(page.locator("#voice-language")).toBeVisible();
    await page.locator("summary").filter({ hasText: "Transcription options" }).click();
    await page.getByRole("button", { name: "Check connection", exact: true }).click();
    await page.getByRole("button", { name: "Finish setup", exact: true }).click();
    await expect(page.getByRole("button", { name: "Finishing…", exact: true })).toBeDisabled();
    await saves.complete(await saves.waitForStart(), "failure");
    await expect(setup).toBeVisible();
    await page.getByRole("button", { name: "Finish setup", exact: true }).click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect(setup).toHaveCount(0);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
      true,
    );
  });
}

test("failed metadata check retries without unlocking first run", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&setup=retry&pickers");
  await page.getByRole("button", { name: "Check connection", exact: true }).click();
  await expect(page.getByRole("button", { name: "Finish setup", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Check again", exact: true }).click();
  await expect(page.getByRole("button", { name: "Finish setup", exact: true })).toBeEnabled();
});

test("missing model can still be discovered and saved during setup", async ({ page, saves }) => {
  await page.goto("/tests/browser/app/?view=workspace&setup=missing-model&pickers");
  await page.getByRole("button", { name: "Refresh models", exact: true }).click();
  await page.locator("#voice-model").fill("speech/stt");
  await page.getByRole("option", { name: /speech\/stt/ }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(page.locator("#voice-model")).toHaveValue("speech/stt");
  await expect(
    page.getByRole("button", { name: /Check connection|Finish setup/, exact: true }),
  ).toBeEnabled();
});

test("microphone recovery is focused, dismissible, and does not block files", async ({
  page,
}, info) => {
  await page.setViewportSize({ width: 860, height: 660 });
  await page.goto("/tests/browser/app/?view=workspace&setup=microphone&pickers&theme=dark");
  await expect(page.getByRole("heading", { name: "Check your microphone" })).toBeVisible();
  await expect(page.locator("#voice-model")).toHaveCount(0);
  await expect(page.getByText("Not checked during this session.")).toHaveCount(0);
  await page.getByRole("button", { name: "Choose microphone" }).click();
  await expect(page.locator("footer")).toContainText("Audio settings");
  await page.screenshot({ path: info.outputPath("microphone-recovery.png") });
  await page.getByRole("button", { name: "Back to workspace" }).click();
  await expect(page.getByRole("region", { name: "Task recovery" })).toHaveCount(0);
  await page.getByRole("tab", { name: "Audio file", exact: true }).click();
  await expect(page.getByRole("region", { name: "Task recovery" })).toHaveCount(0);
});

test("connection recovery keeps its editor collapsed and returns after retry", async ({
  page,
}, info) => {
  await page.goto("/tests/browser/app/?view=workspace&setup=connection&pickers&theme=dark");
  await expect(page.getByRole("heading", { name: "Connection needs attention" })).toBeVisible();
  await expect(page.locator("#voice-model")).not.toBeVisible();
  await page.screenshot({ path: info.outputPath("connection-recovery.png") });
  await page.locator("summary").filter({ hasText: "Connection and model" }).click();
  await expect(page.locator("#voice-model")).toBeVisible();
  await page.getByRole("button", { name: "Check again", exact: true }).click();
  await expect(page.getByRole("region", { name: "Task recovery" })).toHaveCount(0);
});

test("device discovery cannot advance first run", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&setup=loading&pickers");
  await expect(
    page.getByRole("button", { name: "Looking for a microphone…", exact: true }),
  ).toBeDisabled();
  await expect(page.getByRole("button", { name: "Finish setup", exact: true })).toHaveCount(0);
});
