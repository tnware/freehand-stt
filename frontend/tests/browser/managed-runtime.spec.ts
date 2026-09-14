import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

async function installRuntime(page: Page) {
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Download selected model", exact: true }),
  ).toBeVisible();
  return page.evaluate(() => {
    const call = window.testRuntime.calls.find((c) =>
      c.startsWith("SetInstance:"),
    );
    if (!call) throw new Error("Instance was not saved");
    return call.slice("SetInstance:".length);
  });
}

test("collapsed runtime controls can stop, download and restart without opening details", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="local-runtime"]').click();
  const id = await installRuntime(page);
  const toggle = page.getByRole("button", { name: "Manage", exact: true });
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await page.getByRole("button", { name: "Download", exact: true }).click();
  await expect(
    page.getByRole("progressbar", { name: "Runtime download progress" }),
  ).toBeInViewport();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await page.getByRole("button", { name: "Download", exact: true }).click();
  await page.evaluate((id) => window.testRuntime.finishDownload(id), id);
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
});

test("catalog downloads keep progress, cancellation and completion at the model", async ({
  page,
}) => {
  await page.setViewportSize({ width: 900, height: 640 });
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="local-runtime"]').click();
  const id = await installRuntime(page);
  const model = page.getByRole("article", {
    name: "Parakeet TDT v3",
    exact: true,
  });
  await model.scrollIntoViewIfNeeded();
  await model.getByRole("button", { name: "Download", exact: true }).click();
  const progress = model.getByRole("progressbar");
  await expect(progress).toBeInViewport();
  await page.evaluate(
    (id) =>
      window.testRuntime.change(id, {
        acquisition: {
          phase: "downloading",
          bytes: 500_000_000,
          totalBytes: 1_000_000_000,
        },
      }),
    id,
  );
  await expect(progress).toHaveAttribute("value", "50");
  const cancel = model.getByRole("button", { name: "Cancel", exact: true });
  await expect(cancel).toBeInViewport();
  await cancel.click();
  await expect(model.getByRole("status")).toContainText("cancelled");
  await model.getByRole("button", { name: "Download", exact: true }).click();
  await page.evaluate((id) => window.testRuntime.finishDownload(id), id);
  await expect(model.getByRole("status")).toContainText(
    "downloaded and verified",
  );
  await expect(model.getByRole("status")).toBeInViewport();
});

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
      page.getByRole("button", { name: "Install", exact: true }),
    ).toBeInViewport();
    const instanceID = await installRuntime(page);
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
    const progress = page.getByRole("progressbar", {
      name: "Runtime operation progress",
    });
    for (const bytes of [250_000_000, 750_000_000]) {
      await page.evaluate(
        ({ id, bytes }) =>
          window.testRuntime.change(id, {
            acquisition: {
              phase: "downloading",
              bytes,
              totalBytes: 1_000_000_000,
            },
          }),
        { id: instanceID, bytes },
      );
      await expect(progress).toHaveAttribute(
        "value",
        String(bytes / 10_000_000),
      );
    }
    await cancel.click();
    const setup = page.getByRole("region", {
      name: "Runtime setup",
      exact: true,
    });
    await expect(setup.getByText(/Operation cancelled\./)).toBeInViewport();
    await expect(download).toBeInViewport();
    await download.click();
    await page.evaluate(
      (id) =>
        window.testRuntime.change(id, {
          acquisition: {
            phase: "verifying",
            bytes: 1_000_000_000,
            totalBytes: 1_000_000_000,
          },
        }),
      instanceID,
    );
    await expect(progress).toHaveCount(0);
    await expect(page.getByText(/Model downloaded and verified\./)).toHaveCount(
      0,
    );
    await expect(
      page.getByRole("button", { name: "Start runtime", exact: true }),
    ).toHaveCount(0);
    await page.evaluate(
      (id) => window.testRuntime.finishDownload(id),
      instanceID,
    );
    await expect(
      setup.getByText(/Model downloaded and verified\./),
    ).toBeInViewport();
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
  await installRuntime(page);
  await expect(
    page.getByRole("region", { name: "Model catalog" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Refresh catalog", exact: true })
    .click();
  expect(
    await page.evaluate(() =>
      window.testRuntime.calls.some(
        (call) =>
          call.startsWith("DownloadModel:") || call.startsWith("Start:"),
      ),
    ),
  ).toBe(false);
});

test("unsaved drafts guard immediate runtime operations", async ({ page }) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.locator('[data-settings-section="audio"]').click();
  await page.locator("#max-duration").fill("90");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page.getByRole("button", { name: "Manage", exact: true }).click();
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
    page.getByText("Unavailable on this platform", {
      exact: false,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "Install",
      exact: true,
    }),
  ).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
