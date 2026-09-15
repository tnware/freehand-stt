import { defineConfig } from "@playwright/test";

const port = process.env.PLAYWRIGHT_PORT || "9346";
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: "./tests/browser",
  testMatch: "**/*.spec.ts",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  workers: process.env.CI ? 2 : undefined,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL,
    browserName: "chromium",
    channel:
      process.env.PLAYWRIGHT_CHANNEL ||
      (process.platform === "win32" ? "msedge" : undefined),
    viewport: { width: 1100, height: 950 },
    contextOptions: { reducedMotion: "reduce" },
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  webServer: {
    command: `npm run test:browser:serve -- --port ${port}`,
    url: `${baseURL}/tests/browser/app/`,
    reuseExistingServer:
      process.env.PLAYWRIGHT_REUSE_SERVER === "1" && !process.env.CI,
  },
});
