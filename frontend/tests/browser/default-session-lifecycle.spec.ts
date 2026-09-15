import { readFileSync } from "node:fs";
import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

async function mountDefaultApp(page: Page) {
  const bindings = readFileSync(
    new URL(
      "../../bindings/github.com/tnware/freehand-stt/internal/managedruntime/manager.ts",
      import.meta.url,
    ),
    "utf8",
  );
  const methods = ["GetInstances", "GetProviders", "Start", "Stop"] as const;
  const methodIDs = new Map(
    methods.map((method) => [
      Number(
        bindings.match(
          new RegExp(`function ${method}\\([^]*?ByID\\((\\d+)`),
        )![1],
      ),
      method,
    ]),
  );
  await page.route("**/default-session-test?main", (route) =>
    route.fulfill({
      contentType: "text/html",
      body: '<!doctype html><html><head></head><body style="margin:0"></body></html>',
    }),
  );
  await page.route("**/wails/runtime", async (route) => {
    const request = route.request().postDataJSON();
    if (request?.object === 6) {
      await route.fallback();
      return;
    }
    const method = methodIDs.get(request?.args?.methodID);
    expect(
      method,
      "only selected runtime metadata/lifecycle bindings",
    ).toBeDefined();
    const result = await page.evaluate(
      ({ method, request }) =>
        window.testDefaultSessionRuntime.invoke(method!, request),
      { method, request: request.args.args?.[0] },
    );
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(result ?? null),
    });
  });
  await page.goto("/default-session-test?main");
  const app = await page.evaluateHandle(async () => {
    const path = "/tests/browser/app/default-session-lifecycle.ts";
    const { mountDefaultSessionApp } = await import(path);
    return mountDefaultSessionApp();
  });
  await page
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
  return app;
}

test("default session survives App remount with working runtime commands and one event subscription", async ({
  page,
}) => {
  const app = await mountDefaultApp(page);
  try {
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toBeEnabled();
    expect(await app.evaluate((app) => app.calls())).toEqual([
      "Stop:nemo-default",
    ]);
    await app.evaluate((app) => app.enterCredentials());
    await app.evaluate((app) => app.teardown());
    expect(await app.evaluate((app) => app.credentialsCleared())).toBe(true);

    const before = await app.evaluate((app) => app.deliveries());
    await app.evaluate((app) => app.emit("error"));
    expect(await app.evaluate((app) => app.deliveries())).toBe(before);
    expect(await app.evaluate((app) => app.state())).toBe("stopped");

    await app.evaluate((app) => app.render());
    await page
      .getByRole("button", { name: "Local runtime", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toBeEnabled();
    await app.evaluate((app) => app.emit("error"));
    expect(await app.evaluate((app) => app.deliveries())).toBe(before + 1);
    expect(await app.evaluate((app) => app.state())).toBe("error");
    await app.evaluate((app) => app.emit("stopped"));
    await page.getByRole("button", { name: "Start", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Stop", exact: true }),
    ).toBeEnabled();
    expect(await app.evaluate((app) => app.calls())).toEqual([
      "Stop:nemo-default",
      "Start:nemo-default",
    ]);
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toBeEnabled();
    expect(await app.evaluate((app) => app.calls())).toEqual([
      "Stop:nemo-default",
      "Start:nemo-default",
      "Stop:nemo-default",
    ]);
  } finally {
    await app.evaluate((app) => app.destroy());
    await app.dispose();
  }
});

test("default session resumes from page cache and disposes when its WebView leaves", async ({
  page,
}) => {
  const app = await mountDefaultApp(page);
  try {
    await page.evaluate(() =>
      window.dispatchEvent(
        new PageTransitionEvent("pagehide", { persisted: true }),
      ),
    );
    await page.evaluate(() =>
      window.dispatchEvent(
        new PageTransitionEvent("pageshow", { persisted: true }),
      ),
    );
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toBeEnabled();
    await app.evaluate((app) => app.enterCredentials());
    await page.evaluate(() =>
      window.dispatchEvent(
        new PageTransitionEvent("pagehide", { persisted: false }),
      ),
    );
    expect(await app.evaluate((app) => app.credentialsCleared())).toBe(true);
    expect(await app.evaluate((app) => app.run())).toBe(false);
    expect(await app.evaluate((app) => app.calls())).toEqual([
      "Stop:nemo-default",
    ]);
  } finally {
    await app.evaluate((app) => app.destroy());
    await app.dispose();
  }
});
