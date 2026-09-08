import { test, expect } from "./fixtures";

for (const width of [560, 1000]) {
  test(`speech failures stay local and retry keeps the draft at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 620 });
    await page.goto("/tests/browser/app/?view=workspace&feedback&theme=dark");
    await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
    const composer = page.getByRole("textbox", { name: "Text to speak", exact: true });
    await composer.fill("Keep this draft after a failed request.");
    const before = await composer.boundingBox();
    await page.getByRole("button", { name: "Speak", exact: true }).click();
    await expect(page.getByRole("complementary", { name: "Notifications" })).toHaveCount(0);
    await expect(page.getByRole("alert")).toHaveCount(1);
    expect(await composer.boundingBox()).toEqual(before);
    const details = page.getByRole("button", { name: "Speech error details", exact: true });
    await details.focus();
    await details.press("Enter");
    const dialog = page.getByRole("dialog", { name: "Speech error details", exact: true });
    await expect(dialog).toContainText("selected voice");
    await page.screenshot({ path: info.outputPath(`speech-error-${width}.png`) });
    await page.keyboard.press("Escape");
    await expect(details).toBeFocused();
    await details.click();
    await dialog.getByRole("button", { name: "Speech settings", exact: true }).click();
    await expect(page.locator("footer")).toContainText("Speech settings");
    await expect(dialog).toHaveCount(0);
    await page.getByRole("tab", { name: "Voice", exact: true }).click();
    await expect(page.getByRole("complementary", { name: "Notifications" })).toBeVisible();
    await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
    await expect(page.getByRole("complementary", { name: "Notifications" })).toHaveCount(0);
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await expect(composer).toHaveValue("Keep this draft after a failed request.");
    await expect(page.getByRole("alert")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Save generated speech" })).toBeVisible();
    await page.getByRole("button", { name: "Save generated speech" }).click();
    await expect(page.getByRole("complementary", { name: "Notifications" })).toBeVisible();
    expect(await composer.boundingBox()).toEqual(before);
    await page.screenshot({ path: info.outputPath(`save-error-${width}.png`) });
    await page.getByRole("button", { name: "Error details", exact: true }).click();
    await expect(page.getByRole("dialog", { name: "Error details", exact: true })).toContainText(
      "Choose another folder",
    );
    await page.keyboard.press("Escape");
    await page.getByRole("button", { name: "Dismiss this message" }).click();
    await page.getByRole("button", { name: "Save generated speech" }).click();
    await expect(
      page.getByRole("status").filter({ hasText: "Generated speech saved" }),
    ).toBeVisible();
    expect(await composer.boundingBox()).toEqual(before);
    await page
      .getByRole("button", { name: "Clear generated speech from memory" })
      .click({ timeout: 1000 });
    await expect(page.getByRole("complementary", { name: "Notifications" })).toHaveCount(0);
    await expect(composer).toHaveValue("Keep this draft after a failed request.");
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
      true,
    );
  });
}

test("file errors keep details and settings beside the existing retry", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&file-error&theme=dark");
  await page.getByRole("tab", { name: "Audio file", exact: true }).click();
  const retry = page.getByRole("button", { name: "Retry", exact: true });
  const before = await retry.boundingBox();
  await page.getByRole("button", { name: "File transcription error details", exact: true }).click();
  const details = page.getByRole("dialog", {
    name: "File transcription error details",
    exact: true,
  });
  await details.getByRole("button", { name: "Transcription settings", exact: true }).click();
  await expect(page.locator("footer")).toContainText("File transcription settings");
  expect(await retry.boundingBox()).toEqual(before);
  await retry.click();
  await expect(page.getByRole("button", { name: "Cancel", exact: true })).toBeVisible();
});

test("a settings save failure has one owner while the decision dialog is open", async ({
  page,
  saves,
}) => {
  await page.setViewportSize({ width: 860, height: 660 });
  await page.locator('[data-settings-section="audio"]').click();
  await page.locator("#max-duration").fill("90");
  await page.locator('[data-settings-section="connections"]').click();
  const decision = page.getByRole("dialog", {
    name: "Save settings before changing connections?",
    exact: true,
  });
  await decision.getByRole("button", { name: "Save and continue", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "failure");
  await expect(decision.getByRole("alert")).toContainText("Fixture save failed");
  await expect(page.getByRole("complementary", { name: "Notifications" })).toHaveCount(0);
  await decision.getByRole("button", { name: "Keep editing", exact: true }).click();
  await expect(page.getByRole("complementary", { name: "Notifications" })).toBeVisible();
  const save = page.getByRole("button", { name: "Save settings", exact: true });
  const before = await save.boundingBox();
  await page.getByRole("button", { name: "Dismiss this message" }).click();
  expect(await save.boundingBox()).toEqual(before);
});

test("long dictation errors can be read by keyboard without enlarging the transport", async ({
  page,
}, info) => {
  await page.setViewportSize({ width: 860, height: 660 });
  await page.goto("/tests/browser/app/?view=workspace&feedback=voice&theme=dark");
  const record = page.getByRole("button", { name: "Record again", exact: true });
  const before = await record.boundingBox();
  await expect(page.getByText(/The transcription request timed out\./)).toHaveCount(1);
  await expect(page.getByText("No transcript to show", { exact: true })).toBeVisible();
  const details = page.getByRole("button", { name: "Dictation error details", exact: true });
  await details.focus();
  await details.press("Enter");
  await expect(
    page.getByRole("dialog", { name: "Dictation error details", exact: true }),
  ).toContainText("recording again");
  await page.screenshot({ path: info.outputPath("dictation-error.png") });
  await page.keyboard.press("Escape");
  await expect(details).toBeFocused();
  expect(await record.boundingBox()).toEqual(before);
});
