import { test, expect } from "./fixtures";

for (const width of [1280, 560]) {
  test(`Voice microphone selection saves and recovers at ${width}px`, async ({
    page,
    saves,
  }, info) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/tests/browser/app/?main&workflows&pickers&setup-ready");
    const sidebar = page.locator("#workbench-primary-sidebar");
    if (!(await sidebar.isVisible()))
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();
    const input = sidebar.getByRole("group", {
      name: "Microphone input",
      exact: true,
    });
    const microphone = input.getByRole("button", {
      name: "Microphone",
      exact: true,
    });
    await expect(microphone).toContainText("System default microphone");

    await microphone.click();
    await page
      .getByRole("option", { name: "Fixture microphone", exact: true })
      .click();
    const failed = await saves.waitForStart();
    await expect(microphone).toBeDisabled();
    await expect(input.getByRole("status")).toContainText("Saving…");
    await saves.complete(failed, "failure");
    await expect(microphone).toBeEnabled();
    await expect(microphone).toContainText("System default microphone");
    await expect(input.getByRole("status")).toContainText(
      "Your previous settings are still active",
    );

    await microphone.click();
    await page
      .getByRole("option", { name: "Fixture microphone", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect(microphone).toContainText("Fixture microphone");
    await expect(input.getByRole("status")).toHaveText("Saved");
    await input
      .getByRole("button", { name: "Refresh microphones", exact: true })
      .click();
    await expect(microphone).toContainText("Fixture microphone");
    await page.screenshot({ path: info.outputPath("microphone-sidebar.png") });
    expect(
      await sidebar.evaluate((el) => el.scrollWidth <= el.clientWidth),
    ).toBe(true);

    if (width < 700)
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();
    await page
      .getByRole("button", { name: "Voice settings", exact: true })
      .click();
    const inspector = page.locator('[data-pane="configuration"]');
    await inspector
      .getByRole("tab", { name: "Audio settings", exact: true })
      .click();
    await expect(inspector.locator("#microphone-select")).toContainText(
      "Fixture microphone",
    );
    await inspector.locator("#max-duration").fill("140");
    // At narrow widths the inspector covers the still-mounted primary sidebar.
    const mountedMicrophone = sidebar.locator('button[id$="-microphone"]');
    await expect(mountedMicrophone).toBeDisabled();
    await inspector
      .getByRole("button", { name: "Discard changes", exact: true })
      .click();
    await expect(mountedMicrophone).toBeEnabled();
    await inspector.getByRole("button", { name: "Done", exact: true }).click();
    if (!(await sidebar.isVisible()))
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();

    await microphone.click();
    await page
      .getByRole("option", { name: "System default microphone", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect(microphone).toContainText("System default microphone");
    await page.getByRole("button", { name: "Audio file", exact: true }).click();
    await expect(input).toHaveCount(0);
  });
}

for (const workflow of ["Voice transcription", "Audio file"]) {
  test(`${workflow} custom language commits the complete value`, async ({
    page,
    saves,
  }) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.goto("/tests/browser/app/?main&workflows&pickers&setup-ready");
    await page.getByRole("button", { name: workflow, exact: true }).click();
    const sidebar = page.locator("#workbench-primary-sidebar");
    const language = sidebar.locator(
      workflow === "Audio file"
        ? 'input[id$="-quick-file-language"]'
        : 'input[id$="-voice-language"]',
    );
    await language.fill("unlisted");
    await page
      .getByRole("option", { name: "Custom server value…", exact: true })
      .click();
    const custom = sidebar.getByRole("textbox", {
      name: "Custom language value",
      exact: true,
    });
    await custom.fill("");
    await custom.pressSequentially("en-US");
    await expect(custom).toHaveValue("en-US");
    await expect(custom).toBeEnabled();
    await custom.press("Enter");
    const save = await saves.waitForStart();
    await expect(custom).toBeDisabled();
    await saves.complete(save, "success");
    await expect(custom).toBeEnabled();
    await expect(custom).toHaveValue("en-US");
    await expect(page.locator("#workbench-secondary-sidebar")).toBeHidden();
  });
}

test("Voice quick model saves immediately, recovers from failure, and respects a dirty inspector", async ({
  page,
  saves,
}, info) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&workflows&pickers&setup-ready");
  const sidebar = page.locator("#workbench-primary-sidebar");
  const model = sidebar.locator('input[id$="-voice-model"]');
  const original = await model.inputValue();
  await expect(
    sidebar
      .getByRole("combobox", { name: "Choose connection", exact: true })
      .first(),
  ).toBeVisible();
  await model.fill("voice/next");
  await page.getByRole("option", { name: /voice\/next/ }).click();
  const firstSave = await saves.waitForStart();
  await expect(model).toBeDisabled();
  await saves.complete(firstSave, "failure");
  await expect(model).toBeEnabled();
  await page.keyboard.press("Escape");
  await expect(model).toHaveValue(original);
  await model.fill("voice/next");
  await page.getByRole("option", { name: /voice\/next/ }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(model).toHaveValue("voice/next");
  await expect(page.locator("#workbench-secondary-sidebar")).toBeHidden();
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  const inspector = page.locator('[data-pane="configuration"]');
  await inspector.locator("#voice-profile").click();
  await page.getByRole("option", { name: /Qwen3-ASR/ }).click();
  await expect(model).toBeDisabled();
  await expect(inspector.locator("#voice-profile")).toContainText("Qwen3-ASR");
  await page.screenshot({
    path: info.outputPath("voice-quick-and-detailed.png"),
  });
});

test("file language stays independent and cleanup quick changes are shared with Voice", async ({
  page,
  saves,
}, info) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&workflows&pickers&setup-ready");
  const sidebar = page.locator("#workbench-primary-sidebar");
  const voiceLanguage = await sidebar
    .locator('input[id$="-voice-language"]')
    .inputValue();
  await page.getByRole("button", { name: "Audio file", exact: true }).click();
  const language = sidebar.getByRole("combobox", {
    name: "Language",
    exact: true,
  });
  await language.fill("fr");
  await page.getByRole("option", { name: "French (fr)", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(language).toHaveValue("French (fr)");
  const cleanup = sidebar.getByRole("switch", {
    name: "Post-process transcripts",
    exact: true,
  });
  const previous = await cleanup.getAttribute("aria-checked");
  await cleanup.click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(cleanup).toHaveAttribute(
    "aria-checked",
    previous === "true" ? "false" : "true",
  );
  await page.screenshot({ path: info.outputPath("file-quick-settings.png") });
  await page
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await expect(sidebar.locator('input[id$="-voice-language"]')).toHaveValue(
    voiceLanguage,
  );
  await expect(cleanup).toHaveAttribute(
    "aria-checked",
    previous === "true" ? "false" : "true",
  );
  await page.getByRole("button", { name: "Audio file", exact: true }).click();
  await expect(language).toHaveValue("French (fr)");
  await expect(page.locator("#workbench-secondary-sidebar")).toBeHidden();
});

test("speech quick controls save speed directly and fit the sidebar", async ({
  page,
  saves,
}, info) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&workflows&pickers&setup-ready");
  await page
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  const sidebar = page.locator("#workbench-primary-sidebar");
  await expect(
    sidebar.getByRole("group", { name: "Speech quick settings", exact: true }),
  ).toBeVisible();
  await expect(
    sidebar.getByRole("combobox", { name: "Choose connection", exact: true }),
  ).toBeVisible();
  const speed = sidebar.getByRole("slider", { name: "Speech playback speed" });
  await speed.focus();
  await speed.press("ArrowRight");
  await saves.complete(await saves.waitForStart(), "success");
  await expect(speed).toHaveAttribute("aria-valuenow", "1.05");
  await expect(
    page.getByRole("dialog", { name: "Speech settings", exact: true }),
  ).toHaveCount(0);
  await expect(page.locator("#workbench-secondary-sidebar")).toBeHidden();
  await page.screenshot({ path: info.outputPath("speech-quick-settings.png") });
  expect(
    await sidebar.evaluate(
      (element) => element.scrollWidth <= element.clientWidth,
    ),
  ).toBe(true);
});
