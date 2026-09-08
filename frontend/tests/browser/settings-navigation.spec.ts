import { test, expect } from "./fixtures";

const content = (page: import("@playwright/test").Page) =>
  page.locator('section[aria-labelledby="settings-section-title"]').locator("..");
const section = (page: import("@playwright/test").Page, id: string) =>
  page.locator(`[data-settings-section="${id}"]`);

for (const width of [1100, 520]) {
  test.describe(`settings at ${width}px`, () => {
    test.use({ viewport: { width, height: 620 } });

    test("navigation resets the destination scroll without taking keyboard focus", async ({
      page,
    }) => {
      await section(page, "audio").click();
      await content(page).evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await expect.poll(() => content(page).evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
      await section(page, "overlay").click();
      await expect.poll(() => content(page).evaluate((el) => el.scrollTop)).toBe(0);
      await content(page).evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await section(page, "overlay").focus();
      await page.keyboard.press("ArrowDown");
      await expect(section(page, "general")).toBeFocused();
      await expect.poll(() => content(page).evaluate((el) => el.scrollTop)).toBe(0);
    });

    test("connection editing keeps save visible and Back returns to the top", async ({ page }) => {
      await section(page, "connections").click();
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      await expect(page.locator("#connection-name")).toBeInViewport();
      await expect(
        page.getByRole("button", { name: "Save connection", exact: true }),
      ).toBeInViewport();
      await content(page).evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await expect(
        page.getByRole("button", { name: "Save connection", exact: true }),
      ).toBeInViewport();
      await page.getByRole("button", { name: "Cancel", exact: true }).click();
      await expect.poll(() => content(page).evaluate((el) => el.scrollTop)).toBe(0);
      await expect(section(page, "connections")).toBeFocused();
      await page.getByRole("button", { name: "New connection", exact: true }).click();
      await expect(page.locator("#connection-name")).toBeInViewport();
      await expect(
        page.getByRole("button", { name: "Save connection", exact: true }),
      ).toBeInViewport();
      await expect(
        page.getByRole("button", { name: "Save connection", exact: true }),
      ).toBeDisabled();
    });

    test("a rejected save on History points back to Audio and keeps other edits", async ({
      page,
      saves,
    }) => {
      await page.locator("#file-transcription-timeout").fill("75");
      await section(page, "audio").click();
      await page.locator("#max-duration").fill("0");
      await section(page, "history").click();
      await page.getByRole("button", { name: "Save settings", exact: true }).click();
      await saves.complete(await saves.waitForStart(), "invalid-duration");
      await expect(section(page, "audio")).toHaveAttribute("aria-current", "page");
      await expect(page.locator("#max-duration")).toBeFocused();
      await expect(page.locator("#max-duration")).toHaveAttribute("aria-invalid", "true");
      await expect(page.locator("#max-duration")).toHaveValue("0");
      await expect(
        page
          .getByText("Enter a recording limit from 1 to 262 seconds.", {
            exact: false,
          })
          .last(),
      ).toBeVisible();
      await expect(section(page, "audio")).toHaveAccessibleName(/needs attention/i);
      await expect(page.getByText(/RuntimeError:/)).toHaveCount(0);
      await section(page, "history").click();
      await expect(page.getByRole("button", { name: "Review Audio", exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Review Audio", exact: true }).click();
      await expect(page.locator("#max-duration")).toBeFocused();
      await page.locator("#max-duration").fill("120");
      await expect(page.locator("#max-duration")).not.toHaveAttribute("aria-invalid", "true");
      await section(page, "server").click();
      await expect(page.locator("#file-transcription-timeout")).toHaveValue("75");
      await expect(page.getByText("Unsaved changes", { exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Save settings", exact: true }).click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(page.getByRole("button", { name: "Save settings", exact: true })).toBeDisabled();
      await expect(page.locator("#file-transcription-timeout")).toHaveValue("75");
    });

    test("a rejected Save and continue reveals the field rather than trapping it behind a dialog", async ({
      page,
      saves,
    }) => {
      await section(page, "audio").click();
      await page.locator("#max-duration").fill("0");
      await section(page, "server").click();
      await section(page, "connections").click();
      await page.getByRole("button", { name: "Save and continue", exact: true }).click();
      await saves.complete(await saves.waitForStart(), "invalid-duration");
      await expect(page.getByRole("dialog")).toHaveCount(0);
      await expect(page.getByRole("alertdialog")).toHaveCount(0);
      await expect(page.locator("#max-duration")).toBeFocused();
      await expect(page.locator("#max-duration")).toHaveValue("0");
      await section(page, "server").click();
      await expect(page.locator("#saved-connection-stt")).toHaveValue("Original server");
    });

    test("everyday controls keep mechanics in disclosures and omit deferred delivery", async ({
      page,
    }) => {
      await section(page, "audio").click();
      const details = page.locator("details").filter({ hasText: "Speech detection tuning" });
      await expect(details).not.toHaveAttribute("open", "");
      await details.locator("summary").click();
      await expect(
        details.getByRole("radiogroup", { name: "Voice activity detection mode" }),
      ).toBeVisible();
      await section(page, "general").click();
      await expect(page.getByText("Direct input", { exact: true })).toBeVisible();
      await expect(page.getByText("Manual copy", { exact: true })).toBeVisible();
      await expect(page.getByText("Clipboard paste", { exact: true })).toHaveCount(0);
    });
  });
}
