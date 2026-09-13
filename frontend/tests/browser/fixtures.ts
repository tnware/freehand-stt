import { test as base, expect } from "@playwright/test";
import type { SaveControl } from "./app/save-control";

type Saves = {
  waitForStart: () => Promise<number>;
  complete: (...args: Parameters<SaveControl["complete"]>) => Promise<void>;
};

export const test = base.extend<{ saves: Saves }>({
  page: async ({ page, baseURL }, use) => {
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.route("**/*", (route) =>
      new URL(route.request().url()).origin === new URL(baseURL!).origin
        ? route.continue()
        : route.abort(),
    );
    await page.route("**/bindings/**/internal/windowing/service.*", route => route.fulfill({
      contentType: "application/javascript", body: `
      const bridge = () => window.testConnectionWindows;
      export const OpenConnectionManager = request => bridge().open(request);
      export const OpenTaskConnection = (request, origin) => bridge().open(request, origin);
      export const TakeSettingsRequest = async () => bridge().take();
      export const SettingsReady = async () => bridge().ready((name, data = null) => window._wails.dispatchWailsEvent({ name, data }));
      export const SettingsVisible = async () => bridge().visible;
      export const FinishSettings = origin => bridge().finish(origin);
      export const HideSettings = () => bridge().hide();
      export const OpenSettings = section => bridge().openSettings(section);
      export const OpenTaskSettings = (section, origin) => bridge().openSettings(section, origin);
      export const ShellReady = async () => {};
      export const AboutVisible = async () => false;
      export const OpenAbout = async () => {};
      export const HideAbout = async () => {};
      `,
    }));
    await page.route("**/bindings/**/internal/buildinfo/service.*", route => route.fulfill({
      contentType: "application/javascript", body: `export const Current = async () => ({version: "fixture"});`,
    }));
    await page.goto("/tests/browser/app/");
    await expect(page.locator("#saved-connection-stt")).toHaveValue("Original server");
    await use(page);
    expect(errors, "uncaught browser errors").toEqual([]);
  },
  saves: async ({ page }, use) => {
    let last = 0;
    await use({
      waitForStart: async () => {
        last = await page.evaluate((after) => window.testSaves.waitForStart(after), last);
        return last;
      },
      complete: async (id, outcome) => {
        await page.evaluate(({ id, outcome }) => window.testSaves.complete(id, outcome), {
          id,
          outcome,
        });
      },
    });
  },
});
export { expect } from "@playwright/test";
