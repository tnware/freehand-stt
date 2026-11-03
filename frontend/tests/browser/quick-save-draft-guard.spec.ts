import { test, expect } from "./fixtures";

test("an already-open inspector is read-only until a sidebar save finishes", async ({
  page,
  saves,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&workflows&pickers&setup-ready");
  const sidebar = page.locator("#workbench-primary-sidebar");
  const chooser = sidebar.getByRole("button", {
    name: "Change engine or model",
    exact: true,
  });
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  const inspector = page.locator('[data-pane="configuration"]');
  const language = inspector.locator("#voice-language");
  const previousLanguage = await language.inputValue();
  await expect(language).toBeEnabled();
  await expect(chooser).toBeEnabled();
  await chooser.click();
  const modelDialog = page.getByRole("dialog", {
    name: "Engine and model",
    exact: true,
  });
  const model = modelDialog.getByRole("combobox", {
    name: "Choose model",
    exact: true,
  });
  await model.fill("guard/next-model");
  await page.getByRole("option", { name: /Use “guard\/next-model”/ }).click();
  const quickSave = await saves.waitForStart();
  await page.keyboard.press("Escape");
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("dialog", { name: "Engine and model", exact: true }),
  ).toBeHidden();
  await expect(
    inspector.getByText("Applying quick settings…", { exact: true }),
  ).toBeVisible();
  await expect(language).toBeDisabled();
  await expect(language).not.toBeEditable();
  await expect(language).toHaveValue(previousLanguage);
  await expect(inspector.locator("#voice-profile")).toBeDisabled();
  await expect(inspector.locator("#saved-connection-voice")).toBeDisabled();
  await expect(
    inspector.getByRole("button", { name: "Save", exact: true }),
  ).toBeDisabled();
  await expect(
    inspector.getByRole("button", { name: "Done", exact: true }),
  ).toBeDisabled();

  // Every section shares the lock, including controls without their own busy prop.
  await inspector
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  await expect(inspector.locator("#max-duration")).toBeDisabled();
  await inspector
    .getByRole("tab", { name: "Vocabulary settings", exact: true })
    .click();
  const terms = inspector.locator("#vocabulary-terms");
  const previousTerms = await terms.inputValue();
  await expect(terms).toBeDisabled();
  await expect(terms).not.toBeEditable();
  await expect(
    inspector.locator('fieldset[aria-label="Settings fields"]'),
  ).toHaveAttribute("inert", "");

  await saves.complete(quickSave, "success");
  await expect(terms).toBeEnabled();
  await expect(terms).toHaveValue(previousTerms);
  await expect(chooser).toContainText("guard/next-model");
  const nextTerms = [previousTerms, "Save guard regression"]
    .filter(Boolean)
    .join("\n");
  await terms.fill(nextTerms);
  await expect(chooser).toBeDisabled();
  await expect(
    inspector.getByText("Unsaved changes", { exact: true }),
  ).toBeVisible();
  await inspector.getByRole("button", { name: "Save", exact: true }).click();
  const draftSave = await saves.waitForStart();
  await expect(
    inspector.getByRole("button", { name: "Discard changes", exact: true }),
  ).toBeDisabled();
  await saves.complete(draftSave, "success");
  await expect(terms).toHaveValue(nextTerms);
  await expect(
    inspector.getByText("All changes saved", { exact: true }),
  ).toBeVisible();
  await inspector.getByRole("button", { name: "Done", exact: true }).click();
  await expect(chooser).toContainText("guard/next-model");
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  await inspector
    .getByRole("tab", { name: "Vocabulary settings", exact: true })
    .click();
  await expect(terms).toHaveValue(nextTerms);
});

test("opening the inspector during a failed quick save cannot create a draft that the response replaces", async ({
  page,
  saves,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&workflows&pickers&setup-ready");
  const chooser = page
    .locator("#workbench-primary-sidebar")
    .getByRole("button", { name: "Change engine or model", exact: true });
  const original = await chooser.innerText();
  await chooser.click();
  const model = page
    .getByRole("dialog", { name: "Engine and model", exact: true })
    .getByRole("combobox", { name: "Choose model", exact: true });
  await model.fill("guard/failed-model");
  await page.getByRole("option", { name: /Use “guard\/failed-model”/ }).click();
  const quickSave = await saves.waitForStart();
  await page.keyboard.press("Escape");
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("dialog", { name: "Engine and model", exact: true }),
  ).toBeHidden();
  await page
    .getByRole("button", { name: "Voice settings", exact: true })
    .click();
  const inspector = page.locator('[data-pane="configuration"]');
  await expect(inspector.locator("#voice-profile")).toBeDisabled();
  await inspector
    .getByRole("tab", { name: "Audio settings", exact: true })
    .click();
  const duration = inspector.locator("#max-duration");
  const previousDuration = await duration.inputValue();
  await expect(duration).toBeDisabled();
  await saves.complete(quickSave, "failure");
  await expect(duration).toBeEnabled();
  await expect(duration).toHaveValue(previousDuration);
  await expect(chooser).toHaveText(original);
  const changed = previousDuration === "140" ? "141" : "140";
  await duration.fill(changed);
  await expect(
    inspector.getByText("Unsaved changes", { exact: true }),
  ).toBeVisible();
  await inspector
    .getByRole("button", { name: "Discard changes", exact: true })
    .click();
  await expect(duration).toHaveValue(previousDuration);
  await expect(
    inspector.getByText("All changes saved", { exact: true }),
  ).toBeVisible();
});
