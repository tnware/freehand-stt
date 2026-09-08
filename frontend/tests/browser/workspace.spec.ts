import { test, expect } from "./fixtures";

test("file and speech workspaces retain usable controls and the speech draft", async ({
  page,
}, testInfo) => {
  await page.goto("/tests/browser/app/?view=workspace");
  await page.getByRole("tab", { name: "Audio file", exact: true }).click();
  await expect(
    page.getByRole("region", { name: "Current result", exact: true }),
  ).toBeInViewport();
  await expect(
    page.getByRole("button", { name: "Audio settings", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
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
  await page.getByRole("button", { name: "Speech settings", exact: true }).click();
  await expect(
    page.getByRole("dialog", { name: "Speech settings", exact: true }),
  ).toBeInViewport();
  await page.keyboard.press("Escape");
  await page.screenshot({
    path: testInfo.outputPath("speech-workspace-light.png"),
  });
  await page.getByRole("tab", { name: "Voice", exact: true }).click();
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
  await expect(composer).toHaveValue(
    "A draft to review, without generating audio.",
  );
});

for (const width of [560, 900, 1156]) {
  test(`workspace remains stable with popovers and nested menus at ${width}px`, async ({
    page,
    saves,
  }, testInfo) => {
    await page.setViewportSize({ width, height: width === 560 ? 560 : 850 });
    await page.goto("/tests/browser/app/?view=workspace&theme=dark");
    const result = page.getByRole("region", {
      name: "Current result",
      exact: true,
    });
    const transcript = page.getByRole("textbox", {
      name: "Current transcript",
    });
    await expect(transcript).toBeVisible();
    const original = await transcript.boundingBox();
    await page
      .getByRole("button", { name: "Audio settings", exact: true })
      .click();
    const audio = page.getByRole("dialog", {
      name: "Audio settings",
      exact: true,
    });
    await expect(audio).toBeVisible();
    expect(await transcript.boundingBox()).toEqual(original);
    await audio
      .getByRole("button", { name: "Microphone: System default microphone" })
      .click();
    await page
      .getByRole("menuitemradio", { name: "Desk microphone", exact: true })
      .click();
    const save = await saves.waitForStart();
    await saves.complete(save, "success");
    await expect(
      audio.getByRole("button", { name: "Microphone: Desk microphone" }),
    ).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(
      page.getByRole("button", { name: "Audio settings", exact: true }),
    ).toBeFocused();
    await page
      .getByRole("button", { name: "Transcription settings", exact: true })
      .click();
    const stt = page.getByRole("dialog", {
      name: "Transcription settings",
      exact: true,
    });
    await stt.getByRole("button", { name: "Model", exact: true }).click();
    await page
      .getByRole("option", { name: "speech/alternate", exact: true })
      .click();
    await saves.complete(await saves.waitForStart(), "failure");
    await expect(
      stt.getByRole("button", { name: "Model", exact: true }),
    ).toContainText("speech/stt");
    await page.keyboard.press("Escape");
    await page
      .getByRole("button", { name: "Cleanup settings", exact: true })
      .click();
    await expect(
      page.getByRole("dialog", { name: "Cleanup settings", exact: true }),
    ).toBeInViewport();
    await page.screenshot({
      path: testInfo.outputPath(`workspace-${width}-cleanup.png`),
    });
    await page.keyboard.press("Escape");
    await expect(result).toBeInViewport();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    if (width < 900) {
      await page
        .getByRole("button", { name: "History · 2", exact: true })
        .click();
      await expect(
        page.getByRole("complementary", { name: "Recent history" }),
      ).toBeVisible();
      await expect(result).toHaveCount(0);
    }
  });
}

test("desktop split resizes with pointer and keyboard and restores the chosen width", async ({
  page,
}, testInfo) => {
  await page.goto("/tests/browser/app/?view=workspace");
  const handle = page.getByRole("separator", {
    name: "Resize current result and history",
  });
  const result = page.getByRole("region", {
    name: "Current result",
    exact: true,
  });
  const initial = (await result.boundingBox())!;
  const grip = (await handle.boundingBox())!;
  await page.mouse.move(grip.x + grip.width / 2, grip.y + grip.height / 2);
  await page.mouse.down();
  await page.mouse.move(grip.x - 70, grip.y + grip.height / 2, { steps: 8 });
  await page.mouse.up();
  await expect
    .poll(async () => (await result.boundingBox())!.width)
    .toBeLessThan(initial.width - 40);
  await handle.focus();
  const beforeKeyboard = (await result.boundingBox())!.width;
  await page.keyboard.press("ArrowRight");
  await expect
    .poll(async () => (await result.boundingBox())!.width)
    .toBeGreaterThan(beforeKeyboard + 10);
  const adjusted = (await result.boundingBox())!.width;
  const adjustedPercent = Number(await handle.getAttribute("aria-valuenow"));
  // PaneForge debounces persistence. An earlier drag write can change storage
  // before the keyboard resize is saved, so wait for the actual final size.
  // The separator exposes rounded percentages; retain the pixel-level reload
  // assertion below to verify restoration of the complete layout.
  await expect
    .poll(() =>
      page.evaluate(() => {
        const stored = localStorage.getItem("paneforge:freehand-workspace-v1");
        if (!stored) return null;
        const layouts = Object.values(JSON.parse(stored)) as {
          layout: number[];
        }[];
        const resultPercent = layouts[0]?.layout[0];
        return resultPercent === undefined ? null : Math.round(resultPercent);
      }),
    )
    .toBe(adjustedPercent);
  await page.reload();
  await expect
    .poll(async () => Math.abs((await result.boundingBox())!.width - adjusted))
    .toBeLessThan(2);
  await page.screenshot({ path: testInfo.outputPath("workspace-light.png") });
});
