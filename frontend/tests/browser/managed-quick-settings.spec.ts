import { test, expect } from "./fixtures";

for (const task of ["Voice", "Audio file"]) {
  test(`${task} can start a stopped local runtime from its quick settings`, async ({
    page,
  }) => {
    await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
    await page.evaluate(() =>
      window.testRuntime.change("nemo-default", { state: "stopped" }),
    );
    await page.getByRole("tab", { name: task, exact: true }).click();
    await page
      .getByRole("button", { name: "Transcription settings", exact: true })
      .click();
    const panel = page.getByRole("dialog", { name: "Transcription settings" });
    await expect(
      panel.getByLabel("Selected model", { exact: true }),
    ).toHaveText("Nemotron 3.5 Streaming");
    const connection = await panel
      .getByRole("combobox", { name: "Choose connection", exact: true })
      .boundingBox();
    const model = await panel
      .getByLabel("Selected model", { exact: true })
      .boundingBox();
    expect(connection).not.toBeNull();
    expect(model).not.toBeNull();
    expect(connection!.y + connection!.height).toBeLessThan(model!.y);
    await panel.getByRole("button", { name: "Start", exact: true }).click();
    await expect(
      panel.getByRole("button", { name: "Stop", exact: true }),
    ).toBeEnabled();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      "Start:nemo-default",
    ]);
  });
}

test("managed speech keeps transcription controls and process controls together", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page
    .getByRole("button", { name: "Transcription settings", exact: true })
    .click();
  const panel = page.getByRole("dialog", { name: "Transcription settings" });
  await expect(
    panel.getByRole("button", { name: "Stop", exact: true }),
  ).toBeVisible();
  await expect(
    panel.getByLabel("Selected model", { exact: true }),
  ).toBeVisible();
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
  await panel.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(
    panel.getByRole("button", { name: "Start", exact: true }),
  ).toBeVisible();
  await panel.getByRole("button", { name: "Start", exact: true }).click();
  await expect(
    panel.getByRole("button", { name: "Stop", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
    "Start:nemo-default",
  ]);
});
