import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";
import { openSection } from "./context-navigation";

const configuration = (page: Page) =>
  page.locator('[data-pane="configuration"]');
const content = (page: Page) =>
  page
    .locator('section[aria-labelledby="settings-section-title"]')
    .locator("..");
const contextTab = (page: Page, label: string) =>
  configuration(page).getByRole("tab", {
    name: `${label} settings`,
    exact: true,
  });
const changes = (page: Page) =>
  page.getByRole("dialog", { name: "Save changes?", exact: true });

async function openSidebar(page: Page) {
  const toggle = page.getByRole("button", {
    name: "Toggle primary sidebar",
    exact: true,
  });
  await expect(toggle).toBeVisible();
  if ((await toggle.getAttribute("aria-pressed")) === "false")
    await toggle.click();
}

async function requestOptions(page: Page) {
  const details = configuration(page)
    .locator("details")
    .filter({
      has: page.locator("summary", { hasText: "Request settings" }),
    });
  if (!(await details.evaluate((element: HTMLDetailsElement) => element.open)))
    await details.locator("summary").click();
  return details;
}

for (const width of [1100, 520]) {
  test.describe(`configuration at ${width}px`, () => {
    test.use({ viewport: { width, height: 620 } });
    test.beforeEach(async ({ page }) => {
      await page.goto("/tests/browser/app/?workflows&setup-ready");
    });

    test("context tabs reset destination scroll and retain keyboard focus", async ({
      page,
    }) => {
      await openSection(page, "audio");
      await content(page).evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await expect
        .poll(() => content(page).evaluate((el) => el.scrollTop))
        .toBeGreaterThan(0);
      await contextTab(page, "Cleanup").click();
      await expect
        .poll(() => content(page).evaluate((el) => el.scrollTop))
        .toBe(0);
      await content(page).evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await contextTab(page, "Cleanup").focus();
      await page.keyboard.press("Home");
      await expect(contextTab(page, "Voice transcription")).toBeFocused();
      await expect(contextTab(page, "Voice transcription")).toHaveAttribute(
        "aria-selected",
        "true",
      );
      await expect
        .poll(() => content(page).evaluate((el) => el.scrollTop))
        .toBe(0);
    });

    test("Connections keeps Save visible and resets editor scroll when creating another connection", async ({
      page,
    }) => {
      await openSection(page, "connections");
      await expect(
        page
          .getByRole("navigation", { name: "Workspace", exact: true })
          .getByRole("button", { name: "Connections", exact: true }),
      ).toHaveAttribute("aria-current", "page");
      await openSidebar(page);
      await page
        .getByRole("navigation", { name: "Saved connections", exact: true })
        .getByRole("button", { name: /Original server/ })
        .click();
      const editor = page.getByRole("region", {
        name: "Connection editor",
        exact: true,
      });
      const fields = editor.locator(".connection-fields");
      const save = editor.getByRole("button", {
        name: "Save connection",
        exact: true,
      });
      await expect(editor.locator("#connection-name")).toBeInViewport();
      await expect(save).toBeInViewport();
      await fields.evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await expect
        .poll(() => fields.evaluate((el) => el.scrollTop))
        .toBeGreaterThan(0);
      await expect(save).toBeInViewport();
      await editor.getByRole("button", { name: "Cancel", exact: true }).click();
      await expect(editor).toBeHidden();
      await page
        .getByRole("button", { name: "Create connection", exact: true })
        .click();
      await expect.poll(() => fields.evaluate((el) => el.scrollTop)).toBe(0);
      await expect(editor.locator("#connection-name")).toBeInViewport();
      await expect(save).toBeInViewport();
      await expect(save).toBeDisabled();
      await page
        .locator('[data-pane="connections"]')
        .getByRole("button", { name: "Done", exact: true })
        .click();
      await expect(page.locator('[data-pane="connections"]')).toHaveCount(0);
      await expect(page.locator("iframe")).toHaveCount(0);
    });

    test("a rejected save from Cleanup reveals Audio and retains other Voice edits", async ({
      page,
      saves,
    }) => {
      await openSection(page, "voice-transcription");
      await requestOptions(page);
      await page.locator("#voice-timeout").fill("75");
      await contextTab(page, "Audio").click();
      await page.locator("#max-duration").fill("0");
      await contextTab(page, "Cleanup").click();
      await configuration(page)
        .getByRole("button", { name: "Save", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "invalid-duration");
      await expect(contextTab(page, "Audio")).toHaveAttribute(
        "aria-selected",
        "true",
      );
      await expect(page.locator("#max-duration")).toBeFocused();
      await expect(page.locator("#max-duration")).toHaveAttribute(
        "aria-invalid",
        "true",
      );
      await expect(page.locator("#max-duration")).toHaveValue("0");
      await expect(
        configuration(page)
          .getByText("Enter a recording limit from 1 to 262 seconds.", {
            exact: false,
          })
          .last(),
      ).toBeVisible();
      await expect(page.getByText(/RuntimeError:/)).toHaveCount(0);
      await contextTab(page, "Cleanup").click();
      await expect(
        configuration(page).getByRole("button", {
          name: "Review Audio",
          exact: true,
        }),
      ).toBeVisible();
      await configuration(page)
        .getByRole("button", { name: "Review Audio", exact: true })
        .click();
      await expect(page.locator("#max-duration")).toBeFocused();
      await page.locator("#max-duration").fill("120");
      await expect(page.locator("#max-duration")).not.toHaveAttribute(
        "aria-invalid",
        "true",
      );
      await contextTab(page, "Voice transcription").click();
      await requestOptions(page);
      await expect(page.locator("#voice-timeout")).toHaveValue("75");
      await expect(
        configuration(page).getByText("Unsaved changes", { exact: true }),
      ).toBeVisible();
      await configuration(page)
        .getByRole("button", { name: "Save", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(
        configuration(page).getByRole("button", { name: "Save", exact: true }),
      ).toBeDisabled();
      await configuration(page)
        .getByRole("button", { name: "Done", exact: true })
        .click();
      await expect(configuration(page)).toHaveCount(0);
      await openSection(page, "voice-transcription");
      await requestOptions(page);
      await expect(page.locator("#voice-timeout")).toHaveValue("75");
    });

    test("a rejected Save and continue reveals the field without trapping it behind a dialog", async ({
      page,
      saves,
    }) => {
      await openSection(page, "audio");
      await page.locator("#max-duration").fill("0");
      await contextTab(page, "Voice transcription").click();
      await configuration(page)
        .getByRole("button", { name: "Show connections", exact: true })
        .click();
      await page
        .getByRole("button", { name: "Add connection…", exact: true })
        .click();
      await page
        .getByRole("button", { name: "Save and continue", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "invalid-duration");
      await expect(page.getByRole("dialog")).toHaveCount(0);
      await expect(page.getByRole("alertdialog")).toHaveCount(0);
      await expect(page.locator("#max-duration")).toBeFocused();
      await expect(page.locator("#max-duration")).toHaveValue("0");
      await contextTab(page, "Voice transcription").click();
      await expect(
        configuration(page).getByRole("combobox", {
          name: "Choose connection",
          exact: true,
        }),
      ).toHaveValue("Original server");
      await expect(page.locator('[data-pane="connections"]')).toHaveCount(0);
    });

    test("crossing configuration areas guards drafts through Keep editing, failed Save, retry, and Discard", async ({
      page,
      saves,
    }) => {
      await openSection(page, "audio");
      await page.locator("#max-duration").fill("140");
      await openSection(page, "general");
      await changes(page)
        .getByRole("button", { name: "Keep editing", exact: true })
        .click();
      await expect(page.locator("#max-duration")).toHaveValue("140");
      await expect(configuration(page)).toBeVisible();
      await openSection(page, "general");
      await changes(page)
        .getByRole("button", { name: "Save", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "failure");
      await expect(changes(page)).toContainText(
        "Fixture save failed. Try again.",
      );
      await expect(page.locator("#max-duration")).toHaveValue("140");
      await expect(page.locator('[data-pane="settings"]')).toHaveCount(0);
      await changes(page)
        .getByRole("button", { name: "Save", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(page.locator('[data-pane="settings"]')).toBeVisible();
      await expect(configuration(page)).toHaveCount(0);
      await openSection(page, "audio");
      await expect(page.locator("#max-duration")).toHaveValue("140");
      await page.locator("#max-duration").fill("160");
      await openSection(page, "connections");
      await changes(page)
        .getByRole("button", { name: "Discard", exact: true })
        .click();
      await expect(page.locator('[data-pane="connections"]')).toBeVisible();
      await openSection(page, "audio");
      await expect(page.locator("#max-duration")).toHaveValue("140");
    });

    test("everyday controls keep mechanics in disclosures and omit deferred delivery", async ({
      page,
    }) => {
      await openSection(page, "audio");
      const details = configuration(page)
        .locator("details")
        .filter({ hasText: "Speech detection tuning" });
      await expect(details).not.toHaveAttribute("open", "");
      await details.locator("summary").click();
      await expect(
        details.getByRole("radiogroup", {
          name: "Voice activity detection mode",
        }),
      ).toBeVisible();
      await openSection(page, "general");
      await expect(
        page.getByText("Direct input", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Manual copy", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText("Clipboard paste", { exact: true }),
      ).toHaveCount(0);
    });
  });
}
