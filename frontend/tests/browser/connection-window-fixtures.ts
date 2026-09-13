import { test, expect } from "./fixtures";
export { test, expect };
export const manager = (page: import("@playwright/test").Page) => page;
export async function currentManagerState(page: import("@playwright/test").Page) {
  return page.evaluate(() => window.testConnectionWindows.requestState);
}
export async function requestNativeClose(page: import("@playwright/test").Page) {
  await page.evaluate(() => window.testConnectionWindows.requestClose());
}
