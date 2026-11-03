import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

test("header rename collisions preserve both rows and show inline feedback", async ({
  page,
}) => {
  await openSection(page, "connections");
  await page
    .getByRole("navigation", { name: "Saved connections" })
    .getByRole("button", { name: /Original server/ })
    .click();
  await page
    .getByText("Transcription connection options", { exact: true })
    .click();
  await page.getByRole("button", { name: "Add header", exact: true }).click();
  await page
    .getByLabel("Value for X-Custom-Header", { exact: true })
    .fill("first");
  await page.getByRole("button", { name: "Add header", exact: true }).click();
  await page
    .getByLabel("Value for X-Custom-Header-2", { exact: true })
    .fill("second");
  const names = page.getByRole("textbox", { name: "Header name", exact: true });
  await names.first().fill("x-custom-header-2");
  await names.first().press("Tab");
  await expect(page.getByRole("alert")).toContainText(
    "Both headers have been kept",
  );
  await expect(names).toHaveCount(2);
  await expect(names.first()).toHaveValue("X-Custom-Header");
  await expect(
    page.getByLabel("Value for X-Custom-Header", { exact: true }),
  ).toHaveValue("first");
  await expect(
    page.getByLabel("Value for X-Custom-Header-2", { exact: true }),
  ).toHaveValue("second");
  await names.first().fill("X-Renamed");
  await names.first().press("Tab");
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(
    page.getByLabel("Value for X-Renamed", { exact: true }),
  ).toHaveValue("first");
});

test("backend selection locks during save and unlocks after failure", async ({
  page,
  saves,
}) => {
  await openSection(page, "connections");
  await page
    .getByRole("button", { name: "Add connection", exact: true })
    .click();
  await page.locator("#connection-name").fill("Pending server");
  await page.locator("#connection-url").fill("https://fixture.example.test/v1");
  const backend = page.locator("#connection-profile");
  await expect(backend).toBeEnabled();
  const selected = await backend.innerText();
  await page
    .getByRole("button", { name: "Save connection", exact: true })
    .click();
  const pending = await saves.waitForStart();
  await expect(backend).toBeDisabled();
  await saves.complete(pending, "failure");
  await expect(backend).toBeEnabled();
  await expect(backend).toHaveText(selected);
});

test("saving a manual connection with a built-in name returns the manual entry", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await openSection(page, "connections");
  await page
    .getByRole("button", { name: "Add connection", exact: true })
    .click();
  await page.locator("#connection-name").fill("Local speech");
  await page.locator("#connection-url").fill("https://fixture.example.test/v1");
  await page
    .getByRole("button", { name: "Save connection", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  const selected = page
    .getByRole("navigation", { name: "Saved connections" })
    .locator('[aria-current="true"]');
  await expect(selected).toContainText("Local speech");
  await expect(page.locator("#connection-name")).toHaveValue("Local speech");
  await expect(page.locator("#connection-url")).toHaveValue(
    "https://fixture.example.test/v1",
  );
});
