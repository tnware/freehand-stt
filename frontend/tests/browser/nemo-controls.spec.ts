import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

for (const width of [560, 1280]) {
  test(`NeMo controls preserve independent Voice and file preferences at ${width}px`, async ({
    page,
    saves,
  }, info) => {
    await page.setViewportSize({ width, height: 820 });
    await page.goto(
      "/tests/browser/app/?main&runtime&runtime-ready&theme=dark",
    );
    await openSection(page, "voice-transcription");
    const pane = page.locator('[data-pane="configuration"]');
    const controls = pane.locator("details").filter({
      has: page.locator("summary", {
        hasText: "NeMo transcription controls",
      }),
    });
    await controls.locator("summary").click();
    const punctuation = controls.getByRole("switch", {
      name: "Automatic punctuation",
      exact: true,
    });
    await expect(punctuation).toBeChecked();
    await punctuation.click();
    await controls
      .getByRole("switch", { name: "Normalize numbers and dates", exact: true })
      .click();
    await controls
      .getByRole("switch", { name: "Profanity filter", exact: true })
      .click();
    await controls
      .getByRole("spinbutton", {
        name: "Server endpointing delay (ms)",
        exact: true,
      })
      .fill("1200");
    await controls.getByRole("spinbutton").press("Tab");
    await expect(controls).toContainText(
      "Managed runtimes do not install normalization grammars",
    );
    await pane.getByRole("button", { name: "Save", exact: true }).click();
    await saves.complete(await saves.waitForStart(), "success");
    await expect
      .poll(() =>
        page.evaluate(
          () =>
            window.testConnectionWindows.settings().voiceTranscription
              .transcriptionOptions.nemo,
        ),
      )
      .toEqual({
        disablePunctuation: true,
        normalize: true,
        profanityFilter: true,
        endpointingMilliseconds: 1200,
      });
    await openSection(page, "server");
    const fileControls = pane.locator("details").filter({
      has: page.locator("summary", {
        hasText: "NeMo transcription controls",
      }),
    });
    await fileControls.locator("summary").click();
    await expect(
      fileControls.getByRole("switch", {
        name: "Automatic punctuation",
        exact: true,
      }),
    ).toBeChecked();
    await expect(
      fileControls.getByRole("switch", {
        name: "Normalize numbers and dates",
        exact: true,
      }),
    ).not.toBeChecked();
    await expect(fileControls.getByRole("spinbutton")).toHaveCount(0);
    await fileControls
      .getByRole("switch", { name: "Profanity filter", exact: true })
      .scrollIntoViewIfNeeded();
    await page.screenshot({
      path: info.outputPath(`nemo-controls-${width}.png`),
      fullPage: true,
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}
