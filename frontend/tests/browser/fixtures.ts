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
    await page.goto("/tests/browser/app/");
    await expect(page.locator("#saved-connection-stt")).toHaveText("Original server");
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
