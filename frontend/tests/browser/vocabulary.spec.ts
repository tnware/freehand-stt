import { test, expect } from "./fixtures";

for (const width of [520, 1000]) {
  test(`vocabulary keeps the editor and workflow switches together at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await page.goto("/tests/browser/app/?view=vocabulary&theme=dark");
    const terms = page.getByRole("textbox", { name: "Names and phrases", exact: true });
    await expect(page.getByText("2 unique phrases", { exact: true })).toBeVisible();
    await expect(terms).toBeInViewport();
    await expect(
      page.getByRole("switch", { name: "Voice transcription", exact: true }),
    ).toBeInViewport();
    await expect(
      page.getByRole("switch", { name: "Audio-file transcription", exact: true }),
    ).toBeInViewport();
    await expect(page.getByRole("spinbutton", { name: "Vocabulary strength" })).not.toBeVisible();
    await page.screenshot({ path: info.outputPath(`vocabulary-${width}.png`) });
    const support = page.getByRole("button", {
      name: "Voice transcription vocabulary support",
      exact: true,
    });
    await support.focus();
    await support.press("Enter");
    const help = page.getByRole("dialog", {
      name: "Voice transcription vocabulary support",
      exact: true,
    });
    await expect(help).toBeInViewport();
    await expect(help).toContainText("nvidia/nemotron-3.5-asr-streaming-0.6b");
    await expect(help).toContainText("Vocabulary boosting");
    await page.keyboard.press("Escape");
    await expect(support).toBeFocused();

    const tuning = page.locator("summary").filter({ hasText: "Vocabulary tuning" });
    await tuning.focus();
    await tuning.press("Enter");
    const strength = page.getByRole("spinbutton", { name: "Vocabulary strength" });
    await strength.fill("2.5");
    await tuning.click();
    await tuning.click();
    await expect(strength).toHaveValue("2.5");
    await page.getByRole("switch", { name: "Audio-file transcription", exact: true }).click();
    await expect(terms).toHaveValue("Freehand\nOpenAI");
    await expect(
      page.getByRole("switch", { name: "Voice transcription", exact: true }),
    ).toBeChecked();
    await expect(
      page.getByRole("switch", { name: "Audio-file transcription", exact: true }),
    ).toBeChecked();
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true);
  });
}

test("line feedback stays compact, opens by keyboard, and selects the original phrase", async ({
  page,
}, info) => {
  await page.goto("/tests/browser/app/?view=vocabulary&theme=dark&scenario=review");
  const feedback = page.locator('details[aria-label="Vocabulary line feedback"]');
  const summary = feedback.locator("summary");
  await expect(summary).toContainText("2 lines to review");
  await expect(summary).toContainText("1 duplicate");
  await expect(feedback.getByRole("button", { name: "Line 3", exact: true })).not.toBeVisible();
  await expect(
    page.getByText("Review vocabulary for this selection", { exact: true }),
  ).toBeVisible();
  await summary.focus();
  await summary.press("Enter");
  await page.screenshot({ path: info.outputPath("vocabulary-feedback.png") });
  await feedback.getByRole("button", { name: "Line 3", exact: true }).click();
  const terms = page.getByRole("textbox", { name: "Names and phrases", exact: true });
  await expect(terms).toBeFocused();
  expect(
    await terms.evaluate((element: HTMLTextAreaElement) =>
      element.value.slice(element.selectionStart, element.selectionEnd),
    ),
  ).toBe("A phrase to review");
  await terms.fill("Freehand\nOpenAI");
  await expect(feedback).toHaveCount(0);
  await expect(page.getByText("Hints enabled", { exact: true })).toBeVisible();
});

test("unsupported workflows retain their choice and explain it without duplicating paragraphs", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=vocabulary&scenario=unsupported");
  await expect(page.getByText("Unavailable with this selection", { exact: true })).toBeVisible();
  const files = page.getByRole("switch", { name: "Audio-file transcription", exact: true });
  await files.click();
  await expect(files).toBeChecked();
  await page
    .getByRole("button", { name: "Audio-file transcription vocabulary support", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toContainText("Your preference is kept");
  await expect(page.getByRole("dialog")).toContainText("do not support vocabulary hints");
});

test("a failed preview can be retried without leaving the draft", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=vocabulary&scenario=failure");
  await expect(page.getByRole("alert")).toHaveText("Could not check vocabulary support.");
  await page.getByRole("button", { name: "Try again", exact: true }).click();
  await expect(page.getByText("2 unique phrases", { exact: true })).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByRole("textbox", { name: "Names and phrases", exact: true })).toHaveValue(
    "Freehand\nOpenAI",
  );
});

test("newer draft feedback wins over a slow previous preview and shows byte overflow", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=vocabulary");
  const terms = page.getByRole("textbox", { name: "Names and phrases", exact: true });
  await expect(page.getByText("2 unique phrases", { exact: true })).toBeVisible();
  await terms.fill("Old draft");
  await page.waitForTimeout(200);
  await terms.fill("One\nTwo\nThree");
  await expect(page.getByText("3 unique phrases", { exact: true })).toBeVisible();
  await page.waitForTimeout(800);
  await expect(page.getByText("3 unique phrases", { exact: true })).toBeVisible();
  await terms.fill("語".repeat(5500));
  await expect(terms).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("alert")).toHaveText("Shorten the list to 16,384 bytes to save.");
});
