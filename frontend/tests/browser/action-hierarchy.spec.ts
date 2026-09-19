import { test, expect } from "./fixtures";

for (const theme of ["light", "dark"] as const) {
  for (const width of [560, 1280]) {
    test(`workflow actions keep their hierarchy at ${width}px in ${theme}`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 820 });
      await page.emulateMedia({ colorScheme: theme });
      await page.goto(
        `/tests/browser/app/?view=workspace&theme=${theme}&file-streaming&file-actions`,
      );
      const capture = page.getByRole("region", {
        name: "Voice capture",
        exact: true,
      });
      const record = capture.getByRole("button", {
        name: "Start recording",
        exact: true,
      });
      const primaryFill = await record.evaluate(
        (node) => getComputedStyle(node).backgroundColor,
      );
      await expect(record.locator('svg[aria-hidden="true"]')).toHaveCount(1);
      await expect(record).toHaveCSS("height", "28px");
      await record.click();
      await page.mouse.move(0, 0);
      const stop = capture.getByRole("button", {
        name: "Stop recording",
        exact: true,
      });
      await expect(stop).toBeInViewport({ ratio: 1 });
      await expect(stop).toHaveCSS("background-color", primaryFill);
      const cancel = capture.getByRole("button", {
        name: "Cancel",
        exact: true,
      });
      await expect(cancel).not.toHaveCSS("background-color", primaryFill);
      await page.screenshot({
        path: info.outputPath(`voice-actions-${width}-${theme}.png`),
      });
      await cancel.click();

      await page
        .getByRole("navigation", { name: "Workspace", exact: true })
        .getByRole("button", { name: "Audio file", exact: true })
        .click();
      const file = page.getByRole("region", {
        name: "Audio file",
        exact: true,
      });
      const choose = file.getByRole("button", {
        name: "Choose audio file",
        exact: true,
      });
      await expect(choose).toHaveCSS("background-color", primaryFill);
      await page
        .getByRole("button", { name: "Streaming supported", exact: true })
        .click();
      const change = file.getByRole("button", {
        name: "Change audio file",
        exact: true,
      });
      await expect(change).not.toHaveCSS("background-color", primaryFill);
      const transcribe = file.getByRole("button", {
        name: "Transcribe",
        exact: true,
      });
      await expect(transcribe).toHaveCSS("background-color", primaryFill);
      await expect(transcribe).toBeInViewport({ ratio: 1 });
      await transcribe.click();
      const starting = file.getByRole("button", {
        name: "Starting…",
        exact: true,
      });
      await expect(starting).toBeDisabled();
      await expect(starting.locator("svg")).toHaveCSS("animation-name", "none");
      await expect(starting).toBeInViewport({ ratio: 1 });
      await page.screenshot({
        path: info.outputPath(`file-actions-${width}-${theme}.png`),
      });
      await page
        .getByRole("button", { name: "Reject file start", exact: true })
        .click();

      await page
        .getByRole("navigation", { name: "Workspace", exact: true })
        .getByRole("button", { name: "Text to speech", exact: true })
        .click();
      const composer = page.getByRole("region", {
        name: "Speech composer",
        exact: true,
      });
      await composer
        .getByRole("textbox", { name: "Text to speak" })
        .fill("A synthetic action layout example.");
      const speak = composer.getByRole("button", {
        name: "Speak",
        exact: true,
      });
      await expect(speak).toBeEnabled();
      await expect(speak).toHaveCSS("background-color", primaryFill);
      await expect(speak).toBeInViewport({ ratio: 1 });
      await expect(
        composer.getByRole("button", { name: "Clear", exact: true }),
      ).not.toHaveCSS("background-color", primaryFill);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    });
  }
}
