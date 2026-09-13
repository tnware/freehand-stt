import { defineConfig } from "@playwright/test";
import base from "./playwright.config";
export default defineConfig({
  ...base,
  testMatch: "**/shortcut-recovery.spec.ts",
  outputDir: "test-results/shortcut-recovery",
  reporter: [["list"]],
  use: { ...base.use, baseURL: "http://127.0.0.1:9369" },
  webServer: {
    command: "npm run test:browser:serve -- --port 9369",
    url: "http://127.0.0.1:9369/tests/browser/app/",
    reuseExistingServer: false,
  },
});
