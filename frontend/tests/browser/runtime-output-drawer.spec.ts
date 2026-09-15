import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { installOutputFixture, outputCalls } from "./runtime-output-fixtures";

const output = (page: Page) =>
  page.getByRole("region", { name: "Read-only process output", exact: true });
const outputTab = (page: Page) =>
  page.getByRole("tab", { name: "Runtime output", exact: true });
const runtimeTabs = (page: Page) =>
  page.getByRole("tablist", { name: "Output runtimes", exact: true });
const bottomToggle = (page: Page) =>
  page.getByRole("button", { name: "Toggle bottom panel", exact: true });
const workspace = (page: Page) =>
  page.getByRole("navigation", { name: "Workspace", exact: true });

async function openApp(page: Page, width = 1280) {
  await installOutputFixture(page);
  await page.clock.install();
  await page.setViewportSize({ width, height: 850 });
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
}

async function callsOf(page: Page, kind: string) {
  return (await outputCalls(page)).filter((call) =>
    call.startsWith(`${kind}:`),
  );
}

async function hideShell(page: Page) {
  await page.evaluate(() => {
    window.testWindow.shown = false;
    (
      window as unknown as {
        _wails: {
          dispatchWailsEvent: (event: { name: string; data: null }) => void;
        };
      }
    )._wails.dispatchWailsEvent({ name: "shell:hidden", data: null });
  });
}

async function showShell(page: Page) {
  await page.evaluate(() => window.testWindow.emit("WindowShow"));
}

async function documentVisibility(page: Page, visible: boolean) {
  await page.evaluate((visible) => {
    Object.defineProperty(document, "hidden", {
      configurable: true,
      value: !visible,
    });
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      value: visible ? "visible" : "hidden",
    });
    document.dispatchEvent(new Event("visibilitychange"));
  }, visible);
}

async function expectNoOutputActivity(page: Page) {
  const before = await outputCalls(page);
  // Exercise multiple poll intervals without depending on wall-clock sleeps.
  await page.clock.fastForward(3000);
  expect(await outputCalls(page)).toEqual(before);
}

test("opening runtime output reads immediately and window visibility releases then restores the reader", async ({
  page,
}) => {
  await openApp(page);
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "starting",
      phase: "start",
    }),
  );
  await workspace(page)
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await expect(page.getByRole("tab", { name: /^Recent/ })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  expect(await outputCalls(page)).toEqual([]);
  await page.getByRole("button", { name: "View output", exact: true }).click();
  await expect(output(page)).toContainText("Sensitive startup diagnostic");
  await expect(output(page).locator("b")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Show output", exact: true }),
  ).toHaveCount(0);
  expect(await callsOf(page, "enable")).toEqual(["enable:nemo-default"]);
  const follow = page.getByRole("button", { name: "Follow", exact: true });
  await follow.click();
  await expect(follow).toHaveAttribute("aria-pressed", "false");
  // A failed runtime still has useful startup output. Displaying it never
  // starts or stops the runtime process.
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "error",
      phase: "",
      error: "Startup failed.",
    }),
  );
  await expect(output(page)).toContainText("Sensitive startup diagnostic");
  await expect(follow).toHaveAttribute("aria-pressed", "false");
  expect(await callsOf(page, "disable")).toEqual([]);
  expect(await callsOf(page, "enable")).toEqual(["enable:nemo-default"]);

  await hideShell(page);
  await expect
    .poll(() => callsOf(page, "disable"))
    .toEqual(["disable:nemo-default"]);
  await expect(output(page)).toHaveText("");
  await expectNoOutputActivity(page);
  await showShell(page);
  await expect(output(page)).toContainText("Sensitive startup diagnostic");
  expect(await callsOf(page, "enable")).toHaveLength(2);

  await documentVisibility(page, false);
  await expect.poll(() => callsOf(page, "disable")).toHaveLength(2);
  await expect(output(page)).toHaveText("");
  await expectNoOutputActivity(page);
  // A native show event cannot restart reads while the document is hidden.
  await showShell(page);
  await expectNoOutputActivity(page);
  await documentVisibility(page, true);
  await expect(output(page)).toContainText("Sensitive startup diagnostic");
  expect(await callsOf(page, "enable")).toHaveLength(3);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("minimizing stops output reads and native restore resumes the selected runtime", async ({
  page,
}) => {
  await openApp(page);
  await outputTab(page).click();
  await expect(output(page)).toContainText("nemo-default");
  await page.evaluate(() => window.testWindow.emit("WindowMinimise"));
  await expect
    .poll(() => callsOf(page, "disable"))
    .toEqual(["disable:nemo-default"]);
  await expect(output(page)).toHaveText("");
  await expectNoOutputActivity(page);
  await page.evaluate(() => window.testWindow.emit("WindowFocus"));
  await expectNoOutputActivity(page);
  await page.evaluate(() => window.testWindow.emit("WindowUnMinimise"));
  await expect(output(page)).toContainText("nemo-default");
  expect(await callsOf(page, "enable")).toHaveLength(2);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("hidden or unavailable bottom panels stop reading and restore the selected runtime", async ({
  page,
}) => {
  await openApp(page);
  await outputTab(page).click();
  await expect(output(page)).toContainText("nemo-default");
  await bottomToggle(page).click();
  await expect(output(page)).toHaveCount(0);
  await expect
    .poll(() => callsOf(page, "disable"))
    .toEqual(["disable:nemo-default"]);
  await expectNoOutputActivity(page);
  await bottomToggle(page).click();
  await expect(outputTab(page)).toHaveAttribute("aria-selected", "true");
  await expect(output(page)).toContainText("nemo-default");
  await page.setViewportSize({ width: 1280, height: 520 });
  await expect(output(page)).toHaveCount(0);
  await expect(bottomToggle(page)).toBeDisabled();
  await expect.poll(() => callsOf(page, "disable")).toHaveLength(2);
  await expectNoOutputActivity(page);
  await page.setViewportSize({ width: 1280, height: 850 });
  await expect(output(page)).toContainText("nemo-default");
  expect(await callsOf(page, "enable")).toHaveLength(3);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("output survives page navigation but stops in Settings or on a different panel tab", async ({
  page,
}) => {
  await openApp(page);
  await outputTab(page).click();
  await expect(output(page)).toContainText("Sensitive startup diagnostic");
  for (const area of [
    "Local runtime",
    "History",
    "Audio file",
    "Voice transcription",
  ]) {
    await workspace(page)
      .getByRole("button", { name: area, exact: true })
      .click();
    await expect(outputTab(page)).toHaveAttribute("aria-selected", "true");
    await expect(output(page)).toContainText("Sensitive startup diagnostic");
  }
  expect(await callsOf(page, "disable")).toEqual([]);
  expect(await callsOf(page, "enable")).toEqual(["enable:nemo-default"]);
  await workspace(page)
    .getByRole("button", { name: "Settings", exact: true })
    .click();
  await expect(page.locator("#workbench-bottom-panel")).toHaveCount(0);
  await expect
    .poll(() => callsOf(page, "disable"))
    .toEqual(["disable:nemo-default"]);
  await expectNoOutputActivity(page);
  await workspace(page)
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await expect(outputTab(page)).toHaveAttribute("aria-selected", "true");
  await expect(output(page)).toContainText("Sensitive startup diagnostic");
  await page.getByRole("tab", { name: "Diagnostics", exact: true }).click();
  await expect(output(page)).toHaveCount(0);
  await expect.poll(() => callsOf(page, "disable")).toHaveLength(2);
  await expectNoOutputActivity(page);
  await outputTab(page).click();
  await expect(output(page)).toContainText("Sensitive startup diagnostic");
  expect(await callsOf(page, "enable")).toHaveLength(3);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

for (const width of [1280, 560]) {
  test(`runtime tabs release the previous reader and immediately enable the next at ${width}px`, async ({
    page,
  }) => {
    await openApp(page, width);
    await page.evaluate(() => {
      window.testRuntime.addSecondProvider();
      window.testRuntime.change("other-speech", {
        state: "stopped",
        phase: "",
      });
    });
    await outputTab(page).click();
    await expect(output(page)).toContainText("nemo-default");
    const first = runtimeTabs(page).getByRole("tab", { selected: true });
    const next = runtimeTabs(page).getByRole("tab", {
      name: "Second speech provider",
      exact: true,
    });
    await expect(next).toBeInViewport();
    await expect(next).toBeEnabled();
    await first.focus();
    await page.evaluate(() => window.testOutput.holdNextEnable());
    await page.keyboard.press("End");
    await expect(next).toBeFocused();
    await expect(next).toHaveAttribute("aria-selected", "true");
    await expect
      .poll(() => callsOf(page, "disable"))
      .toEqual(["disable:nemo-default"]);
    await expect
      .poll(() => page.evaluate(() => window.testOutput.pendingEnables))
      .toBe(1);
    await expect(output(page)).toHaveText("");
    expect(await callsOf(page, "enable")).toEqual([
      "enable:nemo-default",
      "enable:other-speech",
    ]);
    expect(await callsOf(page, "read")).not.toContain("read:other-speech");
    await page.evaluate(() => window.testOutput.releaseEnable());
    await expect(output(page)).toContainText("other-speech");
    await expect(output(page)).not.toContainText("nemo-default");
    const calls = await outputCalls(page);
    expect(calls.indexOf("disable:nemo-default")).toBeLessThan(
      calls.indexOf("enable:other-speech"),
    );
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  });
}

test("a failed output enable stays idle until Retry output is selected", async ({
  page,
}) => {
  await openApp(page);
  await page.evaluate(() => window.testOutput.failNextEnable());
  await outputTab(page).click();
  await expect(page.getByRole("alert")).toContainText(
    "Could not enable process output.",
  );
  await expect(output(page)).toHaveText("");
  expect(await callsOf(page, "read")).toEqual([]);
  await expectNoOutputActivity(page);
  await page.getByRole("button", { name: "Retry output", exact: true }).click();
  await expect(output(page)).toContainText("nemo-default");
  expect(await callsOf(page, "enable")).toEqual([
    "enable:nemo-default",
    "enable:nemo-default",
  ]);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("an enable reply received after hiding releases its reader without reading output", async ({
  page,
}) => {
  await openApp(page);
  await page.evaluate(() => window.testOutput.holdNextEnable());
  await outputTab(page).click();
  await expect
    .poll(() => page.evaluate(() => window.testOutput.pendingEnables))
    .toBe(1);
  await hideShell(page);
  await page.evaluate(() => window.testOutput.releaseEnable());
  await expect
    .poll(() => callsOf(page, "disable"))
    .toContain("disable:nemo-default");
  await expect
    .poll(() => page.evaluate(() => window.testOutput.enabledInstances()))
    .toEqual([]);
  await expect(output(page)).toHaveText("");
  expect(await callsOf(page, "read")).toEqual([]);
  await expectNoOutputActivity(page);
  await showShell(page);
  await expect(output(page)).toContainText("nemo-default");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("a late initial visibility reply cannot reopen output after the shell hides", async ({
  page,
}) => {
  await openApp(page);
  await page.evaluate(() => window.testWindow.deferNextVisibilityRead());
  await outputTab(page).click();
  await expect
    .poll(() => page.evaluate(() => window.testWindow.pendingVisibilityReads))
    .toBe(1);
  await hideShell(page);
  await page.evaluate(() => window.testWindow.releaseVisibilityRead());
  await expect
    .poll(() => page.evaluate(() => window.testWindow.pendingVisibilityReads))
    .toBe(0);
  await expectNoOutputActivity(page);
  expect(await outputCalls(page)).toEqual([]);
  await expect(output(page)).toHaveText("");
  // Remounting the selected tab while hidden must query current native state.
  await bottomToggle(page).click();
  await bottomToggle(page).click();
  await expectNoOutputActivity(page);
  expect(await outputCalls(page)).toEqual([]);
  await showShell(page);
  await expect(output(page)).toContainText("nemo-default");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("a delayed reader cannot disable the same runtime after the panel reopens", async ({
  page,
}) => {
  await openApp(page);
  await page.evaluate(() => window.testOutput.holdNextEnable());
  await outputTab(page).click();
  await expect
    .poll(() => page.evaluate(() => window.testOutput.pendingEnables))
    .toBe(1);
  await bottomToggle(page).click();
  await expect(output(page)).toHaveCount(0);
  await bottomToggle(page).click();
  await expect(output(page)).toBeVisible();
  await page.evaluate(() => window.testOutput.releaseEnable());
  await expect(output(page)).toContainText("nemo-default");
  await page.clock.fastForward(3000);
  await expect(output(page)).toContainText("nemo-default");
  expect(
    await page.evaluate(() => window.testOutput.enabledInstances()),
  ).toEqual(["nemo-default"]);
  expect(await callsOf(page, "enable")).toEqual([
    "enable:nemo-default",
    "enable:nemo-default",
  ]);
  const calls = await outputCalls(page);
  expect(calls.lastIndexOf("disable:nemo-default")).toBeLessThan(
    calls.lastIndexOf("enable:nemo-default"),
  );
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("a removed output target stays explicit until another runtime is selected", async ({
  page,
}) => {
  await openApp(page);
  await page.evaluate(() => window.testRuntime.addSecondProvider());
  await outputTab(page).click();
  await expect(output(page)).toContainText("nemo-default");
  await workspace(page)
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await page.evaluate(() => window.testRuntime.removeInstance("nemo-default"));
  await page
    .getByRole("button", { name: "Refresh inventory", exact: true })
    .click();
  const unavailable = runtimeTabs(page).getByRole("tab", {
    name: "Selected runtime unavailable",
    exact: true,
  });
  await expect(unavailable).toHaveAttribute("aria-selected", "true");
  await expect(unavailable).toHaveAttribute("aria-disabled", "true");
  await expect(
    page.getByText(
      "The selected runtime is no longer available. Choose another runtime to inspect its output.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(output(page)).toHaveCount(0);
  await expect
    .poll(() => callsOf(page, "disable"))
    .toEqual(["disable:nemo-default"]);
  await expectNoOutputActivity(page);
  expect(await callsOf(page, "enable")).toEqual(["enable:nemo-default"]);
  await unavailable.focus();
  await page.keyboard.press("ArrowRight");
  await expect(
    runtimeTabs(page).getByRole("tab", {
      name: "Second speech provider",
      exact: true,
    }),
  ).toBeFocused();
  await expect(output(page)).toContainText("other-speech");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
