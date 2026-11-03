import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

const editor = (page: import("@playwright/test").Page) =>
  page.getByRole("region", { name: "Connection editor", exact: true });

test("plain management opens the sole connection and Cancel restores saved details", async ({
  page,
}) => {
  const before = await page.evaluate(() =>
    window.testConnectionWindows.settings(),
  );
  await openSection(page, "connections");
  await expect(editor(page)).toBeVisible();
  await expect(page.locator("#connection-name")).toHaveValue("Original server");
  await editor(page)
    .getByRole("button", { name: "Cancel", exact: true })
    .click();
  await expect(editor(page)).toBeVisible();
  await page.locator("#connection-name").fill("Unsaved rename");
  await editor(page)
    .getByRole("button", { name: "Cancel", exact: true })
    .click();
  const prompt = page.getByRole("dialog", {
    name: "Save connection changes?",
    exact: true,
  });
  await expect(prompt).toBeVisible();
  await prompt
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(page.locator("#connection-name")).toHaveValue("Unsaved rename");
  await editor(page)
    .getByRole("button", { name: "Cancel", exact: true })
    .click();
  await prompt.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(editor(page)).toBeVisible();
  await expect(page.locator("#connection-name")).toHaveValue("Original server");
  expect(
    await page.evaluate(() => window.testConnectionWindows.settings()),
  ).toEqual(before);
});

test("cancel a new connection keeps the catalog unchanged", async ({
  page,
}) => {
  const before = await page.evaluate(() =>
    window.testConnectionWindows.settings(),
  );
  await openSection(page, "connections");
  await page
    .getByRole("button", { name: "Add connection", exact: true })
    .click();
  await editor(page)
    .getByRole("button", { name: "Cancel", exact: true })
    .click();
  await expect(editor(page)).toBeHidden();
  expect(
    await page.evaluate(() => window.testConnectionWindows.settings()),
  ).toEqual(before);
});

test("collapsed catalog offers Show connections", async ({ page }) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  await openSection(page, "connections");
  const toggle = page.getByRole("button", {
    name: "Toggle primary sidebar",
    exact: true,
  });
  if ((await toggle.getAttribute("aria-pressed")) === "true")
    await toggle.click();
  await page
    .getByRole("button", { name: "Show connections", exact: true })
    .click();
  await expect(
    page.getByRole("navigation", { name: "Saved connections", exact: true }),
  ).toBeVisible();
});
