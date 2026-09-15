import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";
const section = (page: import("@playwright/test").Page, id: string) =>
  page.locator(`[data-settings-section="${id}"]`);

test("global settings search finds its own options and preserves drafts", async ({
  page,
}) => {
  await openSection(page, "general");
  const search = page.getByRole("textbox", { name: "Find settings" });
  const launch = page.locator("#show-window-on-launch");
  const original = await launch.getAttribute("aria-checked");
  await launch.click();
  const edited = original === "true" ? "false" : "true";
  await search.fill("padding");
  await expect(section(page, "audio")).toHaveCount(0);
  await expect(section(page, "general")).toHaveCount(0);
  await expect(page.getByText("No matching settings.")).toBeVisible();
  await search.fill("glow");
  await search.press("Enter");
  await expect(section(page, "overlay")).toBeFocused();
  await search.fill("unfindable-setting");
  await expect(page.getByText("No matching settings.")).toBeVisible();
  await search.press("Escape");
  await expect(search).toHaveValue("");
  await section(page, "general").click();
  await expect(launch).toHaveAttribute("aria-checked", edited);
  await expect(
    page.getByText("Unsaved changes", { exact: true }),
  ).toBeVisible();
});

for (const width of [860, 520]) {
  test(`audio tuning stays available and validation reveals it at ${width}px`, async ({
    page,
    saves,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await openSection(page, "audio");
    const tuning = page.locator("details").filter({
      has: page.locator("summary", { hasText: "Speech detection tuning" }),
    });
    await expect(tuning).not.toHaveAttribute("open", "");
    await expect(
      page.getByRole("slider", { name: "Silence indicator delay" }),
    ).toBeHidden();
    await page.locator('label[for="silence-trimming"]').click();
    await expect(page.locator("#silence-trimming")).toHaveAttribute(
      "aria-checked",
      "true",
    );
    await tuning.locator("summary").click();
    const padding = page.getByRole("slider", {
      name: "Speech padding",
      exact: true,
    });
    await padding.focus();
    await padding.press("ArrowRight");
    await expect(padding).toHaveAttribute("aria-valuenow", "350");
    await tuning.locator("summary").click();
    const inspector = page.locator('[data-pane="configuration"]');
    await inspector
      .getByRole("tab", { name: "Cleanup settings", exact: true })
      .click();
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await saves.complete(await saves.waitForStart(), "invalid-speech-padding");
    await expect(
      inspector.getByRole("tab", { name: "Audio settings", exact: true }),
    ).toHaveAttribute("aria-selected", "true");
    await expect(tuning).toHaveAttribute("open", "");
    await expect(padding).toBeFocused();
    await expect(padding).toHaveAttribute("aria-valuenow", "350");
    await tuning.locator("summary").click();
    await inspector
      .getByRole("tab", { name: "Voice transcription settings", exact: true })
      .click();
    await inspector
      .getByRole("tab", { name: "Audio settings", exact: true })
      .click();
    await page.screenshot({ path: info.outputPath(`audio-${width}.png`) });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}

test("overlay has compact defaults and retains tuning when it does not apply", async ({
  page,
}, info) => {
  await page.setViewportSize({ width: 860, height: 740 });
  await page.goto("/tests/browser/app/?theme=dark");
  await openSection(page, "overlay");
  const tuning = page.locator("details").filter({
    has: page.locator("summary", { hasText: "Appearance and behavior" }),
  });
  await expect(
    page.getByRole("button", { name: "Preview overlay", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("slider", { name: "Overlay glow strength" }),
  ).toBeHidden();
  await page.screenshot({ path: info.outputPath("overlay-compact.png") });
  await tuning.locator("summary").click();
  const glow = page.getByRole("slider", { name: "Overlay glow strength" });
  await glow.focus();
  await glow.press("ArrowLeft");
  await expect(glow).toHaveAttribute("aria-valuenow", "95");
  const surface = page.getByRole("group", {
    name: "Overlay surface",
    exact: true,
  });
  await surface.getByRole("radio", { name: "Minimal", exact: true }).click();
  await expect(glow).toHaveAttribute("aria-disabled", "true");
  await surface.getByRole("radio", { name: "Glass", exact: true }).click();
  await expect(glow).not.toHaveAttribute("aria-disabled", "true");
  await expect(glow).toHaveAttribute("aria-valuenow", "95");
  await tuning.locator("summary").click();
  await tuning.locator("summary").click();
  await expect(glow).toHaveAttribute("aria-valuenow", "95");
});
