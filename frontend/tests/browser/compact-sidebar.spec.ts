import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";

const toggle = (page: Page) =>
  page.getByRole("button", { name: "Toggle primary sidebar", exact: true });
const sidebar = (page: Page) => page.locator("#workbench-primary-sidebar");

async function openSidebar(page: Page) {
  await toggle(page).click();
  await expect(toggle(page)).toHaveAttribute("aria-pressed", "true");
  await expect(sidebar(page)).toBeVisible();
}

async function expectClosed(page: Page) {
  await expect(toggle(page)).toHaveAttribute("aria-pressed", "false");
  await expect(sidebar(page)).toBeHidden();
  await expect(toggle(page)).toBeFocused();
}

test("compact Settings activation closes the drawer while arrow navigation stays open and drafts survive", async ({
  page,
}) => {
  await page.setViewportSize({ width: 640, height: 820 });
  await page.goto("/tests/browser/app/?main&setup-ready");
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "Settings", exact: true })
    .click();
  await openSidebar(page);
  const audio = sidebar(page).locator('[data-settings-section="general"]');
  await audio.click();
  await expectClosed(page);
  const launch = page.locator("#show-window-on-launch");
  const original = await launch.getAttribute("aria-checked");
  await launch.click();

  await openSidebar(page);
  await audio.focus();
  await audio.press("ArrowDown");
  await expect(toggle(page)).toHaveAttribute("aria-pressed", "true");
  await expect(sidebar(page)).toBeVisible();
  const selected = sidebar(page).locator(
    '[data-settings-section][aria-current="page"]',
  );
  await expect(selected).toBeFocused();
  await selected.press("Enter");
  await expectClosed(page);

  await openSidebar(page);
  const search = sidebar(page).getByRole("textbox", { name: "Find settings" });
  await search.fill("startup");
  await search.press("Enter");
  await expectClosed(page);
  await expect(launch).toBeVisible();
  await expect(launch).not.toHaveAttribute("aria-checked", original!);
});

test("compact runtime selection returns to the selected detail without starting or changing a runtime", async ({
  page,
}) => {
  await page.setViewportSize({ width: 640, height: 820 });
  await page.goto("/tests/browser/app/?main&setup-ready&runtime&runtime-ready");
  await page.evaluate(() => window.testRuntime.addSecondProvider());
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await openSidebar(page);
  await sidebar(page)
    .getByRole("button", { name: "Refresh inventory", exact: true })
    .click();
  const provider = sidebar(page)
    .getByRole("group", { name: "Runtime inventory", exact: true })
    .getByRole("button", { name: /^Second speech provider/ });
  await provider.focus();
  await provider.press("Enter");
  await expectClosed(page);
  await expect(
    page.getByRole("heading", { name: "Second speech provider", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "View output", exact: true }),
  ).toBeEnabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
