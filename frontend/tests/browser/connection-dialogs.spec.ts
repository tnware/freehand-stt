import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";

const setup = (page: Page) =>
  page.getByRole("dialog", { name: "Add a connection for transcription", exact: true });
const decision = (page: Page) =>
  page.getByRole("dialog", { name: "Save settings before changing connections?", exact: true });
const selection = (page: Page) => page.locator("#saved-connection-stt");
async function addConnection(page: Page) {
  await selection(page).click();
  await page.getByRole("option", { name: "Add connection…", exact: true }).click();
}

for (const gesture of ["Escape", "outside click"] as const) {
  test.describe(gesture, () => {
    async function dismiss(page: Page) {
      if (gesture === "Escape") await page.keyboard.press("Escape");
      // Finish a normal pointer gesture before completing a held service request.
      else await page.mouse.click(5, 5, { delay: 50 });
    }

    test("Keep editing preserves the visible connection draft", async ({ page }) => {
      await addConnection(page);
      await page.locator("#connection-name").fill("Unfinished server");
      await dismiss(page);
      await page.getByRole("button", { name: "Keep editing", exact: true }).click();
      await expect(setup(page)).toBeVisible();
      await expect(page.locator("#connection-name")).toHaveValue("Unfinished server");
      await page.locator("#connection-name").fill("Still editable");
      await expect(page.locator("#connection-name")).toHaveValue("Still editable");
      await expect(selection(page)).toHaveText("Original server");
    });

    test("Discard clears the connection draft and preserves the selection", async ({ page }) => {
      await addConnection(page);
      await page.locator("#connection-name").fill("Unfinished server");
      await dismiss(page);
      await page.getByRole("button", { name: "Discard changes", exact: true }).click();
      await expect(setup(page)).toBeHidden();
      await expect(selection(page)).toHaveText("Original server");
      await addConnection(page);
      await expect(page.locator("#connection-name")).toHaveValue("");
    });

    test("An unchanged connection editor dismisses normally", async ({ page }) => {
      await addConnection(page);
      await expect(setup(page)).toBeVisible();
      await dismiss(page);
      await expect(setup(page)).toBeHidden();
      await expect(selection(page)).toHaveText("Original server");
    });

    test("A pending connection save retains failure and retry actions", async ({ page, saves }) => {
      await addConnection(page);
      await page.locator("#connection-name").fill("Pending server");
      await page.locator("#connection-url").fill("https://fixture.example.test/v1");
      await setup(page)
        .getByRole("button", { name: "Save and use connection", exact: true })
        .click();
      const first = await saves.waitForStart();
      await expect(setup(page).getByRole("button", { name: "Cancel", exact: true })).toBeDisabled();
      await dismiss(page);
      await expect(setup(page)).toBeVisible();
      await saves.complete(first, "failure");
      await expect(setup(page).getByRole("alert")).toContainText("Fixture save failed");
      await expect(page.locator("#connection-name")).toHaveValue("Pending server");
      await expect(selection(page)).toHaveText("Original server");
      await setup(page)
        .getByRole("button", { name: "Save and use connection", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(setup(page)).toBeHidden();
      await expect(selection(page)).toHaveText("Pending server");
    });

    test("A pending settings save retains the prompt through failure and retry", async ({
      page,
      saves,
    }) => {
      await page.locator("#transcription-timeout").fill("75");
      await expect(page.getByText("Unsaved changes", { exact: true })).toBeVisible();
      await addConnection(page);
      await decision(page).getByRole("button", { name: "Save and continue", exact: true }).click();
      const first = await saves.waitForStart();
      await expect(
        decision(page).getByRole("button", { name: "Keep editing", exact: true }),
      ).toBeDisabled();
      await dismiss(page);
      await expect(decision(page)).toBeVisible();
      await saves.complete(first, "failure");
      await expect(decision(page).getByRole("alert")).toContainText("Fixture save failed");
      await expect(page.locator("#transcription-timeout")).toHaveValue("75");
      await expect(selection(page)).toHaveText("Original server");
      await decision(page).getByRole("button", { name: "Save and continue", exact: true }).click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(setup(page)).toBeVisible();
      await setup(page).getByRole("button", { name: "Cancel", exact: true }).click();
      await expect(setup(page)).toBeHidden();
      await expect(page.locator("#transcription-timeout")).toHaveValue("75");
      await expect(page.getByText("All changes saved", { exact: true })).toBeVisible();
    });

    test("An idle save decision dismisses without losing settings edits", async ({ page }) => {
      await page.locator("#transcription-timeout").fill("75");
      await addConnection(page);
      await expect(decision(page)).toBeVisible();
      await dismiss(page);
      await expect(decision(page)).toBeHidden();
      await expect(page.locator("#transcription-timeout")).toHaveValue("75");
      await expect(page.getByText("Unsaved changes", { exact: true })).toBeVisible();
      await expect(selection(page)).toHaveText("Original server");
    });
  });
}
