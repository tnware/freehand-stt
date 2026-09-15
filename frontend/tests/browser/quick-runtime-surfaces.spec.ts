import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { installOutputFixture } from "./runtime-output-fixtures";

const instanceID = "llama-cpp-default";
const sidebar = (page: Page) => page.locator("#workbench-primary-sidebar");
const cleanupRuntime = (page: Page) =>
  sidebar(page).getByRole("group", {
    name: "Local runtime Local cleanup",
    exact: true,
  });
const cleanupToggle = (page: Page) =>
  sidebar(page).getByRole("switch", {
    name: "Post-process transcripts",
    exact: true,
  });

async function revealSidebar(page: Page) {
  if (!(await sidebar(page).isVisible()))
    await page
      .getByRole("button", { name: "Toggle primary sidebar", exact: true })
      .click();
}

async function openCleanup(page: Page, width: number, extra = "") {
  await page.setViewportSize({ width, height: 820 });
  await page.goto(
    `/tests/browser/app/?main&workflows&setup-ready&managed-cleanup${extra}`,
  );
  await revealSidebar(page);
  await expect(cleanupRuntime(page)).toBeVisible();
}

for (const width of [1280, 560]) {
  test(`a disabled Cleanup keeps its local runtime, lifecycle feedback, and output together at ${width}px`, async ({
    page,
  }) => {
    await installOutputFixture(page);
    await page.emulateMedia({ colorScheme: "dark" });
    await openCleanup(page, width, "&theme=dark");
    await expect(page.locator("html")).toHaveClass(/dark/);
    const local = cleanupRuntime(page);
    const status = local.getByRole("status", {
      name: "Local runtime status",
      exact: true,
    });
    await expect(local).toHaveCount(1);
    await expect(
      sidebar(page).getByRole("status", {
        name: "Local runtime status",
        exact: true,
      }),
    ).toHaveCount(1);
    await expect(cleanupToggle(page)).not.toBeChecked();
    await expect(
      sidebar(page).locator('input[id$="-voice-model"]'),
    ).toBeVisible();
    await expect(
      local.getByRole("button", { name: "Selected model", exact: true }),
    ).toContainText("S1-mini");
    await expect(status).toContainText("Running");
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
    expect(await page.evaluate(() => window.testOutput.calls)).toEqual([]);
    const selections = await page.evaluate(
      () => window.testConnectionWindows.settings().savedConnections.selected,
    );
    expect(selections).toMatchObject({
      voice: "original",
      stt: "original",
      cleanup: "local-cleanup",
    });
    if (width === 1280)
      await page.screenshot({
        path: test.info().outputPath("managed-cleanup-sidebar-dark.png"),
        fullPage: true,
      });

    await page.evaluate(() => window.testRuntime.holdLifecycle());
    await local.getByRole("button", { name: "Stop", exact: true }).click();
    await expect(status).toContainText("Stopping runtime");
    await expect(
      local.getByRole("button", { name: "Start", exact: true }),
    ).toHaveCount(0);
    await page.evaluate(
      (id) => window.testRuntime.acknowledgeLifecycle(id),
      instanceID,
    );
    await expect(status).toContainText("Stopping runtime");
    await page.evaluate(
      (id) => window.testRuntime.finishLifecycle(id),
      instanceID,
    );
    await expect(status).toContainText("Stopped");
    await local.getByRole("button", { name: "Start", exact: true }).click();
    await expect(status).toContainText("Starting runtime");
    await page.evaluate((id) => {
      window.testRuntime.acknowledgeLifecycle(id);
      window.testRuntime.change(id, {
        startupProgress: { phase: "warming_up", startedAt: Date.now() - 3000 },
      });
      window.testRuntime.failNextCancel();
    }, instanceID);
    await expect(local).toContainText(
      /Warming up selected model · \d+s in this stage/,
    );
    await local.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(local.getByRole("alert")).toContainText(
      "The runtime operation did not finish.",
    );
    await expect(
      local.getByRole("button", { name: "Cancel", exact: true }),
    ).toBeEnabled();
    await local.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(local).toContainText("Operation cancelled.");
    await expect(local.getByRole("alert")).toHaveCount(0);
    await expect(
      local.getByRole("button", { name: "Start", exact: true }),
    ).toBeEnabled();
    await expect(cleanupToggle(page)).not.toBeChecked();

    await local
      .getByRole("button", { name: "View output", exact: true })
      .click();
    // Close the compact drawer to inspect the shared bottom panel.
    if (width < 700 && (await sidebar(page).isVisible()))
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();
    await expect(
      page
        .getByRole("tablist", { name: "Output runtimes", exact: true })
        .getByRole("tab", { name: "Local cleanup", exact: true }),
    ).toHaveAttribute("aria-selected", "true");
    await expect(
      page.getByRole("region", {
        name: "Read-only process output",
        exact: true,
      }),
    ).toContainText(instanceID);
    if (width === 560)
      await page.screenshot({
        path: test.info().outputPath("managed-cleanup-output-compact.png"),
        fullPage: true,
      });
    await page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: "Audio file", exact: true })
      .click();
    await revealSidebar(page);
    await expect(
      sidebar(page).locator('input[id$="-quick-stt-model"]'),
    ).toBeVisible();
    await expect(local).toHaveCount(1);
    await expect(local).toContainText("Operation cancelled.");
    await expect(cleanupToggle(page)).not.toBeChecked();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      `Stop:${instanceID}`,
      `Start:${instanceID}`,
      `Cancel:${instanceID}`,
      `Cancel:${instanceID}`,
    ]);
    expect(
      await page.evaluate(
        () => window.testConnectionWindows.settings().savedConnections.selected,
      ),
    ).toEqual(selections);
    expect(
      await sidebar(page).evaluate(
        (element) => element.scrollWidth <= element.clientWidth,
      ),
    ).toBe(true);
  });
}

test("managed cleanup respects a dirty workflow inspector without hiding its runtime state", async ({
  page,
}) => {
  await openCleanup(page, 1280);
  const local = cleanupRuntime(page);
  await page.evaluate(
    (id) => window.testRuntime.change(id, { state: "stopped" }),
    instanceID,
  );
  await expect(
    local.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  await expect(
    local.getByRole("button", { name: "Selected model", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  const inspector = page.locator('[data-pane="configuration"]');
  await inspector
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  const duration = inspector.locator("#max-duration");
  const original = await duration.inputValue();
  await duration.fill(original === "140" ? "141" : "140");
  await expect(
    local.getByRole("button", { name: "Start", exact: true }),
  ).toBeDisabled();
  await expect(
    local.getByRole("button", { name: "Selected model", exact: true }),
  ).toBeDisabled();
  await expect(
    local.getByRole("status", { name: "Local runtime status", exact: true }),
  ).toContainText("Stopped");
  await inspector
    .getByRole("button", { name: "Discard changes", exact: true })
    .click();
  await expect(duration).toHaveValue(original);
  await expect(
    local.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("Manage runtime opens the selected cleanup provider instead of another running provider", async ({
  page,
}) => {
  await openCleanup(page, 1280);
  await page.evaluate((id) => {
    window.testRuntime.addSecondProvider();
    window.testRuntime.change(id, { state: "stopped" });
  }, instanceID);
  const navigation = page.getByRole("navigation", {
    name: "Workspace",
    exact: true,
  });
  await navigation
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Refresh inventory", exact: true })
    .click();
  const inventory = page.getByRole("group", {
    name: "Runtime inventory",
    exact: true,
  });
  await inventory
    .getByRole("button", { name: /Second speech provider/ })
    .click();
  await expect(
    page.getByRole("heading", { name: "Second speech provider", exact: true }),
  ).toBeVisible();
  await navigation
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await cleanupRuntime(page)
    .getByRole("button", { name: "Manage runtime", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "llama.cpp", exact: true }),
  ).toBeVisible();
  await expect(
    inventory.getByRole("button", { name: /llama.cpp/ }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(
    inventory.getByRole("button", { name: /Second speech provider/ }),
  ).toHaveAttribute("aria-pressed", "false");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("active dictation locks managed cleanup process and model changes while preserving Stop recording", async ({
  page,
}) => {
  await openCleanup(page, 1280, "&work-busy");
  const local = cleanupRuntime(page);
  await expect(
    local.getByRole("button", { name: "Stop", exact: true }),
  ).toBeDisabled();
  await expect(
    local.getByRole("button", { name: "Selected model", exact: true }),
  ).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Stop recording", exact: true }),
  ).toBeEnabled();
  await expect(cleanupToggle(page)).not.toBeChecked();
  await expect(
    local.getByRole("status", { name: "Local runtime status", exact: true }),
  ).toContainText("Running");
  await page.evaluate(
    (id) => window.testRuntime.change(id, { state: "stopped" }),
    instanceID,
  );
  await expect(
    local.getByRole("button", { name: "Start", exact: true }),
  ).toBeDisabled();
  await expect(
    local.getByRole("button", { name: "Selected model", exact: true }),
  ).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("Voice and files each show one runtime group beside their managed connection", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  for (const task of ["Voice transcription", "Audio file"]) {
    await page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: task, exact: true })
      .click();
    const local = sidebar(page).getByRole("group", {
      name: "Local runtime Local speech",
      exact: true,
    });
    await expect(local).toHaveCount(1);
    await expect(
      local.getByRole("button", { name: "Selected model", exact: true }),
    ).toContainText("Nemotron 3.5 Streaming");
    await expect(
      local.getByRole("button", { name: "Stop", exact: true }),
    ).toBeEnabled();
    await expect(
      sidebar(page).getByRole("status", {
        name: "Local runtime status",
        exact: true,
      }),
    ).toHaveCount(1);
  }
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
