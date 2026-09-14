import { test, expect } from "@playwright/test";

test("consent reveals read-only output without moving the log viewport", async ({
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
        return { enabled, chunks: [{sequence:1,timestamp:0,stream:"stderr",text:"<b>runtime ready</b>"}], next:1, truncated:false };
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
      body: `export const GetSettings = async () => ({appearanceMode:"dark",useMica:false});`,
    }),
  );
  await page.setViewportSize({ width: 920, height: 580 });
  await page.goto("/?window=process-output");
  const log = page.getByRole("region", { name: "Read-only process output" });
  const accept = page.getByRole("button", { name: "Show output", exact: true });
  await expect(accept).toBeEnabled({ timeout: 15000 });
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
  await expect(
    page.getByRole("button", { name: "Follow", exact: true }),
  ).toBeEnabled();
  expect(errors).toEqual([]);
});
