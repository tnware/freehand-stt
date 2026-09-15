import { test, expect } from "./fixtures";
import { Purpose } from "../../bindings/github.com/tnware/freehand-stt/internal/savedconnection/models";

test("managed metadata waits for warm-up and refreshes after readiness", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?main&runtime&runtime-ready&runtime-starting&voice-metadata-control",
  );
  const status = page.getByRole("button", {
    name: /Transcription connection status:/,
  });
  await expect(status).toContainText("Starting runtime");
  expect(await page.evaluate(() => window.testMetadata.calls)).toEqual([]);
  await status.click();
  const panel = page.getByRole("dialog", {
    name: "Active connection",
    exact: true,
  });
  await expect(panel).toContainText("Waiting for model loading and warm-up");
  await expect(panel).not.toContainText("Network failed");
  await expect(panel.getByRole("button", { name: /Check/ })).toBeDisabled();
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "running",
      phase: "",
      startupProgress: undefined,
    }),
  );
  await expect
    .poll(() => page.evaluate(() => window.testMetadata.calls))
    .toEqual([Purpose.Voice]);
  await page.evaluate(
    (purpose) => window.testMetadata.complete(purpose, true),
    Purpose.Voice,
  );
  await expect(status).toContainText("Reachable");
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "starting",
      phase: "start",
      operation: {
        id: 2,
        kind: "restart",
        model: "",
        outcome: "running",
        error: "",
      },
    }),
  );
  await expect(status).toContainText("Starting runtime");
  expect(await page.evaluate(() => window.testMetadata.calls)).toEqual([
    Purpose.Voice,
  ]);
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", { state: "running", phase: "" }),
  );
  await expect
    .poll(() => page.evaluate(() => window.testMetadata.calls))
    .toEqual([Purpose.Voice, Purpose.Voice]);
  await page.evaluate(
    (purpose) => window.testMetadata.complete(purpose, true),
    Purpose.Voice,
  );
  await expect(status).toContainText("Reachable");
});

test("Voice shows initial runtime warm-up and waits for running before enabling Record", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto(
    "/tests/browser/app/?main&runtime&runtime-ready&runtime-starting",
  );
  const capture = page.getByRole("region", {
    name: "Voice capture",
    exact: true,
  });
  await expect(capture).toContainText("Loading and warming up selected model");
  await expect(
    capture.getByRole("button", { name: "Start recording", exact: true }),
  ).toBeDisabled();
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "running",
      phase: "",
      startupProgress: undefined,
      operation: {
        id: 1,
        kind: "startup",
        model: "",
        outcome: "succeeded",
        error: "",
      },
    }),
  );
  await expect(capture).toContainText("Ready to dictate");
  await expect(
    capture.getByRole("button", { name: "Start recording", exact: true }),
  ).toBeEnabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("runtime readiness cannot disable Stop for an existing recording", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?main&runtime&runtime-ready&runtime-starting&work-busy",
  );
  const capture = page.getByRole("region", {
    name: "Voice capture",
    exact: true,
  });
  await expect(
    capture.getByRole("button", { name: "Stop recording", exact: true }),
  ).toBeEnabled();
  await expect(capture).toContainText("Recording");
});

for (const success of [true, false]) {
  test(`Voice reports its initial server probe and ${success ? "reachable result" : "failed check with retry"}`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.goto(
      "/tests/browser/app/?main&setup-ready&workflows&voice-metadata-control",
    );
    const capture = page.getByRole("region", {
      name: "Voice capture",
      exact: true,
    });
    await expect(capture).toContainText("Checking connection");
    await expect(
      capture.getByRole("button", { name: "Start recording", exact: true }),
    ).toBeDisabled();
    await expect
      .poll(() => page.evaluate(() => window.testMetadata.calls))
      .toEqual([Purpose.Voice]);
    await page.evaluate(
      ({ purpose, success }) => window.testMetadata.complete(purpose, success),
      { purpose: Purpose.Voice, success },
    );
    await expect(
      capture.getByRole("button", { name: "Start recording", exact: true }),
    ).toBeEnabled();
    if (success) {
      await expect(capture).toContainText("Ready to dictate");
    } else {
      await expect(capture).not.toContainText("Ready to dictate");
      await capture
        .getByRole("button", { name: "Check connection", exact: true })
        .click();
      await expect(capture).toContainText("Checking connection");
      await expect(
        capture.getByRole("button", { name: "Start recording", exact: true }),
      ).toBeDisabled();
      await page.evaluate(
        (purpose) => window.testMetadata.complete(purpose, true),
        Purpose.Voice,
      );
      await expect(capture).toContainText("Ready to dictate");
      await expect(
        capture.getByRole("button", { name: "Start recording", exact: true }),
      ).toBeEnabled();
    }
  });
}
