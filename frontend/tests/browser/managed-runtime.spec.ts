import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

async function addRuntime(page: Page) {
  await page.getByRole("button", { name: "Add runtime", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Add runtime" });
  await dialog.getByLabel("Runtime name").fill("Local speech");
  await dialog
    .getByRole("button", { name: "Add runtime", exact: true })
    .click();
  await expect(dialog).toBeHidden();
  return page.evaluate(() => {
    const call = window.testRuntime.calls.find((c) =>
      c.startsWith("SetInstance:"),
    );
    if (!call) throw new Error("Instance was not saved");
    return call.slice("SetInstance:".length);
  });
}

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
    await expect(
      page.getByRole("button", { name: "Add runtime", exact: true }),
    ).toBeInViewport();
    const instanceID = await addRuntime(page);
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
      page.getByRole("progressbar", { name: "Runtime operation progress" }),
    ).toBeInViewport();
    const cancel = page.getByRole("button", {
      name: "Cancel operation",
      exact: true,
    });
    await expect(cancel).toBeInViewport();
    await cancel.click();
    await expect(download).toBeInViewport();
    await download.click();
    await page.evaluate(
      (id) => window.testRuntime.finishDownload(id),
      instanceID,
    );
    expect(await page.evaluate(() => window.testRuntime.calls)).not.toContain(
      `Start:${instanceID}`,
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
    page.getByRole("heading", { name: "Managed runtimes", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  const instanceID = await addRuntime(page);
  await expect(
    page.getByRole("region", { name: "Model catalog" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Refresh catalog", exact: true })
    .click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    `SetInstance:${instanceID}`,
    `RefreshCatalog:${instanceID}`,
  ]);
});

test("unsaved drafts guard immediate runtime operations", async ({ page }) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.locator('[data-settings-section="audio"]').click();
  await page.locator("#max-duration").fill("90");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page.getByRole("button", { name: "Stop runtime", exact: true }).click();
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
    page.getByText("No managed provider is available on this platform.", {
      exact: false,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "Add runtime",
      exact: true,
    }),
  ).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
