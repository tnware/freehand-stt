import { test as base, expect } from "./fixtures";

export const test = base.extend({
  page: async ({ page }, use) => {
    await page.route("**/bindings/**/internal/windowing/service.*", (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `const bridge = () => window.parent.testConnectionWindows;
        export const OpenConnectionManager = (request) => bridge().open(request);
        export const CurrentConnectionManager = async () => JSON.parse(JSON.stringify(bridge().requestState));
        export const HideConnectionManager = () => bridge().hide();
        export const OpenSettings = (section) => bridge().openSettings(section);`,
      }),
    );
    await page.route("**/bindings/**/internal/settings/service.*", (route) => {
      if (route.request().frame() === page.mainFrame()) return route.continue();
      return route.fulfill({
        contentType: "application/javascript",
        body: `
        export const GetSettings = async () => window.parent.testConnectionWindows.settings();
        // Wails serializes request DTOs before crossing the WebView boundary.
        export const SaveSettings = (request) => window.parent.testConnectionWindows.save(JSON.parse(JSON.stringify(request)));
      `,
      });
    });
    await page.route("**/connection-manager-fixture", (route) =>
      route.fulfill({
        contentType: "text/html",
        body: '<div id="app"></div><script type="module" src="/tests/browser/app/connection-manager-entry.ts"></script>',
      }),
    );
    // Reload after installing the bridge routes; never invoke a native window.
    await page.reload();
    await expect(page.locator("#saved-connection-stt")).toHaveValue(
      "Original server",
    );
    await use(page);
  },
});
export { expect };
export const manager = (page: import("@playwright/test").Page) =>
  page.frameLocator('iframe[title="Connections window"]');

// Exercise the routed generated service, including while the WebView is hidden.
export async function currentManagerState(
  page: import("@playwright/test").Page,
) {
  return manager(page)
    .locator("body")
    .evaluate(async (_body, servicePath) => {
      const service = await import(/* @vite-ignore */ servicePath);
      return service.CurrentConnectionManager();
    }, "/bindings/github.com/tnware/freehand-stt/internal/windowing/service.js");
}

// Browser proxy for the cancelled native titlebar WindowClosing hook, not the
// in-content Close button or Escape. Deliver through the real Wails event bus.
export async function requestNativeClose(
  page: import("@playwright/test").Page,
) {
  await page.evaluate(() => window.testConnectionWindows.requestClose());
}
