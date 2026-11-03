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

test("configured setup keeps the current choice instead of repeating recommendations", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=workspace&setup=first&pickers");
  const current = page.getByRole("region", {
    name: "Current transcription connection",
    exact: true,
  });
  await expect(current).toBeVisible();
  await expect(
    current.getByRole("button", { name: "Change", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("region", { name: "Local speech setup", exact: true }),
  ).toHaveCount(0);
  await page.getByText("Other connection options", { exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Browse local engines", exact: true }),
  ).toBeVisible();
});
