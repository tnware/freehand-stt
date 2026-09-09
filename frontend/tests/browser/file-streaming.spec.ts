import { test, expect } from "./fixtures";

test.beforeEach(async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&theme=dark&file-streaming");
  await page.getByRole("tab", { name: "Audio file", exact: true }).click();
  await page.getByRole("button", { name: "Streaming supported", exact: true }).click();
});

test("the response preference survives tab changes and unsupported connections", async ({
  page,
}) => {
  const toggle = page.locator("#file-stream-toggle");
  await expect(toggle).toBeChecked();
  await toggle.click();
  await page.getByRole("tab", { name: "Voice", exact: true }).click();
  await page.getByRole("tab", { name: "Audio file", exact: true }).click();
  await expect(toggle).not.toBeChecked();
  await page.getByRole("button", { name: "Streaming rejected", exact: true }).click();
  await expect(toggle).toBeDisabled();
  await page.getByRole("button", { name: "Streaming supported", exact: true }).click();
  await expect(toggle).not.toBeChecked();
  await toggle.click();
  await page.getByRole("button", { name: "Completed-only profile", exact: true }).click();
  await expect(toggle).toBeDisabled();
  await expect(toggle).not.toBeChecked();
  await expect(page.getByRole("button", { name: "Try streaming", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Streaming supported", exact: true }).click();
  await expect(toggle).toBeChecked();
  await expect(page.getByText("File requests: 0; streaming: false", { exact: true })).toBeVisible();
});

test("Try streaming selects text updates for the next explicit request", async ({ page }) => {
  const toggle = page.locator("#file-stream-toggle");
  await toggle.click();
  await page.getByRole("button", { name: "Streaming rejected", exact: true }).click();
  await page.getByRole("button", { name: "Try streaming", exact: true }).click();
  await expect(toggle).toBeEnabled();
  await expect(toggle).toBeChecked();
  await expect(page.getByText("File requests: 0; streaming: false", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Transcribe", exact: true }).click();
  await expect(page.getByText("File requests: 1; streaming: true", { exact: true })).toBeVisible();
});

test("an unsupported connection submits completed mode without losing the preference", async ({
  page,
}) => {
  await page.getByRole("button", { name: "Streaming rejected", exact: true }).click();
  await page.getByRole("button", { name: "Transcribe", exact: true }).click();
  await expect(page.getByText("File requests: 1; streaming: false", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Streaming supported", exact: true }).click();
  await expect(page.locator("#file-stream-toggle")).toBeChecked();
});

test("file start responds immediately, blocks conflicting actions, and recovers after rejection", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=workspace&theme=dark&file-streaming&file-actions");
  await page.getByRole("tab", { name: "Audio file", exact: true }).click();
  await page.getByRole("button", { name: "Streaming supported", exact: true }).click();
  await page.getByRole("button", { name: "Transcribe", exact: true }).click();
  await expect(page.getByRole("button", { name: "Starting…", exact: true })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Change audio file", exact: true })).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Clear selected audio file", exact: true }),
  ).toBeDisabled();
  await expect(page.locator("#file-stream-toggle")).toBeDisabled();
  await page.getByRole("button", { name: "Reject file start", exact: true }).click();
  await expect(page.getByRole("button", { name: "Transcribe", exact: true })).toBeEnabled();
  await expect(page.getByRole("button", { name: "Change audio file", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "Transcribe", exact: true }).click();
  await page.getByRole("button", { name: "Admit file start", exact: true }).click();
  await expect(page.getByRole("button", { name: "Cancel", exact: true })).toBeEnabled();
  await expect(page.getByText("File requests: 2; streaming: true", { exact: true })).toBeVisible();
});
