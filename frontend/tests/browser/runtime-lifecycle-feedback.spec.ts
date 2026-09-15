import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

const id = "nemo-default";
const operation = (page: Page) =>
  page.getByRole("status", { name: "Runtime operation", exact: true });
const rail = (page: Page) =>
  page.getByRole("navigation", { name: "Workspace", exact: true });

async function openRuntime(page: Page) {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page.evaluate(() => window.testRuntime.holdLifecycle());
  await rail(page)
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
}

test("runtime Start and Stop report pending commands, live stages, and completion without refreshing", async ({
  page,
}) => {
  await openRuntime(page);
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(operation(page)).toContainText("Stopping runtime");
  await page.evaluate((id) => window.testRuntime.acknowledgeLifecycle(id), id);
  await expect(operation(page)).toContainText("Stopping runtime");
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toHaveCount(0);
  await page.evaluate((id) => window.testRuntime.finishLifecycle(id), id);
  await expect(operation(page)).toContainText("Runtime stopped.");
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await expect(operation(page)).toContainText("Starting runtime");
  await expect(operation(page)).not.toContainText("Runtime stopped.");
  await page.evaluate((id) => {
    window.testRuntime.acknowledgeLifecycle(id);
    window.testRuntime.change(id, {
      startupProgress: { phase: "warming_up", startedAt: Date.now() - 5000 },
    });
  }, id);
  await expect(operation(page)).toContainText(
    /Warming up selected model · [5-9]s in this stage/,
  );
  await page.evaluate((id) => window.testRuntime.finishLifecycle(id), id);
  await expect(operation(page)).toContainText("Runtime ready.");
  await expect(
    page.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    `Stop:${id}`,
    `Start:${id}`,
  ]);
});

for (const task of ["Voice transcription", "Audio file"]) {
  test(`${task} keeps lifecycle feedback visible in its sidebar`, async ({
    page,
  }) => {
    await openRuntime(page);
    await page.evaluate(
      (id) =>
        window.testRuntime.change(id, { state: "stopped", backend: "cuda" }),
      id,
    );
    await rail(page).getByRole("button", { name: task, exact: true }).click();
    const sidebar = page.getByRole("complementary", {
      name: `${task} settings`,
      exact: true,
    });
    const status = sidebar.getByRole("status", {
      name: "Local runtime status",
      exact: true,
    });
    await sidebar.getByRole("button", { name: "Start", exact: true }).click();
    await expect(status).toContainText("Starting runtime");
    await expect(status).toBeInViewport();
    await page.evaluate((id) => {
      window.testRuntime.acknowledgeLifecycle(id);
      window.testRuntime.change(id, {
        startupProgress: {
          phase: "loading_warming",
          startedAt: Date.now() - 2000,
        },
      });
    }, id);
    await expect(status).toContainText("Loading and warming up selected model");
    await page.evaluate((id) => window.testRuntime.finishLifecycle(id), id);
    await expect(status).toContainText("Running");
    await sidebar.getByRole("button", { name: "Stop", exact: true }).click();
    await expect(status).toContainText("Stopping runtime");
    await page.evaluate(
      (id) => window.testRuntime.acknowledgeLifecycle(id),
      id,
    );
    await page.evaluate((id) => window.testRuntime.finishLifecycle(id), id);
    await expect(status).toContainText("Stopped");
    await expect(
      sidebar.getByRole("button", { name: "Start", exact: true }),
    ).toBeEnabled();
  });
}

test("Restart remains one backend operation through stopping and warming up", async ({
  page,
}) => {
  await openRuntime(page);
  await page.getByRole("button", { name: "Restart", exact: true }).click();
  await expect(operation(page)).toContainText("Restarting runtime");
  await page.evaluate((id) => window.testRuntime.acknowledgeLifecycle(id), id);
  await expect(operation(page)).toContainText("Stopping runtime");
  await page.evaluate(
    (id) =>
      window.testRuntime.change(id, {
        state: "starting",
        startupProgress: { phase: "waiting_ready", startedAt: Date.now() },
      }),
    id,
  );
  await expect(operation(page)).toContainText("Waiting for runtime readiness");
  await page.evaluate((id) => window.testRuntime.finishLifecycle(id), id);
  await expect(operation(page)).toContainText("Runtime restarted.");
  await expect(
    page.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    `Restart:${id}`,
  ]);
});

test("startup failures and cancellation restore actions and keep their outcome visible", async ({
  page,
}) => {
  await openRuntime(page);
  await page.evaluate(
    (id) => window.testRuntime.change(id, { state: "stopped" }),
    id,
  );
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await expect(operation(page)).toContainText("Starting runtime");
  await page.evaluate((id) => window.testRuntime.acknowledgeLifecycle(id), id);
  await page.evaluate(
    (id) => window.testRuntime.finishLifecycle(id, "failed"),
    id,
  );
  await expect(operation(page).getByRole("alert")).toContainText(
    "Runtime could not start.",
  );
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await expect(operation(page)).toContainText("Starting runtime");
  await page.evaluate((id) => window.testRuntime.acknowledgeLifecycle(id), id);
  await page
    .getByRole("button", { name: "Cancel operation", exact: true })
    .click();
  await expect(operation(page)).toContainText("Operation cancelled.");
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    `Start:${id}`,
    `Start:${id}`,
    `Cancel:${id}`,
  ]);
});
