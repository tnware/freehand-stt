import { test, expect } from "./fixtures";

test.use({ viewport: { width: 360, height: 500 } });
const calls = (page: import("@playwright/test").Page) =>
  page.evaluate(() => (window as any).trayCalls);

test("quick view guides destination-focused recording and never offers capture commands", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=tray");
  await expect(page.getByRole("status")).toHaveText("Ready to dictate");
  await expect(
    page.getByText("Focus a text field in your app, then use your shortcut."),
  ).toBeVisible();
  await expect(
    page.getByRole("img", { name: /^Toggle recording:/ }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: /^(Start|Stop) recording$/ }),
  ).toHaveCount(0);
  await expect(page.getByRole("combobox")).toHaveCount(0);
  await page
    .getByRole("button", { name: "Close quick view", exact: true })
    .click();
  expect(await calls(page)).toEqual(["hide"]);
});

for (const state of ["recording", "working"]) {
  test(`${state} exposes safe cancellation without stopping for delivery`, async ({
    page,
  }) => {
    await page.goto(`/tests/browser/app/?view=tray&${state}`);
    await expect(
      page.getByRole("button", { name: /^(Start|Stop) recording$/ }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Copy", exact: true }),
    ).toBeDisabled();
    await page
      .getByRole("button", {
        name: state === "recording" ? "Cancel recording" : "Cancel dictation",
        exact: true,
      })
      .click();
    expect(await calls(page)).toEqual(["cancel"]);
    await expect(page.getByRole("status")).toHaveText("Ready to dictate");
  });
}

test("copy uses the retained result and navigation stays task scoped", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=tray&result");
  await page.getByRole("button", { name: "Copy", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Copied", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Open Freehand", exact: true })
    .click();
  expect(await calls(page)).toEqual([
    "copy",
    "settings:voice-transcription",
    "main",
  ]);
});

test("copy recovery and clipboard failure remain explicit", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=tray&recovery&copy-error");
  await expect(page.getByRole("status")).toHaveText("Transcript ready to copy");
  await page.getByRole("button", { name: "Copy", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("Clipboard unavailable");
  await expect(
    page.getByRole("button", { name: "Copied", exact: true }),
  ).toHaveCount(0);
  expect(await calls(page)).toEqual(["copy-pending"]);
});

test("loading, missing microphone and missing shortcut have reachable next steps", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=tray&loading");
  await expect(page.getByRole("status")).toHaveText("Loading Freehand…");
  await expect(
    page.getByRole("button", { name: "Copy", exact: true }),
  ).toBeDisabled();
  await page.evaluate(() => (window as any).trayFixture.release());
  await expect(page.getByRole("status")).toHaveText("Ready to dictate");
  await page.goto("/tests/browser/app/?view=tray&no-microphone");
  await expect(page.getByRole("status")).toHaveText("Voice setup needed");
  await page
    .getByRole("button", { name: "Open Voice settings", exact: true })
    .click();
  expect(await calls(page)).toEqual(["settings:voice-transcription"]);
  await page.goto("/tests/browser/app/?view=tray&no-shortcut");
  await expect(page.getByRole("status")).toHaveText("Set a recording shortcut");
  await page
    .getByRole("button", { name: "Configure shortcut", exact: true })
    .click();
  expect(await calls(page)).toEqual(["settings:shortcuts"]);
});

test("local runtime status follows its owner without changing runtime or model settings", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=tray&managed");
  const config = page.getByRole("region", {
    name: "Voice configuration",
    exact: true,
  });
  await expect(config).toContainText("Running");
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", { state: "stopped" }),
  );
  await expect(config).toContainText("Stopped");
  await expect(page.getByRole("status")).toHaveText("Voice setup needed");
  await config
    .getByRole("button", { name: "Manage runtime", exact: true })
    .click();
  expect(await calls(page)).toEqual(["settings:local-runtime"]);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

for (const theme of ["dark", "light"] as const) {
  test(`quick view stays readable and keeps navigation reachable in ${theme}`, async ({
    page,
  }, info) => {
    test.setTimeout(60_000);
    await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
    for (const state of [
      "idle",
      "result",
      "recording",
      "working",
      "error",
      "recovery",
      "managed",
      "no-shortcut",
    ]) {
      await page.goto(`/tests/browser/app/?view=tray&${state}&theme=${theme}`);
      await expect(page.getByRole("main")).toHaveAttribute(
        "aria-busy",
        "false",
      );
      const footer = page.getByRole("button", {
        name: "Open Freehand",
        exact: true,
      });
      await expect(footer).toBeInViewport();
      await expect(
        page.getByRole("button", { name: "Close quick view", exact: true }),
      ).toBeInViewport();
      expect(
        await page.evaluate(() => ({
          width: document.documentElement.scrollWidth,
          height: document.documentElement.scrollHeight,
        })),
      ).toEqual({ width: 360, height: 500 });
      if (state === "result") {
        const result = page.getByRole("region", {
          name: "Latest transcript",
          exact: true,
        });
        await result.focus();
        await result.press("End");
        await expect
          .poll(() => result.evaluate((el) => el.scrollTop))
          .toBeGreaterThan(0);
      }
      await page.screenshot({
        path: info.outputPath(`tray-${state}-${theme}.png`),
      });
      expect(await calls(page)).toEqual([]);
    }
  });
}

test("failed dismissal reports an error without touching capture", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=tray&hide-error");
  await page
    .getByRole("button", { name: "Close quick view", exact: true })
    .click();
  await expect(page.getByRole("alert")).toHaveText("Cannot dismiss panel");
  expect(await calls(page)).toEqual(["hide"]);
});
