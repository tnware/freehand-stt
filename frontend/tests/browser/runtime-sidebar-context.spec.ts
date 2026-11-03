import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

const summary = (page: Page) =>
  page.getByRole("region", { name: "Runtime setup summary", exact: true });
const inventory = (page: Page) =>
  page.getByRole("group", { name: "Runtime inventory", exact: true });

async function openRuntime(page: Page, query = "") {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto(`/tests/browser/app/?main&setup-ready&runtime${query}`);
  await page
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await expect(summary(page)).toBeVisible();
}

test("new runtime sidebar labels the recommendation and next setup step without probing hardware", async ({
  page,
}) => {
  await openRuntime(page);
  await expect(summary(page)).toContainText("Recommended model");
  await expect(summary(page)).toContainText("Nemotron 3.5 Streaming");
  await expect(summary(page)).toContainText("Next step");
  await expect(summary(page)).toContainText("Install runtime");
  await expect(summary(page)).not.toContainText("Selected model");
  await expect(inventory(page)).toContainText(
    "Recommended: Nemotron 3.5 Streaming",
  );
  await expect(page.getByText("Platform options", { exact: true })).toHaveCount(
    0,
  );
  await expect(
    summary(page).getByTitle("Installed runtime binary", { exact: true }),
  ).toHaveCount(0);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  expect(await page.evaluate(() => window.testRuntime.snapshot())).toEqual([]);
});

test("sidebar keeps the actual selected model while download, start, error and running states change", async ({
  page,
}) => {
  await openRuntime(page, "&runtime-ready");
  const id = "nemo-default";
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await page
    .getByRole("article", { name: "Parakeet TDT v3", exact: true })
    .getByRole("button", { name: "Select", exact: true })
    .click();
  await expect(summary(page)).toContainText("Selected model");
  await expect(summary(page)).toContainText("Parakeet TDT v3");
  await expect(summary(page)).not.toContainText("Recommended model");
  await expect(summary(page)).toContainText("Start runtime");
  await expect(inventory(page)).toContainText("Parakeet TDT v3");
  await page.evaluate((id) => {
    const row = window.testRuntime
      .snapshot()
      .find((row) => row.instance.id === id)!;
    window.testRuntime.change(id, {
      models: row.status.models!.map((model) => ({
        ...model,
        installed: false,
      })),
    });
  }, id);
  await expect(summary(page)).toContainText("Download selected model");
  await page.evaluate(
    (id) =>
      window.testRuntime.change(id, {
        phase: "download",
        operation: {
          id: 1,
          kind: "download",
          model: "nemotron-3.5",
          outcome: "running",
          error: "",
        },
        acquisition: { phase: "downloading", bytes: 1024, totalBytes: 2048 },
      }),
    id,
  );
  await expect(summary(page)).toContainText("In progress");
  await expect(summary(page)).toContainText("Downloading model");
  // Operation model is separate from the saved selection.
  await expect(summary(page)).toContainText("Parakeet TDT v3");
  await expect(summary(page)).toContainText("Nemotron 3.5 Streaming");
  await page.evaluate(
    (id) =>
      window.testRuntime.change(id, {
        state: "starting",
        phase: "start",
        operation: {
          id: 2,
          kind: "start",
          model: "",
          outcome: "running",
          error: "",
        },
        startupProgress: { phase: "warming_up", startedAt: Date.now() },
      }),
    id,
  );
  await expect(summary(page)).toContainText("Warming up selected model");
  await expect(summary(page)).not.toContainText("Running locally");
  await page.evaluate(
    (id) =>
      window.testRuntime.change(id, {
        state: "error",
        phase: "",
        operation: {
          id: 2,
          kind: "start",
          model: "",
          outcome: "failed",
          error: "Could not start selected model",
        },
      }),
    id,
  );
  await expect(summary(page)).toContainText("Needs attention");
  await expect(summary(page)).toContainText("Review runtime error");
  await page.evaluate(
    (id) =>
      window.testRuntime.change(id, {
        state: "running",
        operation: {
          id: 3,
          kind: "start",
          model: "",
          outcome: "succeeded",
          error: "",
        },
      }),
    id,
  );
  await expect(summary(page)).toContainText("Runtime ready");
  await expect(summary(page)).toContainText("Running locally");
  expect(
    await page.evaluate(() => window.testRuntime.snapshot()[0].instance.model),
  ).toBe("parakeet-tdt");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    `Stop:${id}`,
    `SetInstance:${id}`,
  ]);
});

test("sidebar shows the missing optional speech download before suggesting Start", async ({
  page,
}) => {
  await openRuntime(page, "&runtime-ready&runtime-combined");
  await expect(summary(page)).toContainText("Selected speech model");
  await expect(summary(page)).toContainText("Magpie TTS Multilingual");
  await expect(summary(page)).toContainText("Download speech model");
  await expect(summary(page)).not.toContainText("Start runtime");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("sidebar follows the installation chosen for duplicate recovery", async ({
  page,
}) => {
  await openRuntime(page, "&runtime-ready");
  await page.evaluate(() => window.testRuntime.addDuplicate("zz-recovery"));
  await page
    .getByRole("button", {
      name: "Duplicate speech · zz-recovery",
      exact: true,
    })
    .click();
  await page
    .getByRole("article", { name: "Parakeet TDT v3", exact: true })
    .getByRole("button", { name: "Select", exact: true })
    .click();
  await expect(summary(page)).toContainText("Parakeet TDT v3");
  await expect(summary(page)).toContainText("Start runtime");
  await expect(inventory(page)).toContainText("Parakeet TDT v3");
  const rows = await page.evaluate(() => window.testRuntime.snapshot());
  expect(
    rows.find((row) => row.instance.id === "nemo-default")?.instance.model,
  ).toBe("nemotron-3.5");
  expect(
    rows.find((row) => row.instance.id === "zz-recovery")?.instance.model,
  ).toBe("parakeet-tdt");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "SetInstance:zz-recovery",
  ]);
});
