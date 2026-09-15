import { test, expect } from "./fixtures";

for (const task of ["Voice transcription", "Audio file"]) {
  test(`${task} starts a stopped runtime from the workflow sidebar`, async ({
    page,
  }) => {
    await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
    await page.evaluate(() =>
      window.testRuntime.change("nemo-default", {
        state: "stopped",
        backend: "cpu",
      }),
    );
    await page.getByRole("button", { name: task, exact: true }).click();
    const panel = page.getByRole("complementary", { name: `${task} settings` });
    await expect(panel).toContainText("CPU");
    await expect(panel).toContainText("Stopped");
    await page.evaluate(() =>
      window.testRuntime.change("nemo-default", { backend: "cuda" }),
    );
    await expect(panel).toContainText("NVIDIA GPU (CUDA)");
    await expect(
      panel.getByRole("button", {
        name: "Model Nemotron 3.5 Streaming",
        exact: true,
      }),
    ).toBeVisible();
    await panel.getByRole("button", { name: "Start", exact: true }).click();
    await expect(
      panel.getByRole("button", { name: "Stop", exact: true }),
    ).toBeEnabled();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      "Start:nemo-default",
    ]);
  });
}

test("managed Voice exposes live mode and links to its full controls", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  const panel = page.getByRole("complementary", {
    name: "Voice transcription settings",
  });
  await expect(
    panel.getByRole("switch", { name: "Live dictation", exact: true }),
  ).toBeChecked();
  await panel.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(
    panel.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  await panel.getByRole("button", { name: "Start", exact: true }).click();
  await panel.getByRole("button", { name: /^Language/ }).click();
  await expect(
    page.getByRole("switch", { name: "Realtime transcription" }),
  ).toBeChecked();
  await expect(
    page.getByRole("switch", { name: "Live overlay captions" }),
  ).toBeVisible();
  await expect(
    page.getByText("Spoken language", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Shared vocabulary", { exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
    "Start:nemo-default",
  ]);
});
