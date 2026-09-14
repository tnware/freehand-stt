import { test, expect } from "@playwright/test";

const mockOutput = [
  "\u001b[34m2026-09-14 14:06:01\u001b[0m  \u001b[32mINFO\u001b[0m  <b>runtime ready</b>",
  "\u001b[33mWARN\u001b[0m  This longer mock diagnostic line wraps inside the output panel, leaving every character readable even in a narrow window.",
].join("\r\n");

for (const theme of ["dark", "light"] as const) {
  for (const width of [920, 520]) {
    test(`output consent and reachable controls at ${width}px in ${theme} mode`, async ({
      page,
    }, testInfo) => {
      const errors: string[] = [];
      page.on("pageerror", (error) => errors.push(error.message));
      await page.route(
        "**/bindings/**/internal/managedruntime/manager.*",
        (route) =>
          route.fulfill({
            contentType: "application/javascript",
            body: `
      let enabled = false;
      export const GetInstances = async () => [{instance:{id:"test",provider:"llama-cpp"}}];
      export const GetProviders = async () => [{id:"llama-cpp",name:"llama.cpp"}];
      export const EnableProcessOutput = async () => { enabled = true; };
      export const DisableProcessOutput = async () => { enabled = false; };
      export const ClearProcessOutput = async () => {};
      export const ReadProcessOutput = async () => {
        if (!enabled) throw new Error("read before consent");
        return { enabled, chunks: [{sequence:1,timestamp:0,stream:"stderr",text:${JSON.stringify(mockOutput)}}], next:1, truncated:false };
      };
    `,
          }),
      );
      await page.route("**/bindings/**/internal/windowing/service.*", (route) =>
        route.fulfill({
          contentType: "application/javascript",
          body: `
      export const CurrentProcessOutput = async () => ({instanceID:"test"});
      export const CloseProcessOutput = async () => {};
    `,
        }),
      );
      await page.route("**/bindings/**/internal/settings/service.*", (route) =>
        route.fulfill({
          contentType: "application/javascript",
          body: `export const GetSettings = async () => ({appearanceMode:"${theme}",appearanceModeActive:"${theme}",useMica:false});`,
        }),
      );
      await page.setViewportSize({ width, height: 580 });
      await page.goto("/?window=process-output");
      const log = page.getByRole("region", {
        name: "Read-only process output",
      });
      const accept = page.getByRole("button", {
        name: "Show output",
        exact: true,
      });
      await expect(accept).toBeEnabled({ timeout: 15000 });
      if (theme === "dark")
        await expect(page.locator("html")).toHaveClass(/dark/);
      else await expect(page.locator("html")).not.toHaveClass(/dark/);
      await page.evaluate(() => document.fonts.ready);
      await expect(log).toHaveText("");
      await expect(log).toHaveAttribute("aria-describedby", "output-consent");
      await expect(
        page.getByRole("button", { name: "Follow", exact: true }),
      ).toBeDisabled();
      const before = await log.boundingBox();
      await page.screenshot({ path: testInfo.outputPath("consent.png") });
      await accept.click();
      await expect(log).toContainText("<b>runtime ready</b>");
      await expect(log.locator("b")).toHaveCount(0);
      await expect(accept).toHaveCount(0);

      expect(await log.boundingBox()).toEqual(before);
      const terminalScreen = await log.locator(".xterm-screen").boundingBox();
      expect(terminalScreen).not.toBeNull();
      expect(terminalScreen!.x + terminalScreen!.width).toBeLessThanOrEqual(
        before!.x + before!.width,
      );
      expect(terminalScreen!.y + terminalScreen!.height).toBeLessThanOrEqual(
        before!.y + before!.height,
      );
      expect(
        await log
          .locator(".xterm-rows > div")
          .evaluateAll((rows) =>
            rows.every((row) => row.scrollWidth <= row.clientWidth),
          ),
      ).toBe(true);
      const viewportBackground = await log
        .locator(".xterm-viewport")
        .evaluate((element) => getComputedStyle(element).backgroundColor);
      const outputBackground = await log.evaluate(
        (element) => getComputedStyle(element.parentElement!).backgroundColor,
      );
      expect(viewportBackground).toBe(outputBackground);
      await expect(
        page.getByRole("button", { name: "Follow", exact: true }),
      ).toBeEnabled();
      await page.screenshot({ path: testInfo.outputPath("output.png") });
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth),
      ).toBeLessThanOrEqual(width);

      const follow = page.getByRole("button", { name: "Follow", exact: true });
      await expect(follow).toHaveAttribute("aria-pressed", "true");
      await follow.click();
      await expect(follow).toHaveAttribute("aria-pressed", "false");
      await follow.click();
      await expect(follow).toHaveAttribute("aria-pressed", "true");
      const copy = page.getByRole("button", {
        name: "Copy selection",
        exact: true,
      });
      await expect(copy).toBeDisabled();

      await page
        .getByRole("searchbox", { name: "Search retained output" })
        .fill("runtime");
      // A visible control can still be obscured by its neighbouring toolbar group.
      // Real clicks cover narrow layouts where Follow used to overlap Next.
      await page.getByRole("button", { name: "Next", exact: true }).click();
      await expect(follow).toHaveAttribute("aria-pressed", "false");
      await expect(copy).toBeEnabled();
      await page.getByRole("button", { name: "Previous", exact: true }).click();
      await expect(copy).toBeEnabled();
      await copy.click({ trial: true });
      await page
        .getByRole("button", { name: "Clear", exact: true })
        .click({ trial: true });
      await page
        .getByRole("button", { name: "Close", exact: true })
        .click({ trial: true });
      expect(errors).toEqual([]);
    });
  }
}
