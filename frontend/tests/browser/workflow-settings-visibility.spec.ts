import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";

const area = (page: Page, name: string) =>
  page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name, exact: true });
const secondary = (page: Page) => page.locator("#workbench-secondary-sidebar");
const configuration = (page: Page) =>
  secondary(page).locator('[data-pane="configuration"]');
const toggle = (page: Page) =>
  page.getByRole("button", { name: "Toggle secondary sidebar", exact: true });
const workflows = [
  {
    page: "Voice transcription",
    heading: "Voice transcription",
    cog: "Voice settings",
  },
  {
    page: "Audio file",
    heading: "Audio-file transcription",
    cog: "Audio file settings",
  },
  {
    page: "Text to speech",
    heading: "Text to speech",
    cog: "Text to speech settings",
  },
] as const;

async function openApp(
  page: Page,
  width = 1280,
  scenario = "setup-ready&workflows",
) {
  await page.setViewportSize({ width, height: 820 });
  await page.goto(`/tests/browser/app/?main&${scenario}`);
}

async function expectConfiguration(page: Page, heading: string) {
  await expect(
    configuration(page).getByRole("heading", { name: heading, exact: true }),
  ).toBeVisible();
  await expect(toggle(page)).toHaveAttribute("aria-pressed", "true");
}

test("wide workflow configuration stays optional while History shows its details sidebar", async ({
  page,
}) => {
  await openApp(page);
  for (const workflow of workflows) {
    await area(page, workflow.page).click();
    await expect(secondary(page)).toBeHidden();
    await expect(configuration(page)).toHaveCount(0);
    await page.getByRole("button", { name: workflow.cog, exact: true }).click();
    await expectConfiguration(page, workflow.heading);
    await configuration(page)
      .getByRole("button", { name: "Close secondary sidebar", exact: true })
      .click();
    await expect(configuration(page)).toHaveCount(0);
    await expect(toggle(page)).toBeFocused();
  }
  await area(page, "History").click();
  await expect(secondary(page)).toBeVisible();
  await expect(
    secondary(page).getByRole("group", {
      name: "History sidebar view",
      exact: true,
    }),
  ).toBeVisible();
  await expect(configuration(page)).toHaveCount(0);
  await area(page, "Voice transcription").click();
  await expect(secondary(page)).toBeHidden();
});

test("workflow configuration restores an explicitly opened section across resize but never opens on widening alone", async ({
  page,
}) => {
  await openApp(page, 560);
  await expect(secondary(page)).toBeHidden();
  await page.setViewportSize({ width: 1280, height: 820 });
  await expect(secondary(page)).toBeHidden();
  await toggle(page).click();
  await configuration(page)
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  await expectConfiguration(page, "Audio");
  await page.setViewportSize({ width: 560, height: 820 });
  await expect(secondary(page)).toBeHidden();
  await page.setViewportSize({ width: 1280, height: 820 });
  await expectConfiguration(page, "Audio");
  await toggle(page).click();
  await expect(configuration(page)).toHaveCount(0);
  await area(page, "Audio file").click();
  await area(page, "Voice transcription").click();
  await expect(secondary(page)).toBeHidden();
  await toggle(page).click();
  await expectConfiguration(page, "Voice transcription");
});

test("clean open configuration permits runtime controls and dirty edits keep the navigation guard", async ({
  page,
}) => {
  await openApp(page, 1280, "runtime&runtime-ready");
  await expect(secondary(page)).toBeHidden();
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  await expectConfiguration(page, "Voice transcription");
  const primary = page.locator("#workbench-primary-sidebar");
  await expect(
    primary.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
  await primary.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(
    primary.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  await primary.getByRole("button", { name: "Start", exact: true }).click();
  await expect(
    primary.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
    "Start:nemo-default",
  ]);

  await configuration(page)
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  await configuration(page).locator("#max-duration").fill("90");
  await expect(
    primary.getByRole("button", { name: "Stop", exact: true }),
  ).toBeDisabled();
  await area(page, "Audio file").click();
  const decision = page.getByRole("dialog", {
    name: "Save changes?",
    exact: true,
  });
  await expect(decision).toBeVisible();
  await decision
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(configuration(page).locator("#max-duration")).toHaveValue("90");
  await expect(area(page, "Voice transcription")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await area(page, "Audio file").click();
  await decision.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(secondary(page)).toBeHidden();
  await expect(
    primary.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
});
