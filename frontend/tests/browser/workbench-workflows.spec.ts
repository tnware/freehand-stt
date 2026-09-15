import { test, expect } from "./fixtures";

test("voice capture retains record, stop and cancel controls", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=workspace&history=off");
  const capture = page.getByRole("region", { name: "Voice capture" });
  await capture.getByRole("button", { name: "Start recording" }).click();
  await expect(
    capture.getByRole("button", { name: "Stop recording" }),
  ).toBeEnabled();
  await expect(
    capture.getByRole("button", { name: "Cancel", exact: true }),
  ).toBeEnabled();
  await capture.getByRole("button", { name: "Stop recording" }).click();
  await expect(capture).toContainText("Transcribing audio");
  await capture.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(
    capture.getByRole("button", { name: "Start recording" }),
  ).toBeEnabled();
});

test("generic speech models do not advertise realtime", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace");
  await expect(
    page.getByRole("switch", { name: "Live dictation" }),
  ).toHaveCount(0);
});

test("bottom panel really collapses and its tabs support keyboard navigation", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=workspace");
  const transcript = page.getByRole("region", { name: "Current result" });
  const initial = await transcript.boundingBox();
  await page.getByRole("button", { name: "Hide panel", exact: true }).click();
  await expect(
    page.getByRole("region", { name: "Transcript history" }),
  ).toBeHidden();
  await expect
    .poll(async () => (await transcript.boundingBox())!.height)
    .toBeGreaterThan(initial!.height + 60);
  await page.getByRole("button", { name: "Show panel", exact: true }).click();
  const recent = page.getByRole("tab", { name: /^Recent/ });
  await recent.focus();
  await page.keyboard.press("End");
  await expect(
    page.getByRole("tab", { name: "Diagnostics", exact: true }),
  ).toBeFocused();
  await expect(
    page.getByRole("tab", { name: "Diagnostics", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  await page.keyboard.press("Home");
  await expect(recent).toBeFocused();
});

test("short windows retain output and diagnostic panels", async ({ page }) => {
  await page.setViewportSize({ width: 900, height: 520 });
  await page.goto("/tests/browser/app/?view=workspace");
  await page.getByRole("tab", { name: "Diagnostics", exact: true }).click();
  await expect(
    page.getByRole("tabpanel", { name: "Diagnostics" }),
  ).toBeVisible();
  await page.getByRole("tab", { name: "Runtime output", exact: true }).click();
  await expect(
    page.getByRole("tabpanel", { name: "Runtime output" }),
  ).toBeVisible();
});

test("compact task settings remain reachable without squeezing the transcript", async ({
  page,
}) => {
  await page.setViewportSize({ width: 520, height: 740 });
  await page.goto("/tests/browser/app/?view=workspace");
  await expect(
    page.getByRole("complementary", { name: "Voice transcription settings" }),
  ).toBeHidden();
  await page
    .getByRole("button", { name: "Task settings", exact: true })
    .click();
  await expect(
    page.getByRole("complementary", { name: "Voice transcription settings" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Close task settings", exact: true })
    .click();
  const width = await page
    .getByRole("region", { name: "Current result" })
    .evaluate((node) => node.getBoundingClientRect().width);
  expect(width).toBeGreaterThan(400);
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});
