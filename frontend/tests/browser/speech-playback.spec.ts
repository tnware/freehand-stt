import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";
async function composer(page: Page, theme = "dark") {
  await page.goto(`/tests/browser/app/?view=workspace&theme=${theme}&playback`);
  await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
  return page.getByRole("textbox", { name: "Text to speak", exact: true });
}
async function playing(page: Page, theme = "dark") {
  const text = await composer(page, theme);
  await text.fill("A short synthetic playback example.");
  await text.press("Control+Enter");
  await page.getByRole("button", { name: "Finish generation", exact: true }).click();
  return page.getByRole("slider", { name: "Playback position", exact: true });
}

for (const theme of ["dark", "light"]) {
  test(`transcript playback shares the surrounding panel styles in ${theme} mode`, async ({
    page,
  }, info) => {
    await playing(page, theme);
    await page.getByRole("button", { name: "Show transcript playback", exact: true }).click();
    await page.getByRole("tab", { name: "Voice", exact: true }).click();
    const bar = page.locator('[aria-label="Speech playback"]');
    // The result's outer frame owns its surface, stroke and corner geometry.
    const result = page.getByRole("region", { name: "Current result", exact: true }).locator("..");
    const surface = (element: Element) => {
      const css = getComputedStyle(element);
      return {
        background: css.backgroundColor,
        border: css.borderTopColor,
        width: css.borderTopWidth,
        radius: css.borderTopLeftRadius,
      };
    };
    await expect(bar).toBeVisible();
    expect(await bar.evaluate(surface)).toEqual(await result.evaluate(surface));
    expect(parseFloat((await bar.evaluate(surface)).radius)).toBeGreaterThan(0);
    await page.screenshot({ path: info.outputPath(`transcript-playback-${theme}.png`) });
  });
}
for (const width of [560, 1000]) {
  test(`transcript playback stays compact with accessible controls at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 560 });
    await playing(page);
    await page.getByRole("button", { name: "Show transcript playback", exact: true }).click();
    await page.getByRole("tab", { name: "Voice", exact: true }).click();
    const bar = page.locator('[aria-label="Speech playback"]');
    const slider = bar.getByRole("slider", { name: "Playback position", exact: true });
    expect((await bar.boundingBox())!.height).toBeLessThanOrEqual(48);
    expect((await bar.locator('[data-slot="slider-track"]').boundingBox())!.width).toBeGreaterThan(
      220,
    );
    await slider.focus();
    await slider.press("ArrowRight");
    await expect(slider).toHaveAttribute("aria-valuenow", "10100");
    await bar.getByRole("button", { name: "Pause speech playback", exact: true }).click();
    await expect(
      bar.getByRole("button", { name: "Resume speech playback", exact: true }),
    ).toBeVisible();
    await expect(
      bar.getByRole("button", { name: "Stop and release speech playback", exact: true }),
    ).toBeVisible();
    await page.screenshot({ path: info.outputPath(`compact-playback-${width}.png`) });
    const actions = bar.getByRole("button", { name: "Playback actions", exact: true });
    await actions.focus();
    await actions.press("Enter");
    await expect(
      page.getByRole("menuitem", { name: "Restart speech playback", exact: true }),
    ).toBeVisible();
    await page.getByRole("menuitem", { name: "Save generated speech", exact: true }).click();
    await expect(page.getByRole("menu")).toHaveCount(0);
    await slider.focus();
    await slider.press("End");
    await expect(
      bar.getByRole("button", { name: "Restart speech playback", exact: true }),
    ).toBeVisible();
    await actions.click();
    await page.getByRole("menuitem", { name: "Clear generated speech", exact: true }).click();
    await expect(bar).toHaveCount(0);
    await page.screenshot({ path: info.outputPath(`compact-playback-cleared-${width}.png`) });
  });
}

for (const width of [560, 1000]) {
  test(`speech draft stays editable without changing current audio at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 620 });
    const text = await composer(page);
    const submitted = page.locator('[aria-label="Submitted speech text"]');
    await text.fill("The original request.");
    await text.press("Control+Enter");
    await expect(text).toBeEditable();
    await text.fill("A draft for the next request.");
    await text.press("Control+Enter");
    await expect(submitted).toHaveText("The original request.");
    await expect(
      page.getByText("Speech requests: 1; seek requests: 0", { exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Finish generation", exact: true }).click();
    await expect(text).toHaveValue("A draft for the next request.");
    await text.fill("Edited during playback.");
    const slider = page.getByRole("slider", { name: "Playback position", exact: true });
    await expect(slider).toHaveAttribute("aria-valuenow", "10000");
    await page.getByRole("button", { name: "Pause speech playback", exact: true }).click();
    await expect(text).toBeEditable();
    await page
      .getByRole("region", { name: "Speech composer" })
      .getByRole("button", { name: "Clear", exact: true })
      .click();
    await expect(text).toHaveValue("");
    await expect(slider).toHaveAttribute("aria-valuenow", "10000");
    await expect(
      page.getByRole("button", { name: "Resume speech playback", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Save generated speech", exact: true }),
    ).toBeEnabled();
    await expect(submitted).toHaveText("The original request.");
    await text.fill("The replacement request.");
    await expect(page.getByRole("button", { name: "Speak", exact: false })).toBeEnabled();
    await text.press("Control+Enter");
    await expect(submitted).toHaveText("The replacement request.");
    await expect(
      page.getByText("Speech requests: 2; seek requests: 0", { exact: true }),
    ).toBeVisible();
  });

  test(`speech generation and seeking stay clear and keyboard accessible at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    const text = await composer(page);
    await text.fill("A short synthetic playback example.");
    await text.press("End");
    await text.press("Enter");
    await expect(text).toHaveValue("A short synthetic playback example.\n");
    await expect(
      page.getByText("Speech requests: 0; seek requests: 0", { exact: true }),
    ).toBeVisible();
    await text.press("Control+Enter");
    await expect(page.getByText("Text to speech · Creating audio", { exact: true })).toBeVisible();
    await expect(page.getByText("Audio will play when ready.", { exact: true })).toBeVisible();
    await expect(page.getByRole("slider", { name: "Playback position", exact: true })).toHaveCount(
      0,
    );
    await expect(page.getByText("0:00 / 0:00", { exact: true })).toHaveCount(0);
    const bar = page.locator('[aria-label="Speech playback"]');
    const generatingBox = await bar.boundingBox();
    await page.screenshot({ path: info.outputPath(`speech-generating-${width}.png`) });
    await page.getByRole("button", { name: "Finish generation", exact: true }).click();
    const slider = page.getByRole("slider", { name: "Playback position", exact: true });
    await expect(slider).toBeInViewport();
    expect(await bar.boundingBox()).toEqual(generatingBox);
    await slider.focus();
    await slider.press("ArrowRight");
    await expect(slider).toHaveAttribute("aria-valuenow", "10100");
    await expect(slider).toBeFocused();
    await expect(
      page.getByRole("button", { name: "Pause speech playback", exact: true }),
    ).toBeVisible();
    await slider.press("End");
    await expect(slider).toHaveAttribute("aria-valuenow", "60000");
    await expect(page.getByText("Text to speech · Complete", { exact: true })).toBeVisible();
    await slider.press("Home");
    await expect(slider).toHaveAttribute("aria-valuenow", "0");
    await expect(
      page.getByRole("button", { name: "Resume speech playback", exact: true }),
    ).toBeVisible();
    await page.screenshot({ path: info.outputPath(`speech-seeking-${width}.png`) });
    await page.getByRole("button", { name: "Resume speech playback", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Pause speech playback", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Speech requests: 1; seek requests: 3", { exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    ).toBe(true);
  });
}

test("dragging previews locally, commits on release, and ignores a replaced session", async ({
  page,
}) => {
  const slider = await playing(page);
  const track = page.locator('[data-slot="slider"]').filter({ has: slider });
  const box = (await track.boundingBox())!;
  await page.mouse.move(box.x + box.width * 0.4, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width * 0.75, box.y + box.height / 2);
  const preview = await slider.getAttribute("aria-valuenow");
  await page
    .getByRole("button", { name: "Advance playback", exact: true })
    .evaluate((button: HTMLButtonElement) => button.click());
  await expect(slider).toHaveAttribute("aria-valuenow", preview!);
  await expect(
    page.getByText("Speech requests: 1; seek requests: 0", { exact: true }),
  ).toBeVisible();
  await page.mouse.up();
  await expect(
    page.getByText("Speech requests: 1; seek requests: 1", { exact: true }),
  ).toBeVisible();

  await page.mouse.move(box.x + box.width * 0.25, box.y + box.height / 2);
  await page.mouse.down();
  await page
    .getByRole("button", { name: "Replace audio", exact: true })
    .evaluate((button: HTMLButtonElement) => button.click());
  await page.mouse.up();
  await expect(slider).toHaveAttribute("aria-valuenow", "5000");
  await expect(
    page.getByText("Speech requests: 1; seek requests: 1", { exact: true }),
  ).toBeVisible();
});

test("composer shortcut respects empty, oversized, composing and repeated input", async ({
  page,
}) => {
  const text = await composer(page);
  await text.press("Control+Enter");
  await text.fill("x".repeat(4097));
  await text.press("Control+Enter");
  await text.fill("A draft");
  await text.dispatchEvent("keydown", {
    key: "Enter",
    ctrlKey: true,
    isComposing: true,
    bubbles: true,
  });
  await text.dispatchEvent("keydown", { key: "Enter", ctrlKey: true, repeat: true, bubbles: true });
  await expect(
    page.getByText("Speech requests: 0; seek requests: 0", { exact: true }),
  ).toBeVisible();
  await text.press("Control+Enter");
  await expect(
    page.getByText("Speech requests: 1; seek requests: 0", { exact: true }),
  ).toBeVisible();
});

test("the full-width track follows irregular playback updates before and after seeking", async ({
  page,
}, info) => {
  const slider = await playing(page);
  const bar = page.locator('[aria-label="Speech playback"]');
  const track = bar.locator('[data-slot="slider-track"]');
  const fill = bar.locator('[data-slot="slider-range"]');
  const advance = page.getByRole("button", { name: "Advance playback", exact: true });
  for (const value of [10100, 10300, 10400]) {
    await advance.click();
    await expect(slider).toHaveAttribute("aria-valuenow", String(value));
  }
  const barBox = (await bar.boundingBox())!;
  const trackBox = (await track.boundingBox())!;
  expect(trackBox.width).toBeGreaterThan(barBox.width - 30);
  expect((await fill.boundingBox())!.height).toBeGreaterThan(0);
  // Click the actual visible track, rather than the larger invisible hit area.
  await track.click({ position: { x: trackBox.width * 0.75, y: trackBox.height / 2 } });
  const selected = Number(await slider.getAttribute("aria-valuenow"));
  expect(selected).toBeGreaterThan(44000);
  expect(selected).toBeLessThan(46000);
  await advance.click();
  await expect(slider).toHaveAttribute("aria-valuenow", String(selected + 100));
  await advance.click();
  await expect(slider).toHaveAttribute("aria-valuenow", String(selected + 300));
  const fillBox = (await fill.boundingBox())!;
  const thumbBox = (await slider.boundingBox())!;
  expect(fillBox.width / trackBox.width).toBeGreaterThan(0.74);
  expect(fillBox.width / trackBox.width).toBeLessThan(0.78);
  expect(Math.abs(fillBox.x + fillBox.width - thumbBox.x - thumbBox.width / 2)).toBeLessThan(2);
  await expect(
    page.getByText("Speech requests: 1; seek requests: 1", { exact: true }),
  ).toBeVisible();
  await page.screenshot({ path: info.outputPath("speech-following-playback.png") });
});
