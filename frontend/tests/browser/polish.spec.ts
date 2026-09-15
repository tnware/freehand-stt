import { test, expect } from "./fixtures";

test("speech settings retain a failed voice draft and return to the composer after saving", async ({
  page,
  saves,
}) => {
  await page.setViewportSize({ width: 650, height: 550 });
  await page.goto("/tests/browser/app/?main&workflows&theme=dark");
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  await page
    .getByRole("textbox", { name: "Text to speak", exact: true })
    .fill("Keep this draft.");
  const sidebar = page.getByRole("complementary", {
    name: "Text to speech settings",
    exact: true,
  });
  await page
    .getByRole("button", { name: "Text to speech settings", exact: true })
    .click();
  const settings = page.locator('[data-pane="configuration"]');
  const voice = settings.getByRole("combobox", {
    name: "Choose voice",
    exact: true,
  });
  const save = settings.getByRole("button", {
    name: "Save",
    exact: true,
  });
  await voice.fill("custom-voice");
  await voice.press("Enter");
  await save.click();
  await saves.complete(await saves.waitForStart(), "failure");
  await expect(
    page
      .getByRole("complementary", { name: "Notifications", exact: true })
      .getByText("Fixture save failed. Try again.", { exact: true }),
  ).toBeVisible();
  await expect(voice).toHaveValue("custom-voice");
  await expect(settings).toBeVisible();
  await save.click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(settings).toBeVisible();
  await settings.getByRole("button", { name: "Done", exact: true }).click();
  await expect(settings).toHaveCount(0);
  await expect(sidebar).toBeHidden();
  await page
    .getByRole("button", { name: "Toggle primary sidebar", exact: true })
    .click();
  await expect(
    sidebar.getByRole("combobox", { name: "Choose voice", exact: true }),
  ).toHaveValue("custom-voice");
  await page
    .getByRole("button", { name: "Dismiss primary sidebar", exact: true })
    .click();
  await expect(
    page.getByRole("textbox", { name: "Text to speak", exact: true }),
  ).toHaveValue("Keep this draft.");
});

test("live transcript follows new text, pauses for reading, and resumes explicitly", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1156, height: 600 });
  await page.goto("/tests/browser/app/?view=transcript");
  const scroll = page
    .getByRole("region", { name: "Current result" })
    .locator(".overflow-y-auto");
  const gap = () =>
    scroll.evaluate((el) => el.scrollHeight - el.clientHeight - el.scrollTop);
  await expect.poll(gap).toBeLessThan(2);
  await page.getByRole("button", { name: "Append text" }).click();
  await expect.poll(gap).toBeLessThan(2);
  await scroll.hover();
  await page.mouse.wheel(0, -500);
  await expect(
    page.getByRole("button", { name: "Jump to latest" }),
  ).toBeVisible();
  const top = await scroll.evaluate((el) => el.scrollTop);
  await page.getByRole("button", { name: "Append text" }).click();
  await expect.poll(() => scroll.evaluate((el) => el.scrollTop)).toBe(top);
  await page.getByRole("button", { name: "Finalize" }).click();
  await expect(
    page.getByRole("textbox", { name: "Current transcript" }),
  ).toBeVisible();
  await expect.poll(() => scroll.evaluate((el) => el.scrollTop)).toBe(top);
  await page.getByRole("button", { name: "Jump to latest" }).click();
  await expect.poll(gap).toBeLessThan(2);
  await expect(
    page.getByRole("button", { name: "Jump to latest" }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "New recording" }).click();
  await expect.poll(gap).toBeLessThan(2);
});

test("failed speech model and speed edits can be discarded or saved without losing composed text", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?main&workflows");
  const rail = page.getByRole("navigation", { name: "Workspace", exact: true });
  await rail
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  const composer = page.getByRole("textbox", {
    name: "Text to speak",
    exact: true,
  });
  await composer.fill("Keep the model-change draft.");
  const sidebar = page.getByRole("complementary", {
    name: "Text to speech settings",
    exact: true,
  });
  await page
    .getByRole("button", { name: "Text to speech settings", exact: true })
    .click();
  const settings = page.locator('[data-pane="configuration"]');
  const model = settings.getByRole("combobox", {
    name: "Choose model",
    exact: true,
  });
  const originalModel = await model.inputValue();
  const slider = settings.getByRole("slider", {
    name: "Speech playback speed",
    exact: true,
  });
  const before = Number(await slider.getAttribute("aria-valuenow"));
  const save = settings.getByRole("button", {
    name: "Save",
    exact: true,
  });
  await model.fill("speech/alternate");
  await model.press("Enter");
  await slider.focus();
  await slider.press("ArrowRight");
  await save.click();
  await saves.complete(await saves.waitForStart(), "failure");
  await expect(model).toHaveValue("speech/alternate");
  await expect(slider).toHaveAttribute("aria-valuenow", String(before + 0.05));
  await settings.getByRole("button", { name: "Done", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Save changes?", exact: true })
    .getByRole("button", { name: "Discard", exact: true })
    .click();
  await expect(settings).toHaveCount(0);
  await expect(
    sidebar.getByRole("combobox", { name: "Choose model", exact: true }),
  ).toHaveValue(originalModel);
  await expect(
    sidebar.getByRole("slider", { name: "Speech playback speed", exact: true }),
  ).toHaveAttribute("aria-valuenow", String(before));
  await expect(composer).toHaveValue("Keep the model-change draft.");
  await page
    .getByRole("button", { name: "Text to speech settings", exact: true })
    .click();
  await model.fill("speech/alternate");
  await model.press("Enter");
  await slider.focus();
  await slider.press("ArrowRight");
  await save.click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(settings).toBeVisible();
  await settings.getByRole("button", { name: "Done", exact: true }).click();
  await expect(settings).toHaveCount(0);
  await expect(
    sidebar.getByRole("combobox", { name: "Choose model", exact: true }),
  ).toHaveValue("speech/alternate");
  await expect(
    sidebar.getByRole("slider", { name: "Speech playback speed", exact: true }),
  ).toHaveAttribute("aria-valuenow", String(before + 0.05));
  await expect(composer).toHaveValue("Keep the model-change draft.");
});
