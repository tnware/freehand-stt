import { test, expect } from "./fixtures";

for (const task of ["Voice transcription", "Audio file"]) {
  test(`${task} starts a stopped runtime from the model chooser`, async ({
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
    await panel
      .getByRole("button", { name: "Change engine or model", exact: true })
      .click();
    const chooser = page.getByRole("dialog", {
      name: "Engine and model",
      exact: true,
    });
    await expect(chooser).toContainText("CPU");
    await expect(chooser).toContainText("Stopped");
    await page.evaluate(() =>
      window.testRuntime.change("nemo-default", { backend: "cuda" }),
    );
    await expect(chooser).toContainText("NVIDIA GPU (CUDA)");
    await expect(
      chooser.getByText("Nemotron 3.5 Streaming", { exact: true }),
    ).toBeVisible();
    await chooser.getByRole("button", { name: "Start", exact: true }).click();
    await expect(
      chooser.getByRole("button", { name: "Stop", exact: true }),
    ).toBeEnabled();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      "Start:nemo-default",
    ]);
  });
}

test("managed Voice keeps runtime actions in the chooser and live mode in full controls", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  const panel = page.getByRole("complementary", {
    name: "Voice transcription settings",
  });
  await expect(
    panel.getByRole("switch", { name: "Live dictation", exact: true }),
  ).toHaveCount(0);
  await panel
    .getByRole("button", { name: "Change engine or model", exact: true })
    .click();
  const chooser = page.getByRole("dialog", {
    name: "Engine and model",
    exact: true,
  });
  await chooser.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(
    chooser.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  await chooser.getByRole("button", { name: "Start", exact: true }).click();
  await chooser
    .getByRole("button", { name: "Model options", exact: true })
    .click();
  await expect(chooser).toBeHidden();
  const inspector = page.locator('[data-pane="configuration"]');
  await expect(
    inspector.getByRole("switch", { name: "Realtime transcription" }),
  ).toBeChecked();
  await expect(
    inspector.getByRole("switch", { name: "Live overlay captions" }),
  ).toBeVisible();
  await expect(
    inspector.getByText("Spoken language", { exact: true }),
  ).toBeVisible();
  await expect(
    inspector.getByText("Shared vocabulary", { exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
    "Start:nemo-default",
  ]);
});
