import { openSection } from "./context-navigation";
import { test, expect } from "./fixtures";

test("macOS general settings use login wording and ignore imported Mica", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?platform=darwin");
  await openSection(page, "general");
  await expect(
    page.getByRole("switch", { name: "Start at login", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("switch", { name: "Use Windows Mica backdrop" }),
  ).toHaveCount(0);
  await expect(page.getByText("Follow macOS", { exact: true })).toBeVisible();
  await page.locator('label[for="appearance-dark"]').click();
  await expect(page.locator("#appearance-dark")).toHaveAttribute(
    "data-state",
    "checked",
  );
});

test("macOS permissions show scoped recovery without blocking file or speech settings", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?platform=darwin");
  await openSection(page, "general");
  await expect(
    page.getByText("macOS permissions", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Allow Microphone", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Allow Microphone", exact: true })
    .click();
  await expect(page.getByText("Denied", { exact: true })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Open Microphone settings", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText(
      "Audio files and text-to-speech do not require these permissions.",
      { exact: true },
    ),
  ).toBeVisible();
});

test("Windows general settings retain their native wording and Mica control", async ({
  page,
}) => {
  await openSection(page, "general");
  await expect(
    page.getByRole("switch", { name: "Start with Windows", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("switch", { name: "Use Windows Mica backdrop" }),
  ).toBeVisible();
  await expect(
    page.getByText("macOS permissions", { exact: true }),
  ).toHaveCount(0);
});
