import { test, expect, requestNativeClose } from "./connection-window-fixtures";
import type { Page } from "@playwright/test";
const editor = (page: Page) => page.getByRole("region", { name: "Connection editor", exact: true });
const prompt = (page: Page) => page.getByRole("dialog", { name: "Save connection changes?", exact: true });
async function add(page: Page) {
  await page.getByRole("button", { name: "Show connections", exact: true }).click();
  await page.getByRole("button", { name: "Add connection…", exact: true }).click();
  await expect(editor(page)).toBeVisible();
  await expect(page.locator("iframe")).toHaveCount(0);
}
async function selected(page: Page, name = "Original server") {
  await expect(page.locator("#saved-connection-stt")).toHaveValue(name);
}
for (const gesture of ["Escape", "Done", "titlebar"] as const) {
  async function close(page: Page) {
    if (gesture === "Escape") await page.locator("#connection-name").press("Escape");
    else if (gesture === "titlebar") await requestNativeClose(page);
    else await page.getByRole("button", { name: "Done", exact: true }).click();
  }
  test(`${gesture}: keep editing, discard, and reopen clear only the draft`, async ({ page }) => {
    await add(page);
    await page.locator("#connection-name").fill("Unfinished server");
    await close(page);
    await expect(prompt(page)).toBeVisible();
    await prompt(page).getByRole("button", { name: "Keep editing" }).click();
    await expect(page.locator("#connection-name")).toHaveValue("Unfinished server");
    await close(page);
    await prompt(page).getByRole("button", { name: "Discard", exact: true }).click();
    await expect(editor(page)).toBeHidden();
    if (gesture === "titlebar") await page.evaluate(() => window.testConnectionWindows.openSettings("server"));
    await selected(page);
    await add(page);
    await expect(page.locator("#connection-name")).toHaveValue("");
    await expect(prompt(page)).toBeHidden();
    await close(page);
    await expect(editor(page)).toBeHidden();
  });
  test(`${gesture}: pending save keeps errors and retry, then returns selection`, async ({ page, saves }) => {
    await add(page);
    await page.locator("#connection-name").fill("Pending server");
    await page.locator("#connection-url").fill("https://fixture.example.test/v1");
    await editor(page).getByRole("button", { name: "Save and return", exact: true }).click();
    const first = await saves.waitForStart();
    if (gesture === "Done") await expect(page.getByRole("button", { name: "Done", exact: true })).toBeDisabled();
    else await close(page);
    await expect(editor(page)).toBeVisible();
    await expect(prompt(page)).toBeHidden();
    await saves.complete(first, "failure");
    await expect(editor(page).getByRole("alert")).toContainText("Fixture save failed");
    await expect(page.locator("#connection-name")).toHaveValue("Pending server");
    await editor(page).getByRole("button", { name: "Save and return", exact: true }).click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect(editor(page)).toBeHidden();
    await selected(page, "Pending server");
  });
}
test("save in dirty close prompt keeps activation through failure and retry", async ({
  page,
  saves,
}) => {
  await add(page);
  await page.locator("#connection-name").fill("Saved by prompt");
  await page.locator("#connection-url").fill("https://fixture.example.test/v1");
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await prompt(page).getByRole("button", { name: "Save", exact: true }).click();
  const first = await saves.waitForStart();
  await expect(
    prompt(page).getByRole("button", { name: "Keep editing" }),
  ).toBeDisabled();
  await expect(
    prompt(page).getByRole("button", { name: "Discard", exact: true }),
  ).toBeDisabled();
  await expect(
    prompt(page).getByRole("button", { name: "Close", exact: true }),
  ).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(prompt(page)).toBeVisible();
  await page.mouse.click(5, 5);
  await expect(prompt(page)).toBeVisible();
  await saves.complete(first, "failure");
  await expect(prompt(page).getByRole("alert")).toContainText(
    "Fixture save failed",
  );
  await prompt(page).getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await selected(page, "Saved by prompt");
});
test("external hide clears pending prompt and credential draft before reopening the same shell", async ({ page }) => {
  const document = await page.evaluateHandle(() => window.document);
  await add(page);
  await page.locator("#connection-name").fill("Stale draft");
  await requestNativeClose(page);
  await expect(prompt(page)).toBeVisible();
  await page.evaluate(() => window.testConnectionWindows.hide());
  await expect(editor(page)).toBeHidden();
  await page.evaluate(() => window.testConnectionWindows.openSettings("server"));
  await selected(page);
  await add(page);
  await expect(page.locator("#connection-name")).toHaveValue("");
  await expect(prompt(page)).toBeHidden();
  expect(await document.evaluate(original => original === window.document)).toBe(true);
});
test("reveal cannot replace an already-owned inline draft", async ({ page }) => {
  await add(page);
  await page.locator("#connection-name").fill("Owned draft");
  await page.evaluate(() => window.testConnectionWindows.open({ id: "original", purpose: "stt" as any, create: false }));
  await expect(page.locator("#connection-name")).toHaveValue("Owned draft");
  await expect(editor(page)).toHaveCount(1);
});
test("dirty settings save before adding retains failure and retries without losing edits", async ({ page, saves }) => {
  await page.locator("summary", { hasText: "Request settings" }).click();
  await page.locator("#file-transcription-timeout").fill("75");
  await page.getByRole("button", { name: "Show connections", exact: true }).click();
  await page.getByRole("button", { name: "Add connection…", exact: true }).click();
  const decision = page.getByRole("dialog", { name: "Save settings before continuing?", exact: true });
  await decision.getByRole("button", { name: "Save and continue" }).click();
  const first = await saves.waitForStart();
  await page.keyboard.press("Escape");
  await expect(decision).toBeVisible();
  await saves.complete(first, "failure");
  await expect(decision.getByRole("alert")).toContainText("Fixture save failed");
  await decision.getByRole("button", { name: "Save and continue" }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(editor(page)).toBeVisible();
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await expect(page.locator("#file-transcription-timeout")).toHaveValue("75");
});
