import { test, expect } from "./fixtures";

test("inline runtime output requires consent and clears on hide, collapse and navigation", async ({
  page,
}) => {
  await page.route(
    "**/bindings/**/internal/managedruntime/manager.*",
    (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `
      let enabled = false;
      const calls = window.outputCalls = [];
      export const EnableProcessOutput = async ({instanceID}) => { enabled = true; calls.push('enable:' + instanceID); };
      export const DisableProcessOutput = async ({instanceID}) => { enabled = false; calls.push('disable:' + instanceID); };
      export const ClearProcessOutput = async () => {};
      export const ReadProcessOutput = async ({instanceID}) => {
        calls.push('read:' + instanceID);
        if (!enabled) throw new Error('Read before consent');
        return { enabled, chunks: [{ sequence: 1, timestamp: 0, stream: 'stderr', text: '<b>Sensitive startup diagnostic</b>\\r\\n' }], next: 1, truncated: false };
      };
    `,
      }),
  );
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "starting",
      phase: "start",
    }),
  );
  await page
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  const output = page.getByRole("region", {
    name: "Read-only process output",
    exact: true,
  });
  const show = page.getByRole("button", { name: "Show output", exact: true });
  await expect(show).toBeEnabled();
  await expect(output).toHaveText("");
  expect(
    await page.evaluate(
      () => (window as unknown as { outputCalls: string[] }).outputCalls,
    ),
  ).toEqual([]);
  await show.click();
  await expect(output).toContainText("Sensitive startup diagnostic");
  await expect(output.locator("b")).toHaveCount(0);
  // Process exit must leave diagnostics readable rather than wiping the cause.
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "error",
      phase: "",
      error: "Startup failed.",
    }),
  );
  await expect(output).toContainText("Sensitive startup diagnostic");
  await page.evaluate(() =>
    (
      window as unknown as {
        _wails: {
          dispatchWailsEvent: (event: { name: string; data: null }) => void;
        };
      }
    )._wails.dispatchWailsEvent({ name: "shell:hidden", data: null }),
  );
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  await expect
    .poll(() =>
      page.evaluate(
        () =>
          (window as unknown as { outputCalls: string[] }).outputCalls.filter(
            (call) => call.startsWith("disable:"),
          ).length,
      ),
    )
    .toBe(1);
  await show.click();
  await expect(output).toContainText("Sensitive startup diagnostic");
  await page.getByRole("button", { name: "Hide panel", exact: true }).click();
  await expect
    .poll(() =>
      page.evaluate(
        () =>
          (window as unknown as { outputCalls: string[] }).outputCalls.filter(
            (call) => call.startsWith("disable:"),
          ).length,
      ),
    )
    .toBe(2);
  await page.getByRole("button", { name: "Show panel", exact: true }).click();
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  await show.click();
  await expect(output).toContainText("Sensitive startup diagnostic");
  await page
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await expect
    .poll(() =>
      page.evaluate(
        () =>
          (window as unknown as { outputCalls: string[] }).outputCalls.filter(
            (call) => call.startsWith("disable:"),
          ).length,
      ),
    )
    .toBe(3);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("workflow output releases consent while Settings covers the retained workspace", async ({
  page,
}) => {
  await page.route(
    "**/bindings/**/internal/managedruntime/manager.*",
    (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `
      let enabled = false;
      window.outputCalls = [];
      export const EnableProcessOutput = async () => { enabled = true; window.outputCalls.push('enable'); };
      export const DisableProcessOutput = async () => { enabled = false; window.outputCalls.push('disable'); };
      export const ClearProcessOutput = async () => {};
      export const ReadProcessOutput = async () => ({ enabled, chunks: [{sequence:1,timestamp:0,stream:'stderr',text:'Private workflow diagnostic'}], next:1, truncated:false });
    `,
      }),
  );
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page.getByRole("tab", { name: "Runtime output", exact: true }).click();
  const show = page.getByRole("button", { name: "Show output", exact: true });
  const output = page.getByRole("region", {
    name: "Read-only process output",
    exact: true,
  });
  await show.click();
  await expect(output).toContainText("Private workflow diagnostic");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await expect
    .poll(() =>
      page.evaluate(
        () => (window as unknown as { outputCalls: string[] }).outputCalls,
      ),
    )
    .toEqual(["enable", "disable"]);
  await page
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
