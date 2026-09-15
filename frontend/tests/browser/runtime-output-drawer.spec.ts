import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";

async function installOutputFixture(page: Page) {
  await page.route(
    "**/bindings/**/internal/managedruntime/manager.*",
    (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `
      const enabled = new Set();
      const calls = window.outputCalls = [];
      export const EnableProcessOutput = async ({instanceID}) => { enabled.add(instanceID); calls.push('enable:' + instanceID); };
      export const DisableProcessOutput = async ({instanceID}) => { enabled.delete(instanceID); calls.push('disable:' + instanceID); };
      export const ClearProcessOutput = async () => {};
      export const ReadProcessOutput = async ({instanceID}) => {
        calls.push('read:' + instanceID);
        if (!enabled.has(instanceID)) throw new Error('Read before consent');
        return { enabled: true, chunks: [{ sequence: 1, timestamp: 0, stream: 'stderr', text: '<b>Sensitive startup diagnostic</b> for ' + instanceID + '\\r\\n' }], next: 1, truncated: false };
      };
      `,
      }),
  );
}

function outputCalls(page: Page) {
  return page.evaluate(
    () => (window as unknown as { outputCalls: string[] }).outputCalls,
  );
}

async function disableCalls(page: Page) {
  return (await outputCalls(page)).filter((call) =>
    call.startsWith("disable:"),
  );
}

test("global runtime output requires consent and clears on window or panel hide", async ({
  page,
}) => {
  await installOutputFixture(page);
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "starting",
      phase: "start",
    }),
  );
  await page
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await expect(page.getByRole("tab", { name: /^Recent/ })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await page.getByRole("button", { name: "View output", exact: true }).click();
  const output = page.getByRole("region", {
    name: "Read-only process output",
    exact: true,
  });
  const show = page.getByRole("button", { name: "Show output", exact: true });
  await expect(show).toBeEnabled();
  await expect(output).toHaveText("");
  expect(await outputCalls(page)).toEqual([]);
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
  await expect.poll(() => disableCalls(page)).toEqual(["disable:nemo-default"]);
  await show.click();
  await expect(output).toContainText("Sensitive startup diagnostic");
  const toggle = page.getByRole("button", {
    name: "Toggle bottom panel",
    exact: true,
  });
  await toggle.click();
  await expect(output).toBeHidden();
  await expect.poll(() => disableCalls(page)).toHaveLength(2);
  await toggle.click();
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  await show.click();
  await expect(output).toContainText("Sensitive startup diagnostic");
  await page.setViewportSize({ width: 1156, height: 520 });
  await expect(output).toBeHidden();
  await expect.poll(() => disableCalls(page)).toHaveLength(3);
  await page.setViewportSize({ width: 1156, height: 850 });
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("accepted output survives page navigation and resets in Settings or when its tab changes", async ({
  page,
}) => {
  await installOutputFixture(page);
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  const outputTab = page.getByRole("tab", {
    name: "Runtime output",
    exact: true,
  });
  await outputTab.click();
  const show = page.getByRole("button", { name: "Show output", exact: true });
  const output = page.getByRole("region", {
    name: "Read-only process output",
    exact: true,
  });
  await show.click();
  await expect(output).toContainText("Sensitive startup diagnostic");
  for (const area of [
    "Local runtime",
    "History",
    "Audio file",
    "Voice transcription",
  ]) {
    await page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: area, exact: true })
      .click();
    await expect(outputTab).toHaveAttribute("aria-selected", "true");
    await expect(output).toContainText("Sensitive startup diagnostic");
    await expect(show).toBeHidden();
  }
  expect(await disableCalls(page)).toEqual([]);
  expect(
    (await outputCalls(page)).filter((call) => call.startsWith("enable:")),
  ).toEqual(["enable:nemo-default"]);
  const workspace = page.getByRole("navigation", {
    name: "Workspace",
    exact: true,
  });
  await workspace
    .getByRole("button", { name: "Settings", exact: true })
    .click();
  await expect(page.locator("#workbench-bottom-panel")).toHaveCount(0);
  await expect.poll(() => disableCalls(page)).toEqual(["disable:nemo-default"]);
  await workspace
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await expect(outputTab).toHaveAttribute("aria-selected", "true");
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  await show.click();
  await expect(output).toContainText("Sensitive startup diagnostic");
  expect(
    (await outputCalls(page)).filter((call) => call.startsWith("enable:")),
  ).toEqual(["enable:nemo-default", "enable:nemo-default"]);
  await page.getByRole("tab", { name: "Diagnostics", exact: true }).click();
  await expect
    .poll(() => disableCalls(page))
    .toEqual(["disable:nemo-default", "disable:nemo-default"]);
  await outputTab.click();
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("changing the explicit output runtime ends consent without running either runtime", async ({
  page,
}) => {
  await installOutputFixture(page);
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page.evaluate(() => window.testRuntime.addSecondProvider());
  await page.getByRole("tab", { name: "Runtime output", exact: true }).click();
  const output = page.getByRole("region", {
    name: "Read-only process output",
    exact: true,
  });
  const show = page.getByRole("button", { name: "Show output", exact: true });
  await show.click();
  await expect(output).toContainText("nemo-default");
  await page.getByRole("button", { name: "Runtime", exact: true }).click();
  await page
    .getByRole("option", { name: "Second speech provider", exact: true })
    .click();
  await expect.poll(() => disableCalls(page)).toEqual(["disable:nemo-default"]);
  await expect(show).toBeVisible();
  await expect(output).toHaveText("");
  expect(
    (await outputCalls(page)).filter((call) => call.endsWith(":other-speech")),
  ).toEqual([]);
  await show.click();
  await expect(output).toContainText("other-speech");
  await expect(output).not.toContainText("nemo-default");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
