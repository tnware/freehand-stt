import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

const rail = (page: import("@playwright/test").Page) =>
  page.getByRole("navigation", { name: "Workspace", exact: true });

test("rail navigation resolves settings drafts before leaving and resets the task origin", async ({
  page,
}) => {
  await openSection(page, "general");
  const launch = page.getByRole("switch", {
    name: "Show window when launched",
    exact: true,
  });
  const original = await launch.getAttribute("aria-checked");
  await launch.click();
  await rail(page)
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  await expect(
    page.getByRole("dialog", { name: "Save changes?", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Keep editing", exact: true }).click();
  await expect(launch).not.toHaveAttribute("aria-checked", original!);
  await rail(page)
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  await page.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(
    rail(page).getByRole("button", { name: "Text to speech", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await rail(page)
    .getByRole("button", { name: "Settings", exact: true })
    .click();
  await expect(launch).toHaveAttribute("aria-checked", original!);
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await expect(
    rail(page).getByRole("button", { name: "Text to speech", exact: true }),
  ).toHaveAttribute("aria-current", "page");
});

test("native connection requests preserve purpose and hiding clears the credential draft", async ({
  page,
}) => {
  await page.evaluate(() =>
    window.testConnectionWindows.open(
      { id: "", purpose: "speech" as any, create: true },
      "tts",
    ),
  );
  await expect(page.locator("#connection-name")).toBeVisible();
  await page.locator("#connection-name").fill("Hidden draft");
  await page.locator("#connection-auth").click();
  await page.getByRole("option", { name: "API key", exact: true }).click();
  await page.locator("#connection-api-key").fill("fixture-transient-key");
  await page.evaluate(() =>
    window.testConnectionWindows.openSettings("general"),
  );
  await expect(page.locator("#connection-api-key")).toHaveValue(
    "fixture-transient-key",
  );
  await page.evaluate(() => window.testConnectionWindows.hide());
  await expect(page.locator("#connection-name")).toHaveCount(0);
  await expect(
    rail(page).getByRole("button", { name: "Text to speech", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await page.evaluate(() =>
    window.testConnectionWindows.open(
      { id: "", purpose: "speech" as any, create: true },
      "tts",
    ),
  );
  await expect(page.locator("#connection-name")).toHaveValue("");
  await page.locator("#connection-auth").click();
  await page.getByRole("option", { name: "API key", exact: true }).click();
  await expect(page.locator("#connection-api-key")).toHaveValue("");
  await expect(page.locator("iframe")).toHaveCount(0);
});

test("command palette reaches every setting and protects active workflow navigation", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&work-busy");
  await page.keyboard.press("Control+k");
  const input = page.getByRole("combobox", { name: "Command", exact: true });
  await expect(input).toBeFocused();
  await expect(
    page.getByRole("option", { name: "Audio file", exact: true }),
  ).toHaveCount(0);
  await input.fill("General");
  await input.press("Enter");
  await expect(
    page.getByRole("heading", { name: "General", exact: true }),
  ).toBeVisible();
  await page.keyboard.press("Control+k");
  await expect(
    page.getByRole("option", { name: "History", exact: true }),
  ).toHaveCount(2);
  await input.press("ArrowUp");
  await expect(input).toHaveAttribute(
    "aria-activedescendant",
    "command-settings:history",
  );
  await input.press("Enter");
  await expect(
    page
      .locator('[data-pane="configuration"]')
      .getByRole("heading", { name: "History", exact: true }),
  ).toBeVisible();
});

test("a delayed native request cannot reopen settings after the window is hidden", async ({
  page,
}) => {
  await page.evaluate(() => {
    const bridge = window.testConnectionWindows;
    const take = bridge.take;
    bridge.take = () => {
      const state = take();
      return new Promise((resolve) => {
        (window as any).releaseSettingsRequest = () => resolve(state);
      });
    };
    void bridge.openSettings("speech", "tts");
  });
  await expect
    .poll(() => page.evaluate(() => !!(window as any).releaseSettingsRequest))
    .toBe(true);
  await page.evaluate(() => window.testConnectionWindows.hide());
  await page.evaluate(() => (window as any).releaseSettingsRequest());
  await expect(
    page.locator(
      '[data-pane="settings"], [data-pane="configuration"], [data-pane="connections"]',
    ),
  ).toHaveCount(0);
});
