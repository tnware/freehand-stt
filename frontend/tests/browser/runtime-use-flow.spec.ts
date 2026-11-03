import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

const rail = (page: Page) =>
  page.getByRole("navigation", { name: "Workspace", exact: true });
const runtimePage = (page: Page) =>
  rail(page).getByRole("button", { name: "Local runtime", exact: true });
const snapshot = (page: Page) =>
  page.evaluate(() => window.testConnectionWindows.settings());

async function openFreshRuntime(
  page: Page,
  workflow: "voice" | "file" = "voice",
) {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto(
    "/tests/browser/app/?main&setup-ready&workflows&runtime&runtime-use-flow",
  );
  if (workflow === "file")
    await rail(page)
      .getByRole("button", { name: "Audio file", exact: true })
      .click();
  const before = await snapshot(page);
  await runtimePage(page).click();
  return before;
}

async function installAndDownload(page: Page) {
  await page.getByRole("button", { name: "Install", exact: true }).click();
  const review = page.getByRole("dialog", {
    name: "Install NeMo-Speech.cpp?",
    exact: true,
  });
  await review
    .getByRole("button", { name: "Download and install", exact: true })
    .click();
  const id = await page.evaluate(
    () => window.testRuntime.snapshot()[0].instance.id,
  );
  await page
    .getByRole("button", { name: "Download selected model", exact: true })
    .click();
  await expect(page.getByRole("button", { name: /^Use for / })).toHaveCount(0);
  await page.evaluate((id) => window.testRuntime.finishDownload(id), id);
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  return id;
}

async function installAndStart(page: Page) {
  const id = await installAndDownload(page);
  await page.getByRole("button", { name: "Start", exact: true }).click();
  return id;
}

for (const workflow of ["voice", "file"] as const) {
  const label = workflow === "voice" ? "Voice" : "Audio file";
  const purpose = workflow === "voice" ? "voice" : "stt";
  const other = workflow === "voice" ? "stt" : "voice";
  test(`install, download, start, explicitly use and return to ${label}`, async ({
    page,
    saves,
  }) => {
    const before = await openFreshRuntime(page, workflow);
    // Cancelling installation neither creates an instance nor selects a connection.
    await page.getByRole("button", { name: "Install", exact: true }).click();
    await page
      .getByRole("dialog", { name: "Install NeMo-Speech.cpp?", exact: true })
      .getByRole("button", { name: "Not now", exact: true })
      .click();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
    const id = await installAndStart(page);
    const use = page.getByRole("button", {
      name: `Use for ${label}`,
      exact: true,
    });
    await expect(use).toBeEnabled();
    expect((await snapshot(page)).savedConnections.selected).toEqual(
      before.savedConnections.selected,
    );
    await use.click();
    const save = await saves.waitForStart();
    await expect(runtimePage(page)).toHaveAttribute("aria-current", "page");
    await expect(
      page.getByRole("button", { name: "Selecting…", exact: true }),
    ).toBeDisabled();
    expect((await snapshot(page)).savedConnections.selected).toEqual(
      before.savedConnections.selected,
    );
    await saves.complete(save, "success");
    await expect(
      rail(page).getByRole("button", {
        name: workflow === "voice" ? "Voice transcription" : "Audio file",
        exact: true,
      }),
    ).toHaveAttribute("aria-current", "page");
    const after = await snapshot(page);
    expect(after.savedConnections.selected![purpose]).toBe(
      `builtin-fixture-${id}`,
    );
    expect(after.savedConnections.selected![other]).toBe(
      before.savedConnections.selected![other],
    );
    expect(
      after.savedConnections.entries!.find((entry) => entry.id === "original"),
    ).toEqual(
      before.savedConnections.entries!.find((entry) => entry.id === "original"),
    );
    expect(after.rememberedModels).toEqual(before.rememberedModels);
    expect(after.voiceTranscription.realtime).toBe(
      before.voiceTranscription.realtime,
    );
    await runtimePage(page).click();
    await page
      .getByRole("button", { name: `Open ${label}`, exact: true })
      .click();
    await expect(
      rail(page).getByRole("button", {
        name: workflow === "voice" ? "Voice transcription" : "Audio file",
        exact: true,
      }),
    ).toHaveAttribute("aria-current", "page");
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      `SetInstance:${id}`,
      `Install:${id}`,
      `DownloadModel:${id}:nemotron-3.5`,
      `Start:${id}`,
    ]);
  });
}

test("a failed use transaction stays in Runtimes and preserves the previous connection", async ({
  page,
  saves,
}) => {
  const before = await openFreshRuntime(page);
  await installAndStart(page);
  await page
    .getByRole("button", { name: "Use for Voice", exact: true })
    .click();
  const save = await saves.waitForStart();
  await saves.complete(save, "failure");
  await expect(runtimePage(page)).toHaveAttribute("aria-current", "page");
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Could not select this runtime for Voice" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Use for Voice", exact: true }),
  ).toBeEnabled();
  expect((await snapshot(page)).savedConnections.selected).toEqual(
    before.savedConnections.selected,
  );
});

test("changing recovery installation A to B to A prevents an old save from navigating", async ({
  page,
  saves,
}) => {
  await openFreshRuntime(page);
  const id = await installAndStart(page);
  await page.evaluate(() => window.testRuntime.addDuplicate("zz-recovery"));
  await page
    .getByRole("button", { name: "Use for Voice", exact: true })
    .click();
  const save = await saves.waitForStart();
  await page
    .getByRole("button", {
      name: "Duplicate speech · zz-recovery",
      exact: true,
    })
    .click();
  await page
    .getByRole("button", { name: `NeMo-Speech.cpp · ${id}`, exact: true })
    .click();
  await saves.complete(save, "success");
  await expect(runtimePage(page)).toHaveAttribute("aria-current", "page");
  await expect(
    page.getByRole("button", { name: "Open Voice", exact: true }),
  ).toBeEnabled();
});

test("leaving Runtimes while selection is pending does not return on late success", async ({
  page,
  saves,
}) => {
  await openFreshRuntime(page);
  await installAndStart(page);
  await page
    .getByRole("button", { name: "Use for Voice", exact: true })
    .click();
  const save = await saves.waitForStart();
  const history = rail(page).getByRole("button", {
    name: "History",
    exact: true,
  });
  await history.click();
  await saves.complete(save, "success");
  await expect(history).toHaveAttribute("aria-current", "page");
});

test("cancelled download and failed startup never offer a use action", async ({
  page,
}) => {
  const before = await openFreshRuntime(page);
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Install NeMo-Speech.cpp?", exact: true })
    .getByRole("button", { name: "Download and install", exact: true })
    .click();
  const id = await page.evaluate(
    () => window.testRuntime.snapshot()[0].instance.id,
  );
  await page
    .getByRole("button", { name: "Download selected model", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Cancel operation", exact: true })
    .click();
  await expect(page.getByRole("button", { name: /^Use for / })).toHaveCount(0);
  await page
    .getByRole("button", { name: "Download selected model", exact: true })
    .click();
  await page.evaluate((id) => {
    window.testRuntime.finishDownload(id);
    window.testRuntime.holdLifecycle();
  }, id);
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await page.evaluate((id) => window.testRuntime.acknowledgeLifecycle(id), id);
  await expect(page.getByRole("button", { name: /^Use for / })).toHaveCount(0);
  await page.evaluate(
    (id) => window.testRuntime.finishLifecycle(id, "failed"),
    id,
  );
  await expect(
    page.getByRole("region", { name: "Runtime setup summary", exact: true }),
  ).toContainText("Needs attention");
  await expect(page.getByRole("button", { name: /^Use for / })).toHaveCount(0);
  expect((await snapshot(page)).savedConnections.selected).toEqual(
    before.savedConnections.selected,
  );
});

test("dirty workflow edits stay behind the existing navigation decision without selecting the runtime", async ({
  page,
}) => {
  const before = await openFreshRuntime(page);
  await installAndStart(page);
  await rail(page)
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  const inspector = page.locator('[data-pane="configuration"]');
  await inspector
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  const duration = inspector.locator("#max-duration");
  const previous = await duration.inputValue();
  await duration.fill(previous === "140" ? "141" : "140");
  await runtimePage(page).click();
  const decision = page.getByRole("dialog", {
    name: "Save changes?",
    exact: true,
  });
  await expect(decision).toBeVisible();
  await decision
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(
    inspector.getByText("Unsaved changes", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Use for Voice", exact: true }),
  ).toHaveCount(0);
  expect((await snapshot(page)).savedConnections.selected).toEqual(
    before.savedConnections.selected,
  );
  expect((await snapshot(page)).maxDurationSeconds).toBe(
    before.maxDurationSeconds,
  );
});
