import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";

const panel = (page: Page) => page.locator("#workbench-bottom-panel");
const toggle = (page: Page) =>
  page.getByRole("button", { name: "Toggle bottom panel", exact: true });

for (const bottomOpen of [true, false]) {
  test(`setup preserves the ${bottomOpen ? "open" : "closed"} panel preference through navigation and completion`, async ({
    page,
    saves,
  }) => {
    await page.addInitScript((open) => {
      localStorage.setItem(
        "freehand-workbench-layout-v1",
        JSON.stringify({ bottomOpen: open, bottomSize: 42, tab: "recent" }),
      );
    }, bottomOpen);
    await page.goto("/tests/browser/app/?view=workspace&setup=first&pickers");
    await expect(
      page.getByRole("region", { name: "First-run setup", exact: true }),
    ).toBeVisible();
    await expect(panel(page)).toBeHidden();
    await expect(toggle(page)).toBeDisabled();
    await expect(toggle(page)).toHaveAttribute(
      "title",
      "The bottom panel is hidden during setup",
    );
    const saved = () =>
      page.evaluate(() =>
        JSON.parse(localStorage.getItem("freehand-workbench-layout-v1")!),
      );
    expect(await saved()).toMatchObject({ bottomOpen, bottomSize: 42 });

    const navigation = page.getByRole("navigation", {
      name: "Workspace",
      exact: true,
    });
    await navigation
      .getByRole("button", { name: "History", exact: true })
      .click();
    await expect(toggle(page)).toBeEnabled();
    if (bottomOpen) await expect(panel(page)).toBeVisible();
    else await expect(panel(page)).toBeHidden();
    await navigation
      .getByRole("button", { name: "Voice transcription", exact: true })
      .click();
    await expect(toggle(page)).toBeDisabled();
    await expect(panel(page)).toBeHidden();

    await page
      .getByRole("button", { name: "Check connection", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Finish setup", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "failure");
    await expect(panel(page)).toBeHidden();
    await expect(toggle(page)).toBeDisabled();
    await page
      .getByRole("button", { name: "Finish setup", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect(toggle(page)).toBeEnabled();
    if (bottomOpen) await expect(panel(page)).toBeVisible();
    else await expect(panel(page)).toBeHidden();
    expect(await saved()).toMatchObject({ bottomOpen, bottomSize: 42 });
  });
}

test("recoverable microphone attention keeps the history panel available", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?view=workspace&setup=microphone&pickers",
  );
  await expect(
    page.getByRole("region", { name: "Task recovery", exact: true }),
  ).toBeVisible();
  await expect(panel(page)).toBeVisible();
  await expect(toggle(page)).toBeEnabled();
});
