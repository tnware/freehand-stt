import { test, expect } from "@playwright/test";

test("Voice popup stays open and manual model works during delayed discovery", async ({ page }) => {
  await page.route("**/delayed-voice-fixture", route => route.fulfill({ contentType: "text/html", body: '<html><body><div id="app"></div><script type="module" src="/tests/browser/app/delayed-voice-entry.ts"></script></body></html>' }));
  await page.goto("/delayed-voice-fixture");
  const input = page.getByRole("combobox", { name: "Choose model" });
  await input.click();
  await expect(page.getByText("Loading model list…", { exact: true })).toBeVisible();
  await expect(input).toHaveAttribute("aria-expanded", "true");
  await expect(page.locator('[data-slot="combobox-content"]')).toBeVisible();
  await expect(input).toBeEnabled();
  await input.fill("fixture/manual");
  await page.getByRole("option", { name: /Use “fixture\/manual”/ }).click();
  await expect(page.getByTestId("chosen-model")).toHaveText("fixture/manual");
  await expect(page.getByText("Loading model list…", { exact: true })).toBeVisible();
  await input.click();
  await expect(input).toHaveAttribute("aria-expanded", "true");
  await page.evaluate(() => (window as unknown as { delayedMetadata: { release(): void } }).delayedMetadata.release());
  await expect(page.getByText("Loading model list…", { exact: true })).toBeHidden();
  await expect(input).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("option", { name: /fixture\/discovered/ })).toBeVisible();
  await expect(page.getByTestId("chosen-model")).toHaveText("fixture/manual");
});
