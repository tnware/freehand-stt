import { test, expect } from "./fixtures";

test("runtime connections have read-only details and navigate to their owners", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.locator('[data-settings-section="connections"]').click();
  await page.getByRole("button", { name: /Local speech/ }).click();
  const details = page.getByRole("region", {
    name: "Connection details",
    exact: true,
  });
  await expect(details).toBeVisible();
  await expect(details).toContainText("nemotron-3.5");
  await expect(details.locator("input")).toHaveCount(0);
  await expect(details.getByRole("switch")).toHaveCount(0);
  await expect(
    details.getByRole("button", { name: "Connection actions" }),
  ).toHaveCount(0);
  await expect(details.getByRole("button", { name: /^Save/ })).toHaveCount(0);
  await details
    .getByRole("button", {
      name: "Audio-file transcription settings",
      exact: true,
    })
    .click();
  await expect(page.locator("#saved-connection-stt")).toHaveValue(
    "Local speech",
  );
  await page
    .getByRole("button", { name: "Connection details", exact: true })
    .click();
  await details
    .getByRole("button", { name: "Manage runtime", exact: true })
    .click();
  await expect(
    page.locator('[data-settings-section="local-runtime"]'),
  ).toHaveAttribute("aria-current", "page");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("new manual connections never offer managed-target creation, while legacy aliases stay editable", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready&legacy-runtime");
  await page.locator('[data-settings-section="connections"]').click();
  await page.getByRole("button", { name: /Local speech/ }).click();
  await expect(page.locator("#connection-name")).toHaveValue("Local speech");
  await expect(page.locator("#connection-instance")).toBeVisible();
  await expect(page.locator("#connection-url")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Save connection", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("button", { name: "Add connection", exact: true })
    .click();
  await expect(page.locator("#connection-url")).toBeVisible();
  await expect(page.locator("#connection-target")).toHaveCount(0);
  await expect(page.locator("#connection-instance")).toHaveCount(0);
});
