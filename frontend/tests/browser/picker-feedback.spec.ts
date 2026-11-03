import { test, expect } from "./fixtures";

for (const sidebar of [true, false]) {
  for (const theme of ["light", "dark"] as const) {
    test(`picker refresh feedback stays stable in ${sidebar ? "sidebar" : "settings"} / ${theme}`, async ({
      page,
    }, info) => {
      await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
      await page.goto(
        `/tests/browser/app/?view=picker-feedback&theme=${theme}${sidebar ? "&sidebar" : ""}`,
      );
      const controls = page.getByRole("region", { name: "Picker controls" });
      const model = controls.getByRole("button", {
        name: "Refresh models",
        exact: true,
      });
      const voices = controls.getByRole("button", {
        name: "Refresh voices",
        exact: true,
      });
      const modelInput = controls.getByRole("combobox", {
        name: "Choose model",
      });
      const modelInputBefore = (await modelInput.boundingBox())!;
      const modelBefore = (await model.boundingBox())!;
      const voicesBefore = (await voices.boundingBox())!;
      expect(modelBefore.height).toBe(voicesBefore.height);
      if (sidebar) expect(modelBefore.width).toBe(voicesBefore.width);
      await model.click();
      await voices.click();
      expect((await modelInput.boundingBox())!.y).toBe(modelInputBefore.y);
      for (const [action, before] of [
        [model, modelBefore],
        [voices, voicesBefore],
      ] as const) {
        await expect(action).toBeDisabled();
        await expect(action).toHaveAttribute("aria-busy", "true");
        expect((await action.boundingBox())!.width).toBe(before.width);
        await expect(action.locator("svg")).toHaveCSS("animation-name", "none");
      }
      await expect(
        controls.getByText("Loading voices…", { exact: true }),
      ).toBeVisible();
      await expect(
        controls.getByRole("combobox", { name: "Choose model" }),
      ).toHaveAccessibleDescription(/Loading model list/);
      await expect(
        controls.getByRole("combobox", { name: "Choose voice" }),
      ).toHaveAccessibleDescription("Loading voices…");
      await page.getByRole("button", { name: "Fail refreshes" }).click();
      await expect(model).toBeEnabled();
      await expect(voices).toBeEnabled();
      await expect(
        controls.getByText(/Could not load the model list/),
      ).toBeVisible();
      await expect(
        controls.getByText(/preset list remains available/),
      ).toBeVisible();
      await controls.getByRole("button", { name: "Show voices" }).click();
      await page.getByRole("option", { name: "Aria", exact: true }).click();
      await expect(
        controls.getByRole("combobox", { name: "Choose voice" }),
      ).toHaveValue("Aria");
      expect(
        await controls.evaluate((el) => el.scrollWidth <= el.clientWidth),
      ).toBe(true);
      await page.screenshot({ path: info.outputPath("picker-feedback.png") });
    });
  }
}

test("model guidance distinguishes discovery from selection and saved changes", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=picker-feedback&sidebar");
  const input = page.getByRole("combobox", { name: "Choose model" });
  await expect(
    page.getByText("Selection applies when you save.", { exact: true }),
  ).toBeVisible();
  await expect(input).toHaveAccessibleDescription(
    /does not download, load, or run models/,
  );
  await page.getByRole("button", { name: "About model settings" }).click();
  await expect(
    page.getByRole("dialog", { name: "About model settings" }),
  ).toContainText("a listed model may not support this task");
  await page.keyboard.press("Escape");
  await input.click();
  await expect(
    page.getByText(/Listed models are not tested for compatibility/),
  ).toBeVisible();
  await expect(
    page.getByRole("option", { name: /fixture\/model/ }),
  ).toContainText("Selected");
  await input.fill("my-server/exact-model-id");
  await expect(
    page.getByRole("option", { name: /Use “my-server\/exact-model-id”/ }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(input).toHaveValue("fixture/model");
});
