import { test, expect, requestNativeClose } from "./connection-window-fixtures";
test("deferred general reveal close cannot resume a hidden task", async ({
  page,
}) => {
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("speech", "tts"),
  );
  await expect(page.locator('[data-settings-section="speech"]')).toBeVisible();
  await page.evaluate(() => window.testConnectionWindows.hide());
  await expect(page.locator('[data-window="settings"]')).toHaveCount(0);
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
  expect(await page.evaluate(() => (window as any).finishedOrigins)).toEqual([
    "",
  ]);
  await page.evaluate(() => (window as any).releaseTake());
  await expect(page.locator('[data-window="settings"]')).toHaveCount(0);
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
          { id: "", purpose: "tts" as any, create: true },
          "tts",
        );
      return bridge.openSettings(
        target === "task" ? "speech" : "general",
        target === "task" ? "tts" : "",
      );
    }, target);
    if (target === "edit")
      await expect(page.locator("#connection-name")).toBeVisible();
    else
      await expect(
        page.locator(
          `[data-settings-section="${target === "task" ? "speech" : "general"}"]`,
        ),
      ).toBeVisible();
  });
}

test("cancelled close cannot survive a successful connection save", async ({
  page,
  saves,
}) => {
  await page
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

test("blank first run uses one retained Settings renderer and returns to Voice", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?blank&pickers");
  await page
    .getByRole("button", { name: "Configure transcription", exact: true })
    .click();
  const settings = page.frameLocator('iframe[data-window="settings"]');
  await expect(page.locator("iframe")).toHaveCount(1);
  await expect(page.locator("[data-settings-section]")).toHaveCount(0);
  await expect(
    settings.getByRole("tab", { name: "Voice", exact: true }),
  ).toHaveCount(0);
  await expect(settings.getByRole("tablist")).toHaveCount(0);
  await settings
    .getByRole("button", { name: "Show connections", exact: true })
    .click();
  const frame = page.frames().find((frame) => frame !== page.mainFrame())!;
  const document = await frame.evaluateHandle(() => window.document);
  await settings
    .getByRole("button", { name: "Add connection…", exact: true })
    .click();
  await settings.locator("#connection-name").fill("First server");
  await settings
    .locator("#connection-url")
    .fill("https://fixture.example.test/v1");
  await settings
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(page.locator('iframe[data-window="settings"]')).toBeVisible();
  await settings
    .getByRole("combobox", { name: "Choose model", exact: true })
    .click();
  await settings.getByRole("option", { name: /speech\/stt/ }).click();
  await settings
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(page.locator('iframe[data-window="settings"]')).toBeHidden();
  await page.getByRole("button", { name: "Finish setup", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(
    page.getByRole("region", { name: "First-run setup" }),
  ).toBeHidden();
  await expect(
    page.getByRole("tab", { name: "Voice", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  await page
    .getByRole("button", { name: "Open Settings", exact: true })
    .click();
  await expect(page.locator('iframe[data-window="settings"]')).toBeVisible();
  await expect(page.locator("iframe")).toHaveCount(1);
  expect(
    await document.evaluate((original) => original === window.document),
  ).toBe(true);
});

test("general Save stays open without workflow header; task Save and return hides", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?general");
  await expect(page.getByRole("tablist")).toHaveCount(0);
  await expect(
    page.getByRole("tab", { name: "Text to speech", exact: true }),
  ).toHaveCount(0);
  await page.locator("summary", { hasText: "Request settings" }).click();
  await page.locator("#file-transcription-timeout").fill("75");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(page.locator("#file-transcription-timeout")).toHaveValue("75");
  expect(await page.evaluate(() => window.testConnectionWindows.visible)).toBe(
    true,
  );
  await page.evaluate(() => window.testConnectionWindows.hide());
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("server", "file"),
  );
  await page.locator("summary", { hasText: "Request settings" }).click();
  await page.locator("#file-transcription-timeout").fill("80");
  await page
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect
    .poll(() => page.evaluate(() => window.testConnectionWindows.visible))
    .toBe(false);
});

test("main task remains mounted while Settings owns a guarded draft", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?main&workflows");
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
  // The boundary models a task configuration request, not navigation in Main.
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("server", "tts"),
  );
  const settings = page.frameLocator('iframe[data-window="settings"]');
  await expect(
    page.getByRole("tab", { name: "Text to speech", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  await expect(settings.getByRole("tablist")).toHaveCount(0);
  await settings.locator("summary", { hasText: "Request settings" }).click();
  await settings.locator("#file-transcription-timeout").fill("75");
  await requestNativeClose(page);
  await expect(
    settings.getByRole("dialog", { name: "Save changes?", exact: true }),
  ).toBeVisible();
  await settings
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await settings
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(page.locator('iframe[data-window="settings"]')).toBeHidden();
  await expect(
    page.getByRole("tab", { name: "Text to speech", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
});
