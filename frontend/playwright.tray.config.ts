import { defineConfig } from "@playwright/test";
export default defineConfig({ testDir: "./tests/browser", testMatch: "tray-popover.spec.ts", reporter: "list", use: { baseURL: "http://127.0.0.1:9361", viewport: {width:360,height:500} }, webServer: { command: "npm run test:browser:serve -- --port 9361", url: "http://127.0.0.1:9361/tests/browser/app/", reuseExistingServer: false } });
