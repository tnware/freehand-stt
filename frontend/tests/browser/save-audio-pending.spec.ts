import { test, expect } from "./fixtures";

for (const width of [560, 1000]) {
  for (const compact of [false, true]) {
    test(`audio save feedback recovers without blocking ${compact ? "transcript" : "composer"} playback at ${width}px`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 740 });
      await page.goto("/tests/browser/app/?view=workspace&playback&save-pending&theme=dark");
      await page.getByRole("tab", { name: "Text to speech", exact: true }).click();
      const draft = page.getByRole("textbox", { name: "Text to speak", exact: true });
      await draft.fill("Synthetic audio for save controls.");
      await draft.press("Control+Enter");
      await page.getByRole("button", { name: "Finish generation", exact: true }).click();
      if (compact) {
        await page.getByRole("button", { name: "Show transcript playback", exact: true }).click();
        await page.getByRole("tab", { name: "Voice", exact: true }).click();
      }
      const bar = page.locator('[aria-label="Speech playback"]');
      const height = (await bar.boundingBox())!.height;
      let count = 0;
      for (const outcome of ["Cancel save", "Fail save", "Complete save"]) {
        if (compact)
          await bar.getByRole("button", { name: "Playback actions", exact: true }).click();
        const save = page.getByRole(compact ? "menuitem" : "button", {
          name: "Save generated speech",
          exact: true,
        });
        await expect(save).toBeEnabled();
        await save.click();
        count++;
        const pending = bar.getByRole("button", {
          name: compact ? "Playback actions" : "Saving generated speech",
          exact: true,
        });
        await expect(pending).toHaveAttribute("aria-busy", "true");
        expect((await bar.boundingBox())!.height).toBe(height);
        if (compact) await pending.click();
        const disabledSave = page.getByRole(compact ? "menuitem" : "button", {
          name: "Saving generated speech",
          exact: true,
        });
        await expect(disabledSave).toBeDisabled();
        await disabledSave.evaluate((element: HTMLElement) => element.click());
        await expect(page.getByText(`Save requests: ${count}`, { exact: true })).toBeVisible();
        if (compact) await page.keyboard.press("Escape");
        await bar.getByRole("button", { name: "Pause speech playback", exact: true }).click();
        const slider = bar.getByRole("slider", { name: "Playback position", exact: true });
        await expect(slider).toBeEnabled();
        await slider.focus();
        await slider.press("ArrowRight");
        await bar.getByRole("button", { name: "Resume speech playback", exact: true }).click();
        if (count === 1)
          await page.screenshot({
            path: info.outputPath(`saving-${compact ? "compact" : "embedded"}-${width}.png`),
          });
        await page.getByRole("button", { name: outcome, exact: true }).click();
        if (compact) await expect(pending).toHaveAttribute("aria-busy", "false");
        else
          await expect(
            bar.getByRole("button", { name: "Save generated speech", exact: true }),
          ).toBeEnabled();
        if (outcome === "Cancel save") {
          await expect(
            page.getByText("Generated speech saved as a WAV file.", { exact: true }),
          ).toHaveCount(0);
          await expect(page.getByText("Audio could not be saved", { exact: true })).toHaveCount(0);
        } else {
          await expect(
            page
              .getByText(
                outcome === "Fail save"
                  ? "Audio could not be saved"
                  : "Generated speech saved as a WAV file.",
                { exact: true },
              )
              .first(),
          ).toBeVisible();
        }
      }
    });
  }
}
