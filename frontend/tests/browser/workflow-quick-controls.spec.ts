import { test, expect } from "./fixtures";

const workflows = [
  {
    name: "Voice transcription",
    model: "speech/stt",
    preferences: "Voice and microphone settings",
  },
  {
    name: "Audio file",
    model: "speech/stt",
    preferences: "Transcription settings",
  },
  {
    name: "Text to speech",
    model: "speech/tts",
    preferences: "Voice and speech settings",
  },
] as const;

for (const width of [1280, 560]) {
  for (const workflow of workflows) {
    test(`${workflow.name} keeps everyday controls simple at ${width}px`, async ({
      page,
      saves,
    }, info) => {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(
        "/tests/browser/app/?main&workflows&pickers&setup-ready&workflow-identity",
      );
      await page
        .getByRole("button", { name: workflow.name, exact: true })
        .click();
      const sidebar = page.locator("#workbench-primary-sidebar");
      if (!(await sidebar.isVisible()))
        await page
          .getByRole("button", { name: "Toggle primary sidebar", exact: true })
          .click();
      const chooser = sidebar.getByRole("button", {
        name: "Change engine or model",
        exact: true,
      });
      await expect(chooser).toBeVisible();
      await expect(chooser.locator(".model-name")).toHaveText(workflow.model);
      await expect(chooser.locator(".model-connection")).toHaveText(
        "Original server",
      );
      await expect(chooser.locator(".model-identity")).toContainText(
        "Generic OpenAI-compatible",
      );
      expect(
        await chooser.evaluate((el) => {
          const model = el.querySelector(".model-selection")!;
          const engine = el.querySelector(".model-identity")!;
          return (
            model.getBoundingClientRect().bottom <=
            engine.getBoundingClientRect().top
          );
        }),
      ).toBe(true);
      await expect(
        chooser.getByRole("status", { name: "Selected local runtime status" }),
      ).toHaveCount(0);
      const sidebarBox = await sidebar.boundingBox();
      const chooserBox = await chooser.boundingBox();
      expect(sidebarBox).not.toBeNull();
      expect(chooserBox).not.toBeNull();
      // Keep useful model content wide in the narrow rail, with breathing room.
      expect(chooserBox!.x - sidebarBox!.x).toBeLessThanOrEqual(13);
      expect(
        sidebarBox!.x + sidebarBox!.width - chooserBox!.x - chooserBox!.width,
      ).toBeLessThanOrEqual(14);

      await expect(chooser.locator(".provider-icon img")).toBeVisible();
      await expect(
        sidebar.getByRole("region", { name: "Workflow summary" }),
      ).toBeVisible();
      await expect(
        sidebar.getByText(
          /Your words, naturally|Bring your words to life|Speak, then get back/,
        ),
      ).toHaveCount(0);
      await expect(sidebar.getByText("Model", { exact: true })).toBeVisible();
      await expect(sidebar.getByRole("combobox")).toHaveCount(0);
      await expect(sidebar.getByRole("slider")).toHaveCount(0);
      await expect(sidebar.getByRole("switch")).toHaveCount(0);
      await expect(
        sidebar.getByRole("button", { name: "Microphone", exact: true }),
      ).toHaveCount(0);
      const original = await chooser.innerText();
      await chooser.click();
      const dialog = page.getByRole("dialog", {
        name: "Engine and model",
        exact: true,
      });
      await expect(
        dialog.getByRole("combobox", {
          name: "Choose connection",
          exact: true,
        }),
      ).toBeVisible();
      const model = dialog.getByRole("combobox", {
        name: "Choose model",
        exact: true,
      });
      const originalModel = await model.inputValue();
      await expect(dialog.getByRole("status")).toContainText(
        "Changes apply immediately to the next request.",
      );
      await model.fill("unsaved/manual-search");
      await page.keyboard.press("Escape");
      await expect(model).toHaveValue(originalModel);
      await page.keyboard.press("Escape");
      await expect(dialog).toBeHidden();
      await expect(chooser).toHaveText(original);
      await expect(chooser).toBeFocused();

      await chooser.click();
      const longModel = `my-server/${"long-model-name-".repeat(8)}model`;
      await model.fill(longModel);
      await page.getByRole("option", { name: /^Use “my-server\// }).click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(model).toHaveValue(longModel);
      await page.keyboard.press("Escape");
      await expect(dialog).toBeHidden();
      await expect(chooser).toContainText(longModel);
      expect(
        await chooser.evaluate((el) => el.scrollWidth <= el.clientWidth),
      ).toBe(true);
      expect(
        await sidebar.evaluate((el) => el.scrollWidth <= el.clientWidth),
      ).toBe(true);
      await page.screenshot({
        path: info.outputPath("simple-model-sidebar.png"),
      });

      await sidebar
        .getByRole("button", { name: "Workflow preferences", exact: true })
        .click();
      const preferences = page.getByRole("dialog", {
        name: "Workflow preferences",
        exact: true,
      });
      await expect(
        preferences.getByRole("button", {
          name: workflow.preferences,
          exact: true,
        }),
      ).toBeVisible();
      if (workflow.name !== "Text to speech") {
        await expect(
          preferences.getByRole("button", {
            name: "Transcript cleanup",
            exact: true,
          }),
        ).toBeVisible();
        await expect(
          preferences.getByRole("button", {
            name: "Personal vocabulary",
            exact: true,
          }),
        ).toBeVisible();
      }
      if (workflow.name === "Audio file")
        await expect(
          preferences.getByRole("switch", {
            name: "Stream partial results",
            exact: true,
          }),
        ).toBeVisible();
      await preferences
        .getByRole("button", { name: workflow.preferences, exact: true })
        .click();
      await expect(preferences).toBeHidden();
      const inspector = page.locator('[data-pane="configuration"]');
      await expect(inspector).toBeVisible();
      if (workflow.name === "Voice transcription") {
        await expect(inspector.locator("#voice-language")).toBeVisible();
        await inspector
          .getByRole("tab", { name: "Audio settings", exact: true })
          .click();
        await expect(inspector.locator("#microphone-select")).toBeVisible();
      }
    });
  }
}

test("model save failure retains the applied selection and permits retry", async ({
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
  const dialog = page.getByRole("dialog", {
    name: "Engine and model",
    exact: true,
  });
  const model = dialog.getByRole("combobox", {
    name: "Choose model",
    exact: true,
  });
  const previousModel = await model.inputValue();
  await model.fill("voice/next");
  await page.getByRole("option", { name: /Use “voice\/next”/ }).click();
  const pending = await saves.waitForStart();
  await expect(model).toBeDisabled();
  await expect(chooser).toHaveText(original);
  await saves.complete(pending, "failure");
  await expect(model).toBeEnabled();
  await expect(dialog.getByRole("status")).toContainText(
    "Your previous settings are still active",
  );
  await page.keyboard.press("Escape");
  await expect(model).toHaveValue(previousModel);
  await expect(chooser).toHaveText(original);
  await model.fill("voice/next");
  await page.getByRole("option", { name: /Use “voice\/next”/ }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(model).toHaveValue("voice/next");
  await page.keyboard.press("Escape");
  await expect(chooser).toContainText("voice/next");
  await expect(page.locator("#workbench-secondary-sidebar")).toBeHidden();
});

test("managed model card uses the existing runtime identity without inferring from model IDs", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?view=workspace&pickers&local-model");
  const chooser = page.getByRole("button", {
    name: "Change engine or model",
    exact: true,
  });
  await expect(chooser).toContainText("NeMo-Speech.cpp");
  await expect(chooser.locator(".model-name")).toHaveText(
    "Nemotron 3.5 Streaming",
  );
  await expect(chooser.locator(".model-connection")).toHaveText(
    "NeMo-Speech.cpp",
  );
  await expect(
    chooser.locator('[data-provider="nemo-speech-cpp"] img'),
  ).toBeVisible();
});

for (const width of [1280, 560]) {
  test(`Output opens delivery settings by keyboard at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/tests/browser/app/?main&workflows&setup-ready");
    const sidebar = page.locator("#workbench-primary-sidebar");
    if (!(await sidebar.isVisible()))
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();
    const output = sidebar.getByRole("button", {
      name: "Output settings",
      exact: true,
    });
    await expect(output).toContainText("Original app · copy if blocked");
    await output.focus();
    await output.press("Enter");
    const inspector = page.locator('[data-pane="configuration"]');
    await expect(inspector).toBeVisible();
    await expect(
      inspector.getByRole("radiogroup", { name: "Transcript delivery mode" }),
    ).toBeVisible();
    await expect(
      page.getByRole("dialog", { name: "Workflow preferences", exact: true }),
    ).toBeHidden();
  });
}
