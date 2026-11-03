import { test, expect } from "./fixtures";

test("local setup opens the recommended speech engine without installing", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?main&blank&runtime&runtime-all-providers",
  );
  await page
    .getByRole("button", { name: "Set up local speech", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "NeMo-Speech.cpp", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Install", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
