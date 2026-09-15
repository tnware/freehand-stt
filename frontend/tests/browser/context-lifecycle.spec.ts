import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

declare global {
  interface Window {
    releaseContextSettingsRequest?: () => void;
    contextOverlayCalls: string[];
  }
}

const secondary = (page: Page) => page.locator("#workbench-secondary-sidebar");
const configuration = (page: Page) =>
  secondary(page).locator('[data-pane="configuration"]');
const currentArea = (page: Page, name: string) =>
  page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name, exact: true });

test("hiding an Overlay inspector stops preview and retains its unsaved preferences", async ({
  page,
}) => {
  await page.route("**/bindings/**/internal/overlay/service.*", (route) =>
    route.fulfill({
      contentType: "application/javascript",
      body: `
        window.contextOverlayCalls = [];
        export const StartPreview = async () => window.contextOverlayCalls.push("start");
        export const StopPreview = async () => window.contextOverlayCalls.push("stop");
      `,
    }),
  );
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&setup-ready&workflows");
  await page
    .locator("#workbench-primary-sidebar")
    .getByRole("button", { name: /^Overlay/ })
    .click();
  const enabled = configuration(page).getByRole("switch", {
    name: "Show status overlay",
    exact: true,
  });
  const changed = (await enabled.getAttribute("aria-checked")) !== "true";
  await enabled.click();
  await configuration(page)
    .getByRole("button", { name: "Preview overlay", exact: true })
    .click();
  await expect
    .poll(() => page.evaluate(() => window.contextOverlayCalls))
    .toEqual(["start"]);

  await page.setViewportSize({ width: 560, height: 820 });
  await expect(secondary(page)).toBeHidden();
  await expect
    .poll(() => page.evaluate(() => window.contextOverlayCalls))
    .toEqual(["start", "stop"]);
  await page
    .getByRole("button", { name: "Toggle secondary sidebar", exact: true })
    .click();
  await expect(enabled).toHaveAttribute("aria-checked", String(changed));
  await expect(configuration(page)).toContainText("Unsaved changes");
  await configuration(page)
    .getByRole("button", { name: "Preview overlay", exact: true })
    .click();
  await expect
    .poll(() => page.evaluate(() => window.contextOverlayCalls))
    .toEqual(["start", "stop", "start"]);
  await page
    .getByRole("button", { name: "Toggle primary sidebar", exact: true })
    .click();
  await expect(secondary(page)).toBeHidden();
  await expect
    .poll(() => page.evaluate(() => window.contextOverlayCalls))
    .toEqual(["start", "stop", "start", "stop"]);
  await page
    .getByRole("button", { name: "Toggle secondary sidebar", exact: true })
    .click();
  await expect(enabled).toHaveAttribute("aria-checked", String(changed));
  await expect(
    configuration(page).getByRole("button", {
      name: "Preview overlay",
      exact: true,
    }),
  ).toBeVisible();
});

test("a delayed native request cannot reopen a closed contextual inspector", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&setup-ready&workflows");
  await openSection(page, "voice-transcription");
  await expect(configuration(page)).toBeVisible();
  await page.evaluate(() => {
    const bridge = window.testConnectionWindows;
    const take = bridge.take;
    bridge.take = () => {
      const state = take();
      return new Promise<Awaited<ReturnType<typeof take>>>((resolve) => {
        window.releaseContextSettingsRequest = () => resolve(state);
      });
    };
    void bridge.openSettings("speech", "tts");
  });
  await expect
    .poll(() =>
      page.evaluate(() => Boolean(window.releaseContextSettingsRequest)),
    )
    .toBe(true);
  await configuration(page)
    .getByRole("button", { name: "Done", exact: true })
    .click();
  await expect(configuration(page)).toHaveCount(0);
  await page.evaluate(async () => {
    window.releaseContextSettingsRequest?.();
    // Let the released bridge response and its renderer continuation settle.
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => resolve()),
    );
  });
  await expect(configuration(page)).toHaveCount(0);
  await expect(secondary(page)).toBeHidden();
  await expect(currentArea(page, "Voice transcription")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(page.locator('[data-pane="settings"]')).toHaveCount(0);
});

test("a connection confirmation blocks nested layout and native close decisions", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&setup-ready&workflows");
  await openSection(page, "server");
  await configuration(page)
    .locator("summary")
    .filter({ hasText: "Request settings" })
    .click();
  const timeout = configuration(page).locator("#file-transcription-timeout");
  await timeout.fill("75");
  await configuration(page)
    .getByRole("button", { name: "Show connections", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Add connection…", exact: true })
    .click();
  const confirmation = page.getByRole("dialog", {
    name: "Save settings before continuing?",
    exact: true,
  });
  await expect(confirmation).toBeVisible();
  await page.keyboard.press("Control+Alt+b");
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await expect(confirmation).toBeVisible();
  await page.evaluate(() => window.testConnectionWindows.requestClose());
  await expect(page.getByRole("dialog")).toHaveCount(1);
  await expect(confirmation).toBeVisible();
  await confirmation
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(configuration(page)).toBeVisible();
  await expect(timeout).toHaveValue("75");
  await expect(currentArea(page, "Audio file")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await expect(page.locator("#connection-name")).toHaveCount(0);
});

test("compact History switches from guarded settings to visible details", async ({
  page,
}) => {
  await page.setViewportSize({ width: 560, height: 820 });
  await page.goto(
    "/tests/browser/app/?main&setup-ready&workflows&history-workbench",
  );
  await openSection(page, "history");
  await expect(configuration(page)).toBeVisible();
  const retention = configuration(page).getByRole("switch", {
    name: "Keep transcript history",
    exact: true,
  });
  await expect(retention).toHaveAttribute("aria-checked", "true");
  await retention.click();
  const views = secondary(page).getByRole("group", {
    name: "History sidebar view",
    exact: true,
  });
  await views.getByRole("button", { name: "Details", exact: true }).click();
  const decision = page.getByRole("dialog", {
    name: "Save changes?",
    exact: true,
  });
  await expect(decision).toBeVisible();
  await decision
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(retention).toHaveAttribute("aria-checked", "false");
  await views.getByRole("button", { name: "Details", exact: true }).click();
  await decision.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(configuration(page)).toHaveCount(0);
  await expect(secondary(page)).toBeInViewport();
  await expect(
    secondary(page).getByRole("region", {
      name: "Run information",
      exact: true,
    }),
  ).toContainText("fixture/voice-3");
  await expect(
    views.getByRole("button", { name: "Details", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByRole("button", { name: "Toggle secondary sidebar", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect(currentArea(page, "History")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await page
    .getByRole("button", { name: "Dismiss secondary sidebar", exact: true })
    .click();
  await expect(secondary(page)).toBeHidden();
  const toggle = page.getByRole("button", {
    name: "Toggle secondary sidebar",
    exact: true,
  });
  await expect(toggle).toBeFocused();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  await page.keyboard.press("Enter");
  await expect(secondary(page)).toBeInViewport();
  await expect(
    secondary(page).getByRole("region", {
      name: "Run information",
      exact: true,
    }),
  ).toContainText("fixture/voice-3");
});
