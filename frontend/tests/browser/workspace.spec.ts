import { test, expect } from "./fixtures";

const scenarioURL = "/tests/browser/app/?view=workspace";

test("file and speech retain controls and the speech draft", async ({
  page,
}, info) => {
  await page.goto(scenarioURL);
  await page.getByRole("button", { name: "Audio file", exact: true }).click();
  await expect(
    page.getByRole("region", { name: "Current result", exact: true }),
  ).toBeInViewport();
  await page
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  const composer = page.getByRole("textbox", {
    name: "Text to speak",
    exact: true,
  });
  await composer.fill("A draft to review, without generating audio.");
  await expect(
    page.getByRole("button", { name: "Speak", exact: true }),
  ).toBeEnabled();
  await expect(
    page.getByRole("button", { name: "Speak", exact: true }),
  ).toBeInViewport();
  await page
    .getByRole("button", { name: "Speech settings", exact: true })
    .click();
  await expect(
    page.getByRole("dialog", { name: "Speech settings", exact: true }),
  ).toBeInViewport();
  await page.keyboard.press("Escape");
  await page.screenshot({
    path: info.outputPath("speech-workspace-light.png"),
  });
  await page
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  await expect(composer).toHaveValue(
    "A draft to review, without generating audio.",
  );
});

for (const width of [560, 900, 1156]) {
  test(`workflow settings and history stay reachable at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: width === 560 ? 560 : 850 });
    await page.goto(`${scenarioURL}&theme=dark`);
    const result = page.getByRole("region", {
      name: "Current result",
      exact: true,
    });
    const transcript = page.getByRole("textbox", {
      name: "Current transcript",
    });
    await expect(transcript).toBeVisible();
    const original = await transcript.boundingBox();
    if (width < 700)
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();
    const sidebar = page.getByRole("complementary", {
      name: "Voice transcription settings",
    });
    await sidebar.getByRole("button", { name: /^Connection/ }).click();
    await expect(
      page
        .getByRole("status")
        .filter({ hasText: /^voice-transcription settings$/ }),
    ).toBeVisible();
    expect(await transcript.boundingBox()).toEqual(original);
    if (width < 700)
      await page
        .getByRole("button", { name: "Dismiss primary sidebar", exact: true })
        .click();
    await expect(result).toBeInViewport();
    await page.getByRole("button", { name: "History", exact: true }).click();
    await expect(
      page.getByRole("region", { name: "Selected transcript", exact: true }),
    ).toBeVisible();
    await expect(result).toHaveCount(0);
    await page.screenshot({
      path: info.outputPath(`history-workspace-${width}.png`),
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}

test("transcript and bottom panel resize vertically and restore the chosen height", async ({
  page,
}, info) => {
  await page.goto(scenarioURL);
  const handle = page.getByRole("separator", {
    name: "Resize editor and bottom panel",
  });
  const result = page.getByRole("region", {
    name: "Current result",
    exact: true,
  });
  const initial = (await result.boundingBox())!;
  const grip = (await handle.boundingBox())!;
  await page.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2);
  await page.mouse.down();
  await page.mouse.move(grip.x + grip.width / 2, grip.y - 70, { steps: 8 });
  await page.mouse.up();
  await expect
    .poll(async () => (await result.boundingBox())!.height)
    .toBeLessThan(initial.height - 40);
  await handle.focus();
  const beforeKeyboard = (await result.boundingBox())!.height;
  await page.keyboard.press("ArrowDown");
  await expect
    .poll(async () => (await result.boundingBox())!.height)
    .toBeGreaterThan(beforeKeyboard + 8);
  const adjusted = (await result.boundingBox())!.height;
  const adjustedPercent = Number(await handle.getAttribute("aria-valuenow"));
  await expect
    .poll(() =>
      page.evaluate(() => {
        const stored = localStorage.getItem("freehand-workbench-layout-v1");
        if (!stored) return null;
        const percent = JSON.parse(stored).bottomSize;
        return percent === undefined ? null : Math.round(percent);
      }),
    )
    .toBe(adjustedPercent);
  await page.reload();
  await expect
    .poll(async () => Math.abs((await result.boundingBox())!.height - adjusted))
    .toBeLessThan(2);
  await page.screenshot({ path: info.outputPath("workspace-light.png") });
});

test("voice playback works with history off and keyboard tooltips", async ({
  page,
}) => {
  await page.goto(`${scenarioURL}&history=off`);
  const result = page.getByRole("region", {
    name: "Current result",
    exact: true,
  });
  await result.getByRole("button", { name: "Listen", exact: true }).click();
  await expect(
    page.getByRole("status").filter({ hasText: "Voice transcript - Complete" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Restart speech playback", exact: true })
    .focus();
  await expect(page.getByRole("tooltip")).toHaveText("Restart speech playback");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("tooltip")).toHaveCount(0);
});

test("copy confirmation expires and history icons have hover help", async ({
  page,
}) => {
  await page.goto(scenarioURL);
  const result = page.getByRole("region", {
    name: "Current result",
    exact: true,
  });
  await result.getByRole("button", { name: "Copy", exact: true }).click();
  await expect(
    result.getByRole("button", { name: "Copied", exact: true }),
  ).toBeVisible();
  await expect(
    result.getByRole("button", { name: "Copy", exact: true }),
  ).toBeVisible();
  const copyHistory = page
    .getByRole("button", { name: "Copy transcript", exact: true })
    .first();
  await copyHistory.hover();
  await expect(page.getByRole("tooltip")).toHaveText("Copy transcript");
  await copyHistory.click();
  await expect(
    page.getByRole("button", { name: "Transcript copied", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Transcript copied", exact: true }),
  ).toHaveCount(0);
});

test("speech input preserves supplementary Unicode and explains the limit", async ({
  page,
}) => {
  await page.goto(scenarioURL);
  await page
    .getByRole("button", { name: "Text to speech", exact: true })
    .click();
  const composer = page.getByRole("textbox", {
    name: "Text to speak",
    exact: true,
  });
  const text = "😀".repeat(4096);
  await composer.fill(text);
  await expect(composer).toHaveValue(text);
  await expect(
    page.getByRole("button", { name: "Speak", exact: true }),
  ).toBeEnabled();
  await composer.fill(text + "😀");
  await expect(composer).toHaveValue(text + "😀");
  await expect(composer).toHaveAttribute("aria-invalid", "true");
  await expect(
    page.getByText("Shorten text to speak", { exact: false }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Speak", exact: true }),
  ).toBeDisabled();
  await composer.press("Backspace");
  await expect(
    page.getByRole("button", { name: "Speak", exact: true }),
  ).toBeEnabled();
});
