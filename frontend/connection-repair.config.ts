import { defineConfig } from "@playwright/test";
import base from "./playwright.config";
export default defineConfig({
  ...base,
  outputDir: "/tmp/freehand-connection-repair-results",
  reporter: "list",
  use: { ...base.use, baseURL: "http://127.0.0.1:9373" },
  webServer: {
    command: "npm run test:browser:serve -- --port 9373",
    url: "http://127.0.0.1:9373/tests/browser/app/",
    reuseExistingServer: false,
  },
});
