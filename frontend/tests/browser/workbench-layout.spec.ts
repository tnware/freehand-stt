import type { Page } from "@playwright/test";
import { test, expect } from "./fixtures";
import { installOutputFixture } from "./runtime-output-fixtures";

const control = (
  page: Page,
  area: "primary sidebar" | "bottom panel" | "secondary sidebar",
) => page.getByRole("button", { name: `Toggle ${area}`, exact: true });
const primary = (page: Page) => page.locator("#workbench-primary-sidebar");
const bottom = (page: Page) => page.locator("#workbench-bottom-panel");
const secondary = (page: Page) => page.locator("#workbench-secondary-sidebar");
const reader = (page: Page) =>
  page.getByRole("region", { name: "Selected transcript", exact: true });

async function navigate(page: Page, name: string) {
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name, exact: true })
    .click();
}

async function openHistory(page: Page, runtime = false, platform = "windows") {
  await page.goto(
    `/tests/browser/app/?main&setup-ready&history-workbench&platform=${platform}${runtime ? "&runtime&runtime-ready" : ""}`,
  );
  await navigate(page, "History");
  await expect(reader(page)).toContainText("Prepare the release checklist.");
}

async function show(
  page: Page,
  area: "primary sidebar" | "bottom panel" | "secondary sidebar",
) {
  const button = control(page, area);
  if ((await button.getAttribute("aria-pressed")) === "false")
    await button.click();
  await expect(button).toHaveAttribute("aria-pressed", "true");
}

test("titlebar layout controls hide and restore the primary sidebar across workspace areas", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await openHistory(page);
  await expect(
    page.getByRole("group", { name: "Workspace layout" }).getByRole("button"),
  ).toHaveCount(3);
  await show(page, "primary sidebar");
  await expect(control(page, "primary sidebar")).toHaveAttribute(
    "aria-controls",
    "workbench-primary-sidebar",
  );
  await expect(control(page, "primary sidebar")).toHaveAttribute(
    "title",
    "Hide primary sidebar",
  );
  await primary(page)
    .getByRole("searchbox", { name: "Search history" })
    .focus();
  await control(page, "primary sidebar").click();
  await expect(primary(page)).toBeHidden();
  await expect(control(page, "primary sidebar")).toBeFocused();
  await expect(control(page, "primary sidebar")).toHaveAttribute(
    "title",
    "Show primary sidebar",
  );
  await navigate(page, "Settings");
  await expect(primary(page)).toBeHidden();
  await show(page, "primary sidebar");
  await expect(
    primary(page).getByRole("navigation", { name: "Settings sections" }),
  ).toBeVisible();
  await control(page, "primary sidebar").click();
  await page.reload();
  await expect(control(page, "primary sidebar")).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(primary(page)).toBeHidden();
});

test("the bottom panel keeps its selected tab while switching areas and after reopening", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await openHistory(page, true);
  await show(page, "bottom panel");
  const diagnostics = bottom(page).getByRole("tab", {
    name: "Diagnostics",
    exact: true,
  });
  await diagnostics.click();
  const workflow = bottom(page).getByRole("button", {
    name: "Workflow",
    exact: true,
  });
  await workflow.click();
  await page.getByRole("option", { name: "Audio file", exact: true }).click();
  for (const area of [
    "Settings",
    "Local runtime",
    "Voice transcription",
    "History",
  ]) {
    await navigate(page, area);
    if (area === "Settings") {
      await expect(bottom(page)).toHaveCount(0);
      await expect(control(page, "bottom panel")).toBeDisabled();
      continue;
    }
    await expect(bottom(page)).toBeVisible();
    await expect(diagnostics).toHaveAttribute("aria-selected", "true");
    await expect(workflow).toContainText("Audio file");
  }
  await control(page, "bottom panel").click();
  await expect(bottom(page)).toBeHidden();
  await expect(control(page, "bottom panel")).toBeFocused();
  await show(page, "bottom panel");
  await expect(diagnostics).toHaveAttribute("aria-selected", "true");
  await expect(workflow).toContainText("Audio file");
  await page.reload();
  await expect(bottom(page)).toBeVisible();
  await expect(diagnostics).toHaveAttribute("aria-selected", "true");
  await page.setViewportSize({ width: 1280, height: 520 });
  await expect(control(page, "bottom panel")).toBeDisabled();
  await expect(control(page, "bottom panel")).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(bottom(page)).toBeHidden();
  await page.setViewportSize({ width: 1280, height: 820 });
  await expect(control(page, "bottom panel")).toBeEnabled();
  await expect(bottom(page)).toBeVisible();
  await expect(diagnostics).toHaveAttribute("aria-selected", "true");
});

test("History details reopen on each visit and restore the visit preference after narrowing", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await openHistory(page);
  await show(page, "secondary sidebar");
  await expect(
    secondary(page).getByRole("region", { name: "Run information" }),
  ).toContainText("fixture/voice-3");
  const transcriptBounds = (await reader(page).boundingBox())!;
  expect((await secondary(page).boundingBox())!.x).toBeGreaterThan(
    transcriptBounds.x,
  );
  await control(page, "secondary sidebar").click();
  await expect(secondary(page)).toBeHidden();
  await navigate(page, "Voice transcription");
  await expect(control(page, "secondary sidebar")).toBeEnabled();
  await expect(secondary(page)).toBeHidden();
  await navigate(page, "History");
  await expect(secondary(page)).toBeVisible();
  await page.setViewportSize({ width: 1000, height: 820 });
  await expect(control(page, "secondary sidebar")).toBeEnabled();
  await expect(control(page, "secondary sidebar")).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(secondary(page)).toBeHidden();
  await expect(
    page.getByRole("region", { name: "Run information" }),
  ).toHaveCount(0);
  await expect(reader(page)).toBeInViewport();
  await show(page, "secondary sidebar");
  await expect(
    secondary(page).getByRole("region", { name: "Run information" }),
  ).toBeVisible();
  const overlay = (await secondary(page).boundingBox())!;
  expect(overlay.width).toBeLessThanOrEqual(420);
  expect(overlay.x + overlay.width).toBeCloseTo(1000, 0);
  await page.keyboard.press("Escape");
  await expect(secondary(page)).toBeHidden();
  await expect(control(page, "secondary sidebar")).toBeFocused();
  await page.setViewportSize({ width: 1280, height: 820 });
  await expect(secondary(page)).toBeVisible();
  await expect(control(page, "secondary sidebar")).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  expect((await secondary(page).boundingBox())!.x).toBeGreaterThan(
    (await reader(page).boundingBox())!.x,
  );
});

test("compact windows open the primary sidebar on demand without stacking hidden details", async ({
  page,
}) => {
  await page.setViewportSize({ width: 640, height: 740 });
  await openHistory(page);
  await expect(control(page, "primary sidebar")).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(primary(page)).toBeHidden();
  await expect(control(page, "secondary sidebar")).toBeEnabled();
  await expect(secondary(page)).toBeHidden();
  await show(page, "primary sidebar");
  await expect(
    primary(page).getByRole("complementary", { name: "History browser" }),
  ).toBeVisible();
  await control(page, "primary sidebar").click();
  await expect(primary(page)).toBeHidden();
  await expect(control(page, "primary sidebar")).toBeFocused();
  await expect(reader(page)).toBeInViewport();
});

test("runtime output keeps its explicit target across workspace navigation", async ({
  page,
}) => {
  await installOutputFixture(page);
  await page.setViewportSize({ width: 1280, height: 820 });
  await openHistory(page, true);
  await page.evaluate(() => window.testRuntime.addSecondProvider());
  await show(page, "bottom panel");
  const output = bottom(page).getByRole("tab", {
    name: "Runtime output",
    exact: true,
  });
  await output.click();
  const runtime = bottom(page)
    .getByRole("tablist", { name: "Output runtimes", exact: true })
    .getByRole("tab", { name: "Second speech provider", exact: true });
  await runtime.click();
  for (const area of [
    "Local runtime",
    "Voice transcription",
    "Settings",
    "History",
  ]) {
    await navigate(page, area);
    if (area === "Settings") {
      await expect(bottom(page)).toHaveCount(0);
      await expect(control(page, "bottom panel")).toBeDisabled();
      continue;
    }
    await expect(output).toHaveAttribute("aria-selected", "true");
    await expect(runtime).toHaveAttribute("aria-selected", "true");
    await expect(
      bottom(page).getByRole("region", {
        name: "Read-only process output",
        exact: true,
      }),
    ).toContainText("other-speech");
  }
  await control(page, "bottom panel").click();
  await show(page, "bottom panel");
  await expect(runtime).toHaveAttribute("aria-selected", "true");
  await expect(
    bottom(page).getByRole("region", {
      name: "Read-only process output",
      exact: true,
    }),
  ).toContainText("other-speech");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("thin layout dividers resize with the keyboard and retain their proportions", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await openHistory(page);
  await show(page, "bottom panel");
  await show(page, "secondary sidebar");
  const horizontal = page.getByRole("separator", {
    name: "Resize editor and bottom panel",
    exact: true,
  });
  const vertical = page.getByRole("separator", {
    name: "Resize editor and secondary sidebar",
    exact: true,
  });
  expect((await horizontal.boundingBox())!.height).toBeLessThanOrEqual(1);
  expect((await vertical.boundingBox())!.width).toBeLessThanOrEqual(1);
  await expect(horizontal).toHaveAttribute("aria-orientation", "horizontal");
  await expect(vertical).toHaveAttribute("aria-orientation", "vertical");
  const bottomBefore = (await bottom(page).boundingBox())!.height;
  await horizontal.focus();
  await horizontal.press("ArrowUp");
  await expect
    .poll(async () => (await bottom(page).boundingBox())!.height)
    .toBeGreaterThan(bottomBefore);
  const readerBefore = (await reader(page).boundingBox())!.width;
  await vertical.focus();
  await vertical.press("ArrowLeft");
  await expect
    .poll(async () => (await reader(page).boundingBox())!.width)
    .toBeLessThan(readerBefore);
  const bottomSize = await horizontal.getAttribute("aria-valuenow");
  const secondarySize = await vertical.getAttribute("aria-valuenow");
  await navigate(page, "Voice transcription");
  await navigate(page, "History");
  await expect(horizontal).toHaveAttribute("aria-valuenow", bottomSize!);
  await expect(vertical).toHaveAttribute("aria-valuenow", secondarySize!);
  await page.reload();
  await navigate(page, "History");
  await expect(horizontal).toHaveAttribute("aria-valuenow", bottomSize!);
  await expect(vertical).toHaveAttribute("aria-valuenow", secondarySize!);
  await vertical.focus();
  await page.setViewportSize({ width: 1000, height: 820 });
  await expect(vertical).toHaveCount(0);
  await expect(control(page, "secondary sidebar")).toBeFocused();
  await horizontal.focus();
  await page.setViewportSize({ width: 1000, height: 520 });
  await expect(horizontal).toHaveCount(0);
  await expect(control(page, "bottom panel")).toBeDisabled();
  await expect(control(page, "primary sidebar")).toBeFocused();
});

for (const platform of ["windows", "darwin"]) {
  test(`workspace shortcuts use the ${platform} command modifier and restore focus`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await openHistory(page, false, platform);
    const modifier = platform === "darwin" ? "Meta" : "Control";
    const otherModifier = platform === "darwin" ? "Control" : "Meta";
    await primary(page)
      .getByRole("searchbox", { name: "Search history" })
      .focus();
    await page.keyboard.press(`${modifier}+b`);
    await expect(primary(page)).toBeHidden();
    await expect(control(page, "primary sidebar")).toBeFocused();
    await page.keyboard.press(`${modifier}+b`);
    await expect(primary(page)).toBeVisible();
    await page.keyboard.press(`${otherModifier}+b`);
    await expect(primary(page)).toBeVisible();

    await bottom(page)
      .getByRole("tab", { name: "Diagnostics", exact: true })
      .focus();
    await page.keyboard.press(`${modifier}+j`);
    await expect(bottom(page)).toBeHidden();
    await expect(control(page, "bottom panel")).toBeFocused();
    await page.keyboard.press(`${modifier}+j`);
    await expect(bottom(page)).toBeVisible();

    await secondary(page)
      .getByRole("region", { name: "Run information", exact: true })
      .focus();
    await page.keyboard.press(`${modifier}+Alt+b`);
    await expect(secondary(page)).toBeHidden();
    await expect(control(page, "secondary sidebar")).toBeFocused();
    await page.keyboard.press(`${modifier}+Alt+b`);
    await expect(secondary(page)).toBeVisible();
    await page.setViewportSize({ width: 640, height: 820 });
    await expect(secondary(page)).toBeHidden();
    await page.keyboard.press(`${modifier}+Alt+b`);
    await expect(secondary(page)).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(secondary(page)).toBeHidden();
    await expect(control(page, "secondary sidebar")).toBeFocused();
  });
}

test("composed speech remains controllable after leaving its composer", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?view=workspace&playback");
  await navigate(page, "Text to speech");
  const composer = page.getByRole("textbox", {
    name: "Text to speak",
    exact: true,
  });
  await composer.fill("An explicit composed speech request.");
  await composer.press("Control+Enter");
  await page
    .getByRole("button", { name: "Finish generation", exact: true })
    .click();
  const position = page.getByRole("slider", {
    name: "Playback position",
    exact: true,
  });
  await expect(position).toHaveCount(1);
  await navigate(page, "History");
  const player = page.locator('[aria-label="Speech playback"]');
  await expect(player).toHaveCount(1);
  await expect(player).toBeInViewport();
  await player
    .getByRole("button", { name: "Pause speech playback", exact: true })
    .click();
  await expect(
    player.getByRole("button", { name: "Resume speech playback", exact: true }),
  ).toBeVisible();
  await player
    .getByRole("slider", { name: "Playback position", exact: true })
    .press("ArrowRight");
  await expect(position).toHaveAttribute("aria-valuenow", "10100");
  await navigate(page, "Voice transcription");
  await expect(player).toBeInViewport();
  await player
    .getByRole("button", { name: "Resume speech playback", exact: true })
    .click();
  await navigate(page, "Text to speech");
  await expect(position).toHaveCount(1);
  await expect(position).toHaveAttribute("aria-valuenow", "10100");
  await expect(
    page.getByRole("button", { name: "Pause speech playback", exact: true }),
  ).toHaveCount(1);
  await expect(
    page.getByText("Speech requests: 1; seek requests: 1", { exact: true }),
  ).toBeVisible();
});

test("first-run layout controls preserve unavailable sidebar preferences and recover focus", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&blank");
  await expect(
    page.getByRole("region", { name: "First-run setup", exact: true }),
  ).toBeVisible();
  await expect(control(page, "primary sidebar")).toBeDisabled();
  await expect(primary(page)).toBeHidden();
  const savedPrimary = () =>
    page.evaluate(() => {
      const saved = localStorage.getItem("freehand-workbench-layout-v1");
      return saved ? JSON.parse(saved).primaryOpen : null;
    });
  await expect.poll(savedPrimary).toBe(true);
  await page.keyboard.press("Control+b");
  await page.keyboard.press("Control+k");
  const command = page.getByRole("combobox", { name: "Command", exact: true });
  await command.fill("Toggle primary sidebar");
  await expect(
    page.getByRole("option", { name: "Toggle primary sidebar", exact: true }),
  ).toHaveCount(0);
  expect(await savedPrimary()).toBe(true);
  await page.keyboard.press("Escape");

  const divider = page.getByRole("separator", {
    name: "Resize editor and bottom panel",
    exact: true,
  });
  await divider.focus();
  await page.setViewportSize({ width: 1000, height: 520 });
  await expect(divider).toHaveCount(0);
  await expect(control(page, "bottom panel")).toBeDisabled();
  const rail = page.getByRole("navigation", { name: "Workspace", exact: true });
  await expect(
    rail.getByRole("button", { name: "Voice transcription", exact: true }),
  ).toBeFocused();

  await navigate(page, "Settings");
  await expect(control(page, "primary sidebar")).toBeEnabled();
  await expect(primary(page)).toBeVisible();
  await page.keyboard.press("Control+k");
  await command.fill("Toggle primary sidebar");
  await expect(
    page.getByRole("option", { name: "Toggle primary sidebar", exact: true }),
  ).toBeVisible();
});
