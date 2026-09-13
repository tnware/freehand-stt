import { defineConfig } from "@playwright/test";
import base from "../../playwright.config";
export default defineConfig({
  ...base,
  testDir: ".",
  testMatch: "platform-presentation.spec.ts",
  use: { ...base.use, baseURL: "http://127.0.0.1:9357" },
  webServer: {
    command: "npm run test:browser:serve -- --port 9357",
    url: "http://127.0.0.1:9357/tests/browser/app/",
    reuseExistingServer: false,
  },
});
