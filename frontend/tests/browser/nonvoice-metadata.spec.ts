import { openSection } from "./context-navigation";
import { test, expect } from "./fixtures";
import { Purpose } from "../../bindings/github.com/tnware/freehand-stt/internal/savedconnection/models";

const roles = [
  {
    purpose: Purpose.Transcription,
    section: "server",
    tab: "Audio file",
    quickID: "quick-stt-model",
    settingsID: "model",
  },
  {
    purpose: Purpose.Cleanup,
    section: "processing",
    tab: "Audio file",
    quickID: "quick-processing-model",
    settingsID: "cleanup-model",
  },
  {
    purpose: Purpose.Speech,
    section: "speech",
    tab: "Text to speech",
    quickID: "speech-model",
    settingsID: "tts-model",
  },
] as const;
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
        await page.getByRole("button", { name: role.tab, exact: true }).click();
        if (surface === "readiness")
          await page.getByText("Connection and model", { exact: true }).click();
      } else await openSection(page, role.section);
      const picker =
        surface === "settings"
          ? page
              .locator('[data-pane="configuration"]')
              .locator(`#${role.settingsID}`)
          : surface === "readiness"
            ? page.locator(`#${role.quickID}`)
            : page
                .getByRole("complementary", {
                  name: `${role.tab} settings`,
                  exact: true,
                })
                .locator(`input[id$="${role.quickID}"]`);
      const metadata = picker.locator("..").locator("..");
      await picker.click();
      await expect(
        metadata.getByText("Loading model list…", { exact: true }),
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
        metadata.getByText(/Could not load the model list/),
      ).toBeVisible();
      await page.keyboard.press("Escape");
      await picker.click();
      await expect.poll(count).toBe(2);
      await expect(
        metadata.getByText("Loading model list…", { exact: true }),
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
