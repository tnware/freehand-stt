import { test, expect } from "./fixtures";

for (const finishBeforeUnmount of [false, true]) {
  test(`App ignores pending ${finishBeforeUnmount ? "ShellReady" : "initial load"} after teardown`, async ({
    page,
  }) => {
    await page.route("**/app-lifecycle-test", (route) =>
      route.fulfill({
        contentType: "text/html",
        body: "<!doctype html><html><head></head><body></body></html>",
      }),
    );
    await page.route("**/bindings/**/internal/windowing/service.*", (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `
        const pending = window.lifecycleRequests = [];
        const request = kind => new Promise((resolve, reject) => pending.push({kind, resolve, reject}));
        export const SettingsVisible = () => request("settings");
        export const AboutVisible = () => request("about");
        export const ShellReady = () => request("ready");
      `,
      }),
    );
    await page.route("**/bindings/**/internal/buildinfo/service.*", (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `export const Current = () => new Promise((resolve, reject) => window.lifecycleRequests.push({kind:"build", resolve, reject}));`,
      }),
    );
    await page.goto("/app-lifecycle-test");
    const app = await page.evaluateHandle(async () => {
      const fixturePath = "/tests/browser/app/app-lifecycle.ts";
      const { mountLifecycleApp } = await import(fixturePath);
      return mountLifecycleApp();
    });
    await expect
      .poll(() =>
        page.evaluate(() =>
          (window as any).lifecycleRequests
            .map((request: { kind: string }) => request.kind)
            .sort(),
        ),
      )
      .toEqual(["about", "build"]);
    if (finishBeforeUnmount) {
      await app.evaluate((app) => app.finishLoad());
      await expect
        .poll(() =>
          page.evaluate(() => (window as any).lifecycleRequests.length),
        )
        .toBe(3);
    }
    await app.evaluate((app) => app.unmount());
    await page.evaluate(() => {
      for (const request of (window as any).lifecycleRequests)
        request.reject(new Error("Late fixture reply"));
    });
    if (!finishBeforeUnmount) await app.evaluate((app) => app.finishLoad());
    expect(await app.evaluate((app) => app.failures())).toBe(0);
    expect(
      await page.evaluate(
        () =>
          (window as any).lifecycleRequests.filter(
            (request: { kind: string }) => request.kind === "ready",
          ).length,
      ),
    ).toBe(finishBeforeUnmount ? 1 : 0);
    await app.dispose();
  });
}
