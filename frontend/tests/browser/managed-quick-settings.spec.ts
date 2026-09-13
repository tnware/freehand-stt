import { test, expect } from "./fixtures";

test("managed speech keeps transcription controls and process controls together", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page
    .getByRole("button", { name: "Transcription settings", exact: true })
    .click();
  const panel = page.getByRole("dialog", { name: "Transcription settings" });
  await expect(
    panel.getByRole("button", { name: "Stop runtime", exact: true }),
  ).toBeVisible();
  await expect(panel.getByLabel("Model", { exact: true })).toBeVisible();
  await expect(
    panel.getByRole("switch", { name: "Realtime transcription" }),
  ).toBeChecked();
  await expect(
    panel.getByRole("switch", { name: "Live overlay captions" }),
  ).toBeVisible();
  await expect(
    panel.getByText("Spoken language", { exact: true }),
  ).toBeVisible();
  await expect(
    panel.getByText("Shared vocabulary", { exact: true }),
  ).toBeVisible();
  await panel
    .getByRole("button", { name: "Stop runtime", exact: true })
    .click();
  await expect(
    panel.getByRole("button", { name: "Start runtime", exact: true }),
  ).toBeVisible();
  await panel
    .getByRole("button", { name: "Start runtime", exact: true })
    .click();
  await expect(
    panel.getByRole("button", { name: "Stop runtime", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop",
    "Start",
  ]);
});
