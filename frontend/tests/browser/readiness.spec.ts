import { test, expect } from "./fixtures";

for (const width of [520, 1000]) {
  test(`first run stays compact and completes only after a saved check at ${width}px`, async ({
    page,
    saves,
  }, info) => {
    await page.setViewportSize({ width, height: width === 520 ? 620 : 740 });
    await page.goto(
      "/tests/browser/app/?view=workspace&setup=first&pickers&theme=dark",
    );
    const setup = page.getByRole("region", {
      name: "First-run setup",
      exact: true,
    });
    await expect(setup).toBeVisible();
    await expect(
      setup.getByText(
        /Start with your voice|Say what’s on your mind|Welcome to Freehand/,
      ),
    ).toHaveCount(0);
    await expect(
      setup.getByRole("heading", { name: "Set up dictation" }),
    ).toBeVisible();
    await expect(setup.getByRole("list", { name: "Setup steps" })).toHaveCount(
      0,
    );
    await expect(
      setup.getByRole("combobox", { name: "Choose model", exact: true }),
    ).not.toBeVisible();
    await expect(
      page.getByRole("button", { name: "Finish setup", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Back to workspace" }),
    ).toHaveCount(0);
    await expect(setup.locator('[id$="voice-language"]')).not.toBeVisible();
    await expect(
      page.getByRole("button", { name: "Check connection", exact: true }),
    ).toBeInViewport();
    await page.screenshot({ path: info.outputPath(`setup-${width}.png`) });
    const completed = page
      .locator("summary")
      .filter({ hasText: "Setup details" });
    await completed.focus();
    await completed.press("Enter");
    await expect(
      page.getByRole("button", { name: "Review Microphone settings" }),
    ).toBeVisible();
    await completed.press("Enter");
    const engineChoice = setup
      .locator("summary")
      .filter({ hasText: "Connection and model" });
    await engineChoice.click();
    await expect(
      setup.getByRole("combobox", { name: "Choose model", exact: true }),
    ).toBeVisible();
    await page
      .locator("summary")
      .filter({ hasText: "Transcription options" })
      .click();
    await expect(setup.locator('[id$="voice-language"]')).toBeVisible();
    await page
      .locator("summary")
      .filter({ hasText: "Transcription options" })
      .click();
    await page
      .getByRole("button", { name: "Check connection", exact: true })
      .click();
    await expect(
      setup.getByText("You’re ready to begin.", {
        exact: true,
      }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Finish setup", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Finishing…", exact: true }),
    ).toBeDisabled();
    await saves.complete(await saves.waitForStart(), "failure");
    await expect(setup).toBeVisible();
    await page
      .getByRole("button", { name: "Finish setup", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect(setup).toHaveCount(0);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}

test("failed metadata check retries without unlocking first run", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=workspace&setup=retry&pickers");
  await page
    .getByRole("button", { name: "Check connection", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Finish setup", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Check again", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Finish setup", exact: true }),
  ).toBeEnabled();
});

test("missing model can still be discovered and saved during setup", async ({
  page,
  saves,
}) => {
  await page.goto(
    "/tests/browser/app/?view=workspace&setup=missing-model&pickers",
  );
  await page
    .locator("summary")
    .filter({ hasText: "Connection and model" })
    .click();
  await page
    .getByRole("button", { name: "Refresh models", exact: true })
    .click();
  const setup = page.getByRole("region", {
    name: "First-run setup",
    exact: true,
  });
  const model = setup.getByRole("combobox", {
    name: "Choose model",
    exact: true,
  });
  await model.fill("speech/stt");
  await page.getByRole("option", { name: /speech\/stt/ }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(model).toHaveValue("speech/stt");
  await expect(
    page.getByRole("button", {
      name: /Check connection|Finish setup/,
      exact: true,
    }),
  ).toBeEnabled();
});

test("microphone recovery is focused, dismissible, and does not block files", async ({
  page,
}, info) => {
  await page.setViewportSize({ width: 860, height: 660 });
  await page.goto(
    "/tests/browser/app/?view=workspace&setup=microphone&pickers&theme=dark",
  );
  await expect(
    page.getByRole("heading", { name: "Check your microphone" }),
  ).toBeVisible();
  await expect(
    page
      .getByRole("region", { name: "Task recovery", exact: true })
      .getByRole("combobox", { name: "Choose model", exact: true }),
  ).toHaveCount(0);
  await expect(page.getByText("Not checked during this session.")).toHaveCount(
    0,
  );
  await page.getByRole("button", { name: "Choose microphone" }).click();
  await expect(
    page.locator("footer").filter({ hasText: "Audio settings" }),
  ).toContainText("Audio settings");
  await page.screenshot({ path: info.outputPath("microphone-recovery.png") });
  await page.getByRole("button", { name: "Back to workspace" }).click();
  await expect(page.getByRole("region", { name: "Task recovery" })).toHaveCount(
    0,
  );
  await page.getByRole("button", { name: "Audio file", exact: true }).click();
  await expect(page.getByRole("region", { name: "Task recovery" })).toHaveCount(
    0,
  );
});

test("connection recovery keeps the transcript available and retries in the capture strip", async ({
  page,
}, info) => {
  await page.goto(
    "/tests/browser/app/?view=workspace&setup=connection&pickers&theme=dark",
  );
  const capture = page.getByRole("region", {
    name: "Voice capture",
    exact: true,
  });
  const status = capture.getByRole("status", {
    name: "Voice capture status",
    exact: true,
  });
  const result = page.getByRole("region", {
    name: "Current result",
    exact: true,
  });
  const record = capture.getByRole("button", {
    name: "Start recording",
    exact: true,
  });
  await expect(status).toHaveText("Connection check failed");
  await expect(result).toBeVisible();
  await expect(record).toBeEnabled();
  await expect(
    page.getByRole("region", { name: "Task recovery", exact: true }),
  ).toHaveCount(0);
  await page.screenshot({ path: info.outputPath("connection-recovery.png") });
  await capture
    .getByRole("button", { name: "Check connection", exact: true })
    .click();
  await expect(status).toHaveText("Ready to dictate");
  await expect(
    capture.getByText("Server reachable", { exact: true }),
  ).toBeVisible();
  await expect(result).toBeVisible();
  await expect(record).toBeEnabled();
});

test("device discovery cannot advance first run", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=workspace&setup=loading&pickers");
  await expect(
    page.getByRole("button", {
      name: "Looking for a microphone…",
      exact: true,
    }),
  ).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Finish setup", exact: true }),
  ).toHaveCount(0);
});

for (const width of [527, 1180]) {
  test(`fresh setup leads with a supported local model at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 757 });
    await page.goto(
      "/tests/browser/app/?view=workspace&setup=fresh&pickers&local-options",
    );
    const setup = page.getByRole("region", {
      name: "First-run setup",
      exact: true,
    });
    await expect(setup.getByText("Recommended", { exact: true })).toBeVisible();
    await expect(
      setup.getByRole("button", { name: "Set up local speech", exact: true }),
    ).toBeInViewport();
    await expect(
      setup.getByRole("button", { name: "Use a server", exact: true }),
    ).toBeInViewport();
    await expect(setup.getByText(/Nemotron 3.5 Streaming/)).toBeVisible();
    await expect(setup.locator(".provider-icon img")).toBeVisible();
    await expect(
      setup.getByText("NeMo-Speech.cpp", { exact: true }),
    ).toBeVisible();
    await expect(
      setup.getByText(/review the download before it starts/),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await expect(setup.getByRole("combobox")).toHaveCount(0);
    await expect(page.getByRole("list", { name: "Setup steps" })).toHaveCount(
      0,
    );
  });
}
