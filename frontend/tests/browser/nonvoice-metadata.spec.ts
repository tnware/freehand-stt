import { test, expect } from "./fixtures";
import { Purpose } from "../../bindings/github.com/tnware/freehand-stt/internal/savedconnection/models";

const roles = [
  {
    purpose: Purpose.Transcription,
    section: "server",
    tab: "Audio file",
    panel: "Transcription settings",
    quickID: "quick-stt-model",
    settingsID: "model",
  },
  {
    purpose: Purpose.Cleanup,
    section: "processing",
    tab: "Audio file",
    panel: "Cleanup settings",
    quickID: "quick-processing-model",
    settingsID: "cleanup-model",
  },
  {
    purpose: Purpose.Speech,
    section: "speech",
    tab: "Text to speech",
    panel: "Speech settings",
    quickID: "quick-speech-model",
    settingsID: "tts-model",
  },
];
for (const surface of ["quick", "settings", "readiness"] as const) {
  for (const role of roles.filter(
    (role) => surface !== "readiness" || role.purpose === Purpose.Transcription,
  )) {
    test(`${surface} ${role.purpose} picker retries failed metadata on deliberate entry`, async ({
      page,
    }) => {
      const main = surface !== "settings";
      await page.goto(
        `/tests/browser/app/?workflows&pickers&metadata-control${main ? "&main" : ""}${surface === "readiness" ? "&metadata-readiness" : ""}`,
      );
      if (main) {
        await page.getByRole("tab", { name: role.tab, exact: true }).click();
        if (surface === "readiness")
          await page.getByText("Connection and model", { exact: true }).click();
        if (surface === "quick")
          await page
            .getByRole("button", { name: role.panel, exact: true })
            .click();
      } else
        await page.locator(`[data-settings-section="${role.section}"]`).click();
      const picker = page.locator(`#${main ? role.quickID : role.settingsID}`);
      await picker.click();
      await expect(
        page.getByText("Loading model list…", { exact: true }),
      ).toBeVisible();
      await expect(picker).toBeEnabled();
      await expect(picker).toHaveAttribute("aria-expanded", "true");
      const count = () =>
        page.evaluate(
          (purpose) =>
            window.testMetadata.calls.filter((call) => call === purpose).length,
          role.purpose,
        );
      await expect.poll(count).toBe(1);
      await page.evaluate(
        (purpose) => window.testMetadata.complete(purpose, false),
        role.purpose,
      );
      await expect(
        page.getByText(/Could not load the model list/),
      ).toBeVisible();
      await page.keyboard.press("Escape");
      await picker.click();
      await expect.poll(count).toBe(2);
      await expect(
        page.getByText("Loading model list…", { exact: true }),
      ).toBeVisible();
      await page.evaluate(
        (purpose) => window.testMetadata.complete(purpose, true),
        role.purpose,
      );
      await expect(
        page.getByRole("option", { name: /fixture\/discovered/ }),
      ).toBeVisible();
      await expect(picker).toHaveAttribute("aria-expanded", "true");
      await page.keyboard.press("Escape");
      await picker.click();
      expect(await count()).toBe(2);
    });
  }
}
