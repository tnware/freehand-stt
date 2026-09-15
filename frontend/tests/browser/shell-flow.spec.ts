import { test, expect, requestNativeClose } from "./connection-window-fixtures";
import { openSection } from "./context-navigation";
import type { Page } from "@playwright/test";

const configuration = (page: Page) =>
  page.locator('[data-pane="configuration"]');
const connections = (page: Page) => page.locator('[data-pane="connections"]');
const configurationSurfaces = (page: Page) =>
  page.locator(
    '[data-pane="settings"], [data-pane="configuration"], [data-pane="connections"]',
  );
test("deferred general reveal close cannot resume a hidden task", async ({
  page,
}) => {
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("speech", "tts"),
  );
  await expect(
    configuration(page).getByRole("heading", {
      name: "Text to speech",
      exact: true,
    }),
  ).toBeVisible();
  await page.evaluate(() => window.testConnectionWindows.hide());
  await expect(configurationSurfaces(page)).toHaveCount(0);
  await page.evaluate(() => {
    const bridge = window.testConnectionWindows;
    const take = bridge.take;
    (window as any).finishedOrigins = [];
    const finish = bridge.finish;
    bridge.finish = async (origin) => {
      (window as any).finishedOrigins.push(origin);
      await finish(origin);
    };
    bridge.take = (() =>
      new Promise((resolve) => {
        (window as any).releaseTake = () => resolve(take());
      })) as typeof bridge.take;
  });
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("general"),
  );
  await expect
    .poll(() => page.evaluate(() => !!(window as any).releaseTake))
    .toBe(true);
  await requestNativeClose(page);
  expect(await page.evaluate(() => (window as any).finishedOrigins)).toEqual(
    [],
  );
  await page.evaluate(() => (window as any).releaseTake());
  await expect(configurationSurfaces(page)).toHaveCount(0);
});

for (const target of ["task", "general", "edit"] as const) {
  test(`clean catalog accepts external ${target} reentry`, async ({ page }) => {
    await page.evaluate(() =>
      window.testConnectionWindows.openSettings("connections"),
    );
    await expect(
      page.getByRole("button", { name: "Add connection", exact: true }),
    ).toBeVisible();
    await page.evaluate((target) => {
      const bridge = window.testConnectionWindows;
      if (target === "edit")
        return bridge.open(
          { id: "", purpose: "speech" as any, create: true },
          "tts",
        );
      return bridge.openSettings(
        target === "task" ? "speech" : "general",
        target === "task" ? "tts" : "",
      );
    }, target);
    if (target === "edit")
      await expect(page.locator("#connection-name")).toBeVisible();
    else if (target === "task")
      await expect(
        configuration(page).getByRole("heading", {
          name: "Text to speech",
          exact: true,
        }),
      ).toBeVisible();
    else
      await expect(
        page
          .locator('[data-pane="settings"]')
          .getByRole("heading", { name: "General", exact: true }),
      ).toBeVisible();
  });
}

test("cancelled close cannot survive a successful connection save", async ({
  page,
  saves,
}) => {
  await openSection(page, "server");
  await configuration(page)
    .getByRole("button", { name: "Show connections", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Add connection…", exact: true })
    .click();
  await page.locator("#connection-name").fill("Continue editing");
  await page.locator("#connection-url").fill("https://fixture.example.test/v1");
  await requestNativeClose(page);
  await page.getByRole("button", { name: "Keep editing", exact: true }).click();
  await page
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(page.locator("#saved-connection-stt")).toHaveValue(
    "Continue editing",
  );
});

test("blank first run configures Voice beside the workspace and returns to setup", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?blank&pickers");
  await page
    .getByRole("button", { name: "Configure transcription", exact: true })
    .click();
  const settings = configuration(page);
  await expect(page.locator("iframe")).toHaveCount(0);
  await expect(settings).toBeVisible();
  await expect(
    settings
      .getByRole("tablist", { name: "Context settings" })
      .getByRole("tab"),
  ).toHaveText([
    "Transcription",
    "Audio",
    "Cleanup",
    "Vocabulary",
    "Overlay",
    "Delivery",
  ]);
  await settings
    .getByRole("button", { name: "Show connections", exact: true })
    .click();
  const document = await page.evaluateHandle(() => window.document);
  await page
    .getByRole("button", { name: "Add connection…", exact: true })
    .click();
  await connections(page).locator("#connection-name").fill("First server");
  await connections(page)
    .locator("#connection-url")
    .fill("https://fixture.example.test/v1");
  await connections(page)
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(settings).toBeVisible();
  await settings
    .getByRole("combobox", { name: "Choose model", exact: true })
    .click();
  await page.getByRole("option", { name: /speech\/stt/ }).click();
  await settings.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(settings).toBeVisible();
  await settings.getByRole("button", { name: "Done", exact: true }).click();
  await expect(settings).toBeHidden();
  await page.getByRole("button", { name: "Finish setup", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(
    page.getByRole("region", { name: "First-run setup" }),
  ).toBeHidden();
  await expect(
    page.getByRole("button", { name: "Voice transcription", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await page.getByRole("button", { name: "Settings", exact: true }).click();
  await expect(page.locator('[data-pane="settings"]')).toBeVisible();
  await expect(page.locator("iframe")).toHaveCount(0);
  expect(
    await document.evaluate((original) => original === window.document),
  ).toBe(true);
});

test("general and contextual Save stay with their controls until Done", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?general");
  await openSection(page, "general");
  await expect(page.locator('[data-pane="settings"]')).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Workspace" })).toHaveCount(
    1,
  );
  const launch = page.getByRole("switch", {
    name: "Show window when launched",
    exact: true,
  });
  const initialLaunch = await launch.getAttribute("aria-checked");
  await launch.click();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(launch).not.toHaveAttribute("aria-checked", initialLaunch!);
  await expect(page.locator('[data-pane="settings"]')).toBeVisible();
  await page.evaluate(() => window.testConnectionWindows.hide());
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("server", "file"),
  );
  await configuration(page)
    .locator("summary", { hasText: "Request settings" })
    .click();
  await page.locator("#file-transcription-timeout").fill("80");
  await configuration(page)
    .getByRole("button", { name: "Save", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(configuration(page)).toBeVisible();
  await expect(page.locator("#file-transcription-timeout")).toHaveValue("80");
  await configuration(page)
    .getByRole("button", { name: "Done", exact: true })
    .click();
  await expect(configuration(page)).toBeHidden();
  await expect(
    page.getByRole("button", { name: "Audio file", exact: true }),
  ).toHaveAttribute("aria-current", "page");
});

test("the TTS draft survives a guarded visit to native-requested file options", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready&workflows");
  await page
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  const composer = page.getByRole("textbox", {
    name: "Text to speak",
    exact: true,
  });
  await composer.fill("Retain this composer draft while configuring files.");
  // A native request opens the owning workflow without destroying another draft.
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("server", "tts"),
  );
  const settings = configuration(page);
  await expect(
    page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: "Audio file", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await expect(
    settings
      .getByRole("tablist", { name: "Context settings" })
      .getByRole("tab"),
  ).toHaveText(["Transcription", "Cleanup", "Vocabulary"]);
  await settings.locator("summary", { hasText: "Request settings" }).click();
  await settings.locator("#file-transcription-timeout").fill("75");
  await requestNativeClose(page);
  await expect(
    page.getByRole("dialog", { name: "Save changes?", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Keep editing", exact: true }).click();
  await settings.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(settings).toBeVisible();
  await settings.getByRole("button", { name: "Done", exact: true }).click();
  await expect(settings).toBeHidden();
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Text to speech", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await expect(composer).toHaveValue(
    "Retain this composer draft while configuring files.",
  );
});
