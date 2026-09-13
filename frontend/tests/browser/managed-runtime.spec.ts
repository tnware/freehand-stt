import { test, expect } from "./fixtures";

for (const viewport of [
  { width: 1280, height: 720 },
  { width: 900, height: 640 },
]) {
  test(`setup advances in place without scrolling at ${viewport.width}x${viewport.height}`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await page.goto("/tests/browser/app/?runtime");
    await page.locator('[data-settings-section="local-runtime"]').click();
    const enable = page.getByRole("button", {
      name: "Enable local transcription",
      exact: true,
    });
    await expect(enable).toBeInViewport();
    await enable.click();
    const install = page.getByRole("button", {
      name: "Install runtime",
      exact: true,
    });
    await expect(install).toBeInViewport();
    await install.click();
    const download = page.getByRole("button", {
      name: "Download selected model",
      exact: true,
    });
    await expect(download).toBeInViewport();
    await expect(
      page.getByRole("button", { name: "Start runtime", exact: true }),
    ).toHaveCount(0);
    await download.click();
    await expect(
      page.getByRole("progressbar", { name: "Runtime operation" }),
    ).toBeInViewport();
    const cancel = page.getByRole("button", {
      name: "Cancel operation",
      exact: true,
    });
    await expect(cancel).toBeInViewport();
    await cancel.click();
    await expect(download).toBeInViewport();
    await download.click();
    await page.evaluate(() => window.testRuntime.finishDownload());
    expect(await page.evaluate(() => window.testRuntime.calls)).not.toContain(
      "Start",
    );
    const start = page.getByRole("button", {
      name: "Start runtime",
      exact: true,
    });
    await expect(start).toBeInViewport();
    await start.click();
    await expect(
      page.getByRole("button", { name: "Stop runtime", exact: true }),
    ).toBeInViewport();
  });
}

test("local runtime is discoverable and browsing never downloads", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await expect(
    page.getByRole("heading", { name: "Live transcription on this PC" }),
  ).toBeVisible();
  await expect(page.getByText("Nemotron 3.5", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "Enable local transcription",
      exact: true,
    }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  await page
    .getByRole("button", { name: "Refresh catalog", exact: true })
    .click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "RefreshCatalog",
  ]);
});

test("enable, install, download, cancel, retry, use, stop and switch back are explicit", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page
    .getByRole("button", { name: "Enable local transcription", exact: true })
    .click();
  await expect(page.getByText(/No automatic fallback/).first()).toBeVisible();
  await page
    .getByRole("button", { name: "Install runtime", exact: true })
    .click();
  const model = page.getByRole("article", { name: "Nemotron 3.5" });
  await model.getByRole("button", { name: "Download", exact: true }).click();
  await expect(
    page.getByRole("progressbar", { name: "Runtime operation" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Cancel operation", exact: true })
    .click();
  await model.getByRole("button", { name: "Download", exact: true }).click();
  await page.evaluate(() => window.testRuntime.finishDownload());
  await page
    .getByRole("button", { name: "Start runtime", exact: true })
    .click();
  await expect(
    page.getByText("Running", { exact: true }).first(),
  ).toBeVisible();
  await page.getByRole("button", { name: "Stop runtime", exact: true }).click();
  await expect(
    page.getByText("Stopped", { exact: true }).first(),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Start runtime", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Use my own server", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toContainText(
    "saved Voice connection",
  );
  await page
    .getByRole("button", { name: "Disable local runtime", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Enable local transcription",
      exact: true,
    }),
  ).toBeVisible();
});

test("unsaved drafts guard immediate runtime operations", async ({ page }) => {
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="audio"]').click();
  await page.locator("#max-duration").fill("90");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page
    .getByRole("button", { name: "Enable local transcription", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toContainText(
    "Save settings before continuing?",
  );
  await page.getByRole("button", { name: "Keep editing", exact: true }).click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  await page.locator('[data-settings-section="audio"]').click();
  await expect(page.locator("#max-duration")).toHaveValue("90");
});

test("unsupported hosts never offer an enabled Windows runtime", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&platform=darwin");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await expect(
    page.getByText("Local runtime is available on Windows only."),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "Enable local transcription",
      exact: true,
    }),
  ).toHaveCount(0);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
