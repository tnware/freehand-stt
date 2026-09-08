import { test, expect } from "./fixtures";
const status = (page: import("@playwright/test").Page) =>
  page.getByRole("button", { name: /connection status:/ });
const panel = (page: import("@playwright/test").Page) =>
  page.getByRole("dialog", { name: "Active connection", exact: true });

for (const width of [520, 1000]) {
  test(`footer connection details follow the task and stay keyboard accessible at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 640 });
    await page.goto("/tests/browser/app/?view=workspace&diagnostics=normal&theme=dark");
    await status(page).focus();
    await status(page).press("Enter");
    await expect(panel(page)).toContainText("Voice transcription");
    await expect(panel(page)).toContainText("Example transcription server");
    await expect(panel(page)).toContainText("Not checked yet");
    await panel(page).getByRole("button", { name: "Check connection", exact: true }).click();
    await expect(panel(page)).toContainText("Model list received");
    await expect(panel(page).getByText("HTTP 200", { exact: true })).not.toBeVisible();
    const technical = panel(page).locator("summary");
    await technical.focus();
    await technical.press("Enter");
    await expect(panel(page).getByText("HTTP 200", { exact: true })).toBeVisible();
    await technical.press("Enter");
    await page.screenshot({ path: info.outputPath(`connection-panel-${width}.png`) });
    await page.keyboard.press("Escape");
    await expect(status(page)).toBeFocused();
    await status(page).click();
    await panel(page).getByRole("button", { name: "Edit connection", exact: true }).click();
    await expect(panel(page)).toHaveCount(0);
    await expect(
      page.getByRole("status").filter({ hasText: "Edit voice-fixture for voice" }),
    ).toHaveCount(1);
    await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
    await status(page).click();
    await expect(panel(page)).toContainText("Example speech server");
    await expect(panel(page)).toContainText("speech.test");
    await expect(panel(page)).toContainText("Not checked yet");
    await panel(page).getByRole("button", { name: "Check connection", exact: true }).click();
    await expect(panel(page)).toContainText("Model list received");
    await page.keyboard.press("Escape");
    await page.getByRole("tab", { name: "Audio file", exact: true }).click();
    await status(page).click();
    await expect(panel(page)).toContainText("Audio-file transcription");
    await expect(panel(page)).toContainText("files.test");
    await expect(panel(page)).toContainText("Not checked yet");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
      true,
    );
  });
}

test("a failed check can be retried from the footer", async ({ page }, info) => {
  await page.goto("/tests/browser/app/?view=workspace&diagnostics=failure&theme=dark");
  await status(page).click();
  await panel(page).getByRole("button", { name: "Check connection", exact: true }).click();
  await expect(panel(page)).toContainText("Network failed");
  await expect(panel(page)).toContainText("could not be reached");
  await page.screenshot({ path: info.outputPath("connection-failure.png") });
  await panel(page).getByRole("button", { name: "Check again", exact: true }).click();
  await expect(panel(page)).toContainText("Model list received");
});

test("draft checks are not displayed as checks of the applied connection", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&diagnostics=stale");
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
  await status(page).click();
  await expect(panel(page)).toContainText("Settings changed since the last check");
  await expect(panel(page)).not.toContainText("unsaved-draft.test");
  await expect(panel(page).locator("summary")).toHaveCount(0);
  await panel(page).getByRole("button", { name: "Check again", exact: true }).click();
  await expect(panel(page)).toContainText("Model list received");
});

test("disabled speech and missing connections offer setup without probing", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&diagnostics=off");
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
  await status(page).click();
  await expect(panel(page)).toContainText("Text to speech is off");
  await expect(panel(page).getByRole("button", { name: /Check/ })).toHaveCount(0);
  await panel(page).getByRole("button", { name: "Speech settings", exact: true }).click();
  await expect(panel(page)).toHaveCount(0);
  await page.goto("/tests/browser/app/?view=workspace&diagnostics=missing");
  await status(page).click();
  await expect(panel(page)).toContainText("No connection selected");
  await expect(
    panel(page).getByRole("button", { name: "Check connection", exact: true }),
  ).toHaveCount(0);
  await panel(page).getByRole("button", { name: "Choose connection", exact: true }).click();
  await expect(
    page.getByRole("status").filter({ hasText: "Edit connections for voice" }),
  ).toHaveCount(1);
});
