import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

const caption = (page: Page, name: string) =>
  page
    .getByRole("group", { name: "Window controls", exact: true })
    .getByRole("button", { name, exact: true });
const decision = (page: Page) =>
  page.getByRole("dialog", { name: "Save changes?", exact: true });
const nativeActions = (page: Page) =>
  page.evaluate(() =>
    window.testWindow.calls.filter((call) => call !== "IsMaximised"),
  );

for (const width of [1280, 560]) {
  test(`Windows caption controls fit and remain keyboard-operable at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 820 });
    await page.goto("/tests/browser/app/?main&setup-ready&workflows");
    const header = page.locator("header.title-bar");
    const controls = page.getByRole("group", {
      name: "Window controls",
      exact: true,
    });
    await expect(controls.getByRole("button")).toHaveCount(3);
    for (const name of ["Minimize window", "Maximize window", "Close window"]) {
      const button = caption(page, name);
      await expect(button).toBeInViewport();
      const bounds = await button.boundingBox();
      const contentHeight = await header.evaluate(
        (element) =>
          element.getBoundingClientRect().height -
          parseFloat(getComputedStyle(element).borderBottomWidth) -
          parseFloat(getComputedStyle(element).borderTopWidth),
      );
      expect(bounds!.height).toBe(contentHeight);
      expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(width);
    }
    const regions = await header.evaluate((element) => ({
      caption: [
        ...element.querySelectorAll(".title-identity, .title-drag-space"),
      ].map((item) =>
        getComputedStyle(item)
          .getPropertyValue("--wails-non-client-region")
          .trim(),
      ),
      interactive: [...element.querySelectorAll("button, .title-menu")].map(
        (item) =>
          getComputedStyle(item)
            .getPropertyValue("--wails-non-client-region")
            .trim(),
      ),
    }));
    expect(regions.caption.every((value) => value === "caption")).toBe(true);
    expect(regions.interactive).not.toContain("caption");
    await expect(header.locator(".pane-label, .pane-divider")).toHaveCount(0);
    await page.screenshot({
      path: info.outputPath(`windows-caption-${width}.png`),
    });

    await caption(page, "Minimize window").focus();
    await page.keyboard.press("Enter");
    await expect
      .poll(() => page.evaluate(() => window.testWindow.minimised))
      .toBe(true);
    await caption(page, "Maximize window").focus();
    await page.keyboard.press("Space");
    await expect(caption(page, "Restore window")).toBeVisible();
    await caption(page, "Restore window").press("Enter");
    await expect(caption(page, "Maximize window")).toBeVisible();
    expect(await nativeActions(page)).toEqual([
      "Minimise",
      "ToggleMaximise",
      "ToggleMaximise",
    ]);
  });

  test(`macOS reserves the native traffic-light area at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 820 });
    await page.goto(
      "/tests/browser/app/?main&setup-ready&workflows&platform=darwin",
    );
    const header = page.locator("header.title-bar.mac");
    await expect(header).toBeVisible();
    await expect(
      page.getByRole("group", { name: "Window controls", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("menubar", { name: "Application menu" }),
    ).toHaveCount(0);
    const geometry = await header.evaluate((element) => ({
      height: element.getBoundingClientRect().height,
      paddingLeft: getComputedStyle(element).paddingLeft,
      identityLeft: element
        .querySelector(".title-identity")!
        .getBoundingClientRect().left,
      interactive: [...element.querySelectorAll("button, .title-menu")].map(
        (item) => ({
          left: item.getBoundingClientRect().left,
          visible: item.getBoundingClientRect().width > 0,
          draggable: getComputedStyle(item)
            .getPropertyValue("--wails-draggable")
            .trim(),
          region: getComputedStyle(item)
            .getPropertyValue("--wails-non-client-region")
            .trim(),
        }),
      ),
      dragRegions: [
        ...element.querySelectorAll(".title-identity, .title-drag-space"),
      ].map((item) =>
        getComputedStyle(item).getPropertyValue("--wails-draggable").trim(),
      ),
    }));
    expect(geometry.height).toBe(44);
    expect(geometry.paddingLeft).toBe("96px");
    expect(geometry.identityLeft).toBeGreaterThanOrEqual(96);
    expect(geometry.dragRegions.every((value) => value === "drag")).toBe(true);
    for (const item of geometry.interactive.filter((item) => item.visible)) {
      expect(item.left).toBeGreaterThanOrEqual(96);
      expect(item.draggable).toBe("no-drag");
      expect(item.region).not.toBe("caption");
    }
    expect(await page.evaluate(() => window.testWindow.calls)).toEqual([]);
    await page.screenshot({
      path: info.outputPath(`macos-caption-${width}.png`),
    });
  });
}

test("caption state reflects startup maximization and native state events", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready&window-maximised");
  await expect(caption(page, "Restore window")).toBeVisible();
  await page.evaluate(() => window.testWindow.emit("WindowRestore", false));
  await expect(caption(page, "Maximize window")).toBeVisible();
  await page.evaluate(() => window.testWindow.emit("WindowFocus", true));
  await expect(caption(page, "Restore window")).toBeVisible();
  await page.evaluate(() => window.testWindow.emit("WindowUnMaximise", false));
  await expect(caption(page, "Maximize window")).toBeVisible();
  expect(await nativeActions(page)).toEqual([]);
});

test("failed caption action reports the error and remains recoverable", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  await expect(caption(page, "Maximize window")).toBeVisible();
  await page.evaluate(() => window.testWindow.failNext("ToggleMaximise"));
  await caption(page, "Maximize window").click();
  await expect(
    page.getByRole("complementary", { name: "Notifications" }),
  ).toContainText("Window ToggleMaximise failed in the fixture.");
  await expect(caption(page, "Maximize window")).toBeEnabled();
  expect(await page.evaluate(() => window.testWindow.maximised)).toBe(false);
  await caption(page, "Maximize window").click();
  await expect(caption(page, "Restore window")).toBeVisible();
  expect(await nativeActions(page)).toEqual([
    "ToggleMaximise",
    "ToggleMaximise",
  ]);
});

test("late native state replies cannot overwrite a newer maximize event", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  await expect(caption(page, "Maximize window")).toBeVisible();
  await page.evaluate(() => {
    window.testWindow.deferNextStateRead();
    window.testWindow.emit("WindowDidResize", false);
  });
  await expect
    .poll(() => page.evaluate(() => window.testWindow.pendingStateReads))
    .toBe(1);
  const reads = await page.evaluate(
    () =>
      window.testWindow.calls.filter((call) => call === "IsMaximised").length,
  );
  try {
    await page.evaluate(() => {
      window.testWindow.emit("WindowMaximise", true);
      window.testWindow.emit("WindowDidResize");
      window.testWindow.emit("WindowShow");
    });
    // Resize/focus bursts share one outstanding native query.
    expect(
      await page.evaluate(
        () =>
          window.testWindow.calls.filter((call) => call === "IsMaximised")
            .length,
      ),
    ).toBe(reads);
  } finally {
    await page.evaluate(() => window.testWindow.releaseStateRead());
  }
  await expect(caption(page, "Restore window")).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => window.testWindow.pendingStateReads))
    .toBe(0);
  expect(
    await page.evaluate(
      () =>
        window.testWindow.calls.filter((call) => call === "IsMaximised").length,
    ),
  ).toBe(reads + 1);
});

test("window-state reads and native subscriptions stop when the shell is unmounted", async ({
  page,
}) => {
  await page.route("**/frameless-lifecycle-test", (route) =>
    route.fulfill({
      contentType: "text/html",
      body: "<!doctype html><html><head></head><body></body></html>",
    }),
  );
  await page.goto("/frameless-lifecycle-test");
  const app = await page.evaluateHandle(async () => {
    const fixturePath = "/tests/browser/app/app-lifecycle.ts";
    const { mountLifecycleApp } = await import(fixturePath);
    (window as any)._wails.environment = { OS: "windows" };
    window.testWindow.deferNextStateRead();
    return mountLifecycleApp();
  });
  await expect(caption(page, "Maximize window")).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => window.testWindow.pendingStateReads))
    .toBe(1);
  await app.evaluate((app) => app.unmount());
  const stateResponse = page.waitForResponse((response) => {
    if (!response.url().endsWith("/wails/runtime")) return false;
    const call = response.request().postDataJSON();
    return call?.object === 6 && call.method === 14;
  });
  await page.evaluate(() => window.testWindow.releaseStateRead());
  await (await stateResponse).finished();
  await app.evaluate((app) => app.finishLoad());
  await page.evaluate(async () => {
    window.testWindow.emit("WindowMaximise", true);
    window.testWindow.emit("WindowDidResize");
    await new Promise(requestAnimationFrame);
  });
  expect(await page.evaluate(() => window.testWindow.calls)).toEqual([
    "IsMaximised",
  ]);
  expect(await app.evaluate((app) => app.failures())).toBe(0);
  await expect(
    page.getByRole("group", { name: "Window controls" }),
  ).toHaveCount(0);
  await app.dispose();
});

test("application menus support keyboard navigation and preserve draft decisions", async ({
  page,
}) => {
  await page.setViewportSize({ width: 560, height: 820 });
  await page.goto("/tests/browser/app/?main&setup-ready&workflows");
  const menu = page.getByRole("menubar", { name: "Application menu" });
  for (const name of ["File", "View", "Help"]) {
    await expect(
      menu.getByRole("menuitem", { name, exact: true }),
    ).toBeInViewport();
  }
  const view = menu.getByRole("menuitem", { name: "View", exact: true });
  await view.focus();
  await page.keyboard.press("ArrowDown");
  await expect(
    page.getByRole("menuitem", { name: /^Command palette/ }),
  ).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(view).toBeFocused();
  await page.keyboard.press("ArrowRight");
  const help = menu.getByRole("menuitem", { name: "Help", exact: true });
  await expect(help).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await expect(
    page.getByRole("menuitem", { name: "About Freehand", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(help).toBeFocused();

  await openSection(page, "audio");
  await page.locator("#max-duration").fill("90");
  await menu.getByRole("menuitem", { name: "File", exact: true }).click();
  await page
    .getByRole("menuitem", { name: "Connections", exact: true })
    .click();
  await expect(decision(page)).toBeVisible();
  await decision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(page.locator("#max-duration")).toHaveValue("90");
  await view.click();
  await page.getByRole("menuitem", { name: "History", exact: true }).click();
  await decision(page)
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(
    page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: "History", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await view.click();
  const primary = page.getByRole("menuitemcheckbox", {
    name: /^Toggle primary sidebar/,
  });
  await expect(primary).toHaveAttribute("aria-checked", "false");
  await primary.click();
  await expect(
    page.getByRole("button", { name: "Toggle primary sidebar", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await view.click();
  await expect(primary).toHaveAttribute("aria-checked", "true");
  await page.keyboard.press("Escape");
  expect(await nativeActions(page)).toEqual([]);
});

test("caption Close keeps a dirty draft through Keep editing and failed Save", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  await openSection(page, "audio");
  await page.locator("#max-duration").fill("90");
  await caption(page, "Close window").press("Enter");
  await expect(decision(page)).toBeVisible();
  expect(await nativeActions(page)).toEqual(["Close"]);
  await decision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(page.locator("#max-duration")).toHaveValue("90");
  expect(await nativeActions(page)).toEqual(["Close"]);

  await caption(page, "Close window").click();
  await decision(page)
    .getByRole("button", { name: "Save", exact: true })
    .click();
  const failedSave = await saves.waitForStart();
  expect(await nativeActions(page)).toEqual(["Close", "Close"]);
  await saves.complete(failedSave, "failure");
  await expect(decision(page).getByRole("alert")).toContainText(
    "Fixture save failed",
  );
  expect(await nativeActions(page)).toEqual(["Close", "Close"]);
  await decision(page)
    .getByRole("button", { name: "Save", exact: true })
    .click();
  const saved = await saves.waitForStart();
  expect(await nativeActions(page)).toEqual(["Close", "Close"]);
  await saves.complete(saved, "success");
  await expect
    .poll(() => nativeActions(page))
    .toEqual(["Close", "Close", "Hide"]);
  await expect(decision(page)).toHaveCount(0);
  await openSection(page, "audio");
  await expect(page.locator("#max-duration")).toHaveValue("90");
});

for (const source of ["caption", "File menu"]) {
  test(`${source} Close discards a dirty draft only after an explicit decision`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 560, height: 820 });
    await page.goto("/tests/browser/app/?main&setup-ready");
    await openSection(page, "audio");
    const saved = await page.locator("#max-duration").inputValue();
    await page.locator("#max-duration").fill("90");
    if (source === "caption") await caption(page, "Close window").click();
    else {
      await page
        .getByRole("menubar", { name: "Application menu" })
        .getByRole("menuitem", { name: "File", exact: true })
        .click();
      await page.getByRole("menuitem", { name: /^Close window/ }).click();
    }
    await expect(decision(page)).toBeVisible();
    expect(await nativeActions(page)).toEqual(["Close"]);
    await decision(page)
      .getByRole("button", { name: "Discard", exact: true })
      .click();
    await expect.poll(() => nativeActions(page)).toEqual(["Close", "Hide"]);
    await openSection(page, "audio");
    await expect(page.locator("#max-duration")).toHaveValue(saved);
  });
}
