import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";

const secondary = (page: Page) => page.locator("#workbench-secondary-sidebar");
const configuration = (page: Page) =>
  secondary(page).locator('[data-pane="configuration"]');
const toggle = (page: Page) =>
  page.getByRole("button", { name: "Toggle secondary sidebar", exact: true });
const close = (page: Page) =>
  secondary(page).getByRole("button", {
    name: "Close secondary sidebar",
    exact: true,
  });
const details = (page: Page) =>
  page.getByRole("button", { name: "Show transcript details", exact: true });
const reader = (page: Page) =>
  page.getByRole("region", { name: "Selected transcript", exact: true });
const information = (page: Page) =>
  secondary(page).getByRole("region", { name: "Run information", exact: true });
const decision = (page: Page) =>
  page.getByRole("dialog", { name: "Save changes?", exact: true });

async function navigate(page: Page, name: string) {
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name, exact: true })
    .click();
}

async function openApp(page: Page, width: number, savedSecondaryOpen = false) {
  await page.setViewportSize({ width, height: 820 });
  await page.addInitScript((secondaryOpen) => {
    localStorage.setItem(
      "freehand-workbench-layout-v1",
      JSON.stringify({ secondaryOpen }),
    );
  }, savedSecondaryOpen);
  await page.goto(
    "/tests/browser/app/?main&setup-ready&workflows&history-workbench",
  );
}

async function openVoiceAudio(page: Page) {
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  await configuration(page)
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  return configuration(page).locator("#max-duration");
}

for (const width of [1280, 560]) {
  test(`workflow right panes have one reachable close button at ${width}px`, async ({
    page,
  }) => {
    await openApp(page, width);
    for (const workflow of [
      { page: "Voice transcription", cog: "Voice settings" },
      { page: "Audio file", cog: "Audio file settings" },
      { page: "Text to speech", cog: "Text to speech settings" },
    ]) {
      await navigate(page, workflow.page);
      await page
        .getByRole("button", { name: workflow.cog, exact: true })
        .click();
      await expect(configuration(page)).toBeVisible();
      await expect(close(page)).toHaveCount(1);
      await expect(close(page)).toBeInViewport();
      await close(page).focus();
      await page.keyboard.press("Enter");
      await expect(secondary(page)).toBeHidden();
      await expect(configuration(page)).toHaveCount(0);
      await expect(toggle(page)).toBeFocused();
      await expect(toggle(page)).toHaveAttribute("aria-pressed", "false");
      await expect(
        page
          .getByRole("navigation", { name: "Workspace", exact: true })
          .getByRole("button", { name: workflow.page, exact: true }),
      ).toHaveAttribute("aria-current", "page");
    }
  });

  test(`closing a dirty right pane keeps edits until a successful Save or Discard at ${width}px`, async ({
    page,
    saves,
  }) => {
    await openApp(page, width);
    const duration = await openVoiceAudio(page);
    const original = await duration.inputValue();
    const changed = original === "140" ? "141" : "140";
    await duration.fill(changed);
    await close(page).click();
    await expect(decision(page)).toBeVisible();
    await decision(page)
      .getByRole("button", { name: "Keep editing", exact: true })
      .click();
    await expect(decision(page)).toHaveCount(0);
    await expect(secondary(page)).toBeVisible();
    await expect(duration).toHaveValue(changed);

    await close(page).click();
    await decision(page)
      .getByRole("button", { name: "Save", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "failure");
    await expect(decision(page)).toContainText(
      "Fixture save failed. Try again.",
    );
    await expect(secondary(page)).toBeVisible();
    await expect(duration).toHaveValue(changed);
    await decision(page)
      .getByRole("button", { name: "Save", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect(decision(page)).toHaveCount(0);
    await expect(secondary(page)).toBeHidden();
    await expect(toggle(page)).toBeFocused();

    await openVoiceAudio(page);
    await expect(duration).toHaveValue(changed);
    await duration.fill(original);
    await close(page).click();
    await decision(page)
      .getByRole("button", { name: "Discard", exact: true })
      .click();
    await expect(secondary(page)).toBeHidden();
    await expect(toggle(page)).toBeFocused();
    await openVoiceAudio(page);
    await expect(duration).toHaveValue(changed);
  });
}

test("History opens details at the wide breakpoint once per visit despite a saved closed preference", async ({
  page,
}) => {
  await openApp(page, 1100);
  await expect(toggle(page)).toHaveAttribute("aria-pressed", "false");
  await navigate(page, "History");
  await expect(reader(page)).toContainText("Prepare the release checklist.");
  await expect(information(page)).toContainText("fixture/voice-3");
  await expect(details(page)).toHaveText("Details");
  await expect(details(page)).toHaveAttribute("aria-expanded", "true");
  await expect(details(page)).toHaveAttribute(
    "aria-controls",
    "workbench-secondary-sidebar",
  );
  await expect(close(page)).toHaveCount(1);
  await close(page).click();
  await expect(secondary(page)).toBeHidden();
  await expect(details(page)).toHaveAttribute("aria-expanded", "false");
  await expect(toggle(page)).toBeFocused();

  await page
    .getByRole("complementary", { name: "History browser", exact: true })
    .getByRole("button", { name: /^View Audio file transcript from / })
    .click();
  await expect(reader(page)).toContainText("Confirm the microphone settings.");
  await expect(secondary(page)).toBeHidden();
  await page.evaluate(() => window.testHistory.appendVoice());
  await expect(
    page
      .getByRole("complementary", { name: "History browser", exact: true })
      .getByRole("button", { name: /^View .* transcript from / }),
  ).toHaveCount(4);
  await expect(reader(page)).toContainText("Confirm the microphone settings.");
  await expect(secondary(page)).toBeHidden();
  await expect(details(page)).toHaveAttribute("aria-expanded", "false");

  await details(page).click();
  await expect(information(page)).toContainText("fixture/file-2");
  await close(page).click();
  await navigate(page, "Voice transcription");
  await expect(secondary(page)).toBeHidden();
  await navigate(page, "History");
  await expect(information(page)).toBeVisible();
  await expect(details(page)).toHaveAttribute("aria-expanded", "true");
});

for (const width of [1099, 560]) {
  test(`History details remain hidden until explicitly opened below the wide breakpoint at ${width}px`, async ({
    page,
  }) => {
    await openApp(page, width, true);
    await navigate(page, "History");
    await expect(reader(page)).toContainText("Prepare the release checklist.");
    await expect(secondary(page)).toBeHidden();
    await expect(information(page)).toHaveCount(0);
    await expect(details(page)).toHaveAttribute("aria-expanded", "false");
    await expect(details(page)).toBeInViewport();
    await details(page).focus();
    await page.keyboard.press("Enter");
    await expect(information(page)).toContainText("fixture/voice-3");
    await expect(close(page)).toHaveCount(1);
    await expect(close(page)).toBeInViewport();
    await expect(details(page)).toHaveAttribute("aria-expanded", "true");
    await close(page).click();
    await expect(secondary(page)).toBeHidden();
    await expect(toggle(page)).toBeFocused();
    await expect(details(page)).toHaveAttribute("aria-expanded", "false");

    await details(page).click();
    await expect(information(page)).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(secondary(page)).toBeHidden();
    await expect(toggle(page)).toBeFocused();
    await details(page).click();
    await expect(information(page)).toBeVisible();
    await page
      .getByRole("button", { name: "Dismiss secondary sidebar", exact: true })
      .click();
    await expect(secondary(page)).toBeHidden();
    await expect(toggle(page)).toBeFocused();
    await navigate(page, "Voice transcription");
    await navigate(page, "History");
    await expect(secondary(page)).toBeHidden();
    await expect(details(page)).toHaveAttribute("aria-expanded", "false");
  });
}

test("History settings share one guarded close button with the details view", async ({
  page,
}) => {
  await openApp(page, 560);
  await navigate(page, "History");
  await details(page).click();
  await secondary(page)
    .getByRole("group", { name: "History sidebar view", exact: true })
    .getByRole("button", { name: "History settings", exact: true })
    .click();
  const retention = configuration(page).getByRole("switch", {
    name: "Keep transcript history",
    exact: true,
  });
  await expect(retention).toHaveAttribute("aria-checked", "true");
  await expect(close(page)).toHaveCount(1);
  await retention.click();
  await close(page).click();
  await expect(decision(page)).toBeVisible();
  await decision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(retention).toHaveAttribute("aria-checked", "false");
  await close(page).click();
  await decision(page)
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(secondary(page)).toBeHidden();
  await expect(toggle(page)).toBeFocused();
  await details(page).click();
  await expect(configuration(page)).toHaveCount(0);
  await expect(information(page)).toContainText("fixture/voice-3");
  await expect(reader(page)).toContainText("Prepare the release checklist.");
});

test("the History header Details action resolves settings drafts before showing the selected run", async ({
  page,
}) => {
  await openApp(page, 1280);
  await navigate(page, "History");
  await secondary(page)
    .getByRole("button", { name: "History settings", exact: true })
    .click();
  await configuration(page)
    .getByRole("switch", { name: "Keep transcript history", exact: true })
    .click();
  await expect(details(page)).toHaveAttribute("aria-expanded", "false");
  await details(page).click();
  await expect(decision(page)).toBeVisible();
  await decision(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(configuration(page)).toBeVisible();
  await details(page).click();
  await decision(page)
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(configuration(page)).toHaveCount(0);
  await expect(details(page)).toHaveAttribute("aria-expanded", "true");
  await expect(information(page)).toContainText("fixture/voice-3");
  await expect(secondary(page)).toBeFocused();
});
