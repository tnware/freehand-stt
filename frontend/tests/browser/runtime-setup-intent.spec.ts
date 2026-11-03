import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";
import { openSection } from "./context-navigation";

const rail = (page: Page) =>
  page.getByRole("navigation", { name: "Workspace", exact: true });
const guidance = (page: Page, label: string) =>
  page.getByRole("region", {
    name: `Local ${label} setup guidance`,
    exact: true,
  });
const settings = (page: Page) =>
  page.evaluate(() => window.testConnectionWindows.settings());
async function openSetup(page: Page, purpose: "speech" | "cleanup") {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto(
    "/tests/browser/app/?main&blank&runtime&runtime-all-providers&runtime-combined&runtime-use-flow",
  );
  await openSection(page, purpose === "speech" ? "speech" : "processing");
  await page
    .getByRole("button", { name: `Set up local ${purpose}`, exact: true })
    .click();
  await expect(guidance(page, purpose)).toBeVisible();
}

test("speech setup focuses speech models, keeps both downloads explicit, and returns to the speech picker", async ({
  page,
  saves,
}) => {
  await openSetup(page, "speech");
  const before = await settings(page);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  await expect(
    page.getByRole("button", { name: "Use for Voice", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Install NeMo-Speech.cpp?", exact: true })
    .getByRole("button", { name: "Download and install", exact: true })
    .click();
  const choose = page.getByRole("button", {
    name: "Choose speech model",
    exact: true,
  });
  await expect(choose).toBeEnabled();
  const summary = page.getByRole("region", {
    name: "Runtime setup summary",
    exact: true,
  });
  await expect(summary).toContainText("Speech model");
  await expect(summary).toContainText("No model selected");
  await expect(summary).toContainText("Choose a speech model");
  await expect(
    page.getByRole("button", {
      name: "Download selected transcription model",
      exact: true,
    }),
  ).toHaveCount(0);
  const speechGroup = page.getByRole("button", {
    name: "Text to speech models",
    exact: true,
  });
  await speechGroup.click();
  await expect(speechGroup).toHaveAttribute("aria-expanded", "false");
  await choose.focus();
  await page.keyboard.press("Enter");
  await expect(speechGroup).toHaveAttribute("aria-expanded", "true");
  await expect(
    page.getByRole("button", {
      name: "Model details: Magpie TTS Multilingual",
      exact: true,
    }),
  ).toBeFocused();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "SetInstance:nemo-speech-cpp",
    "Install:nemo-speech-cpp",
  ]);
  expect((await settings(page)).savedConnections.selected).toEqual(
    before.savedConnections.selected,
  );
  await page
    .getByRole("article", { name: "Magpie TTS Multilingual", exact: true })
    .getByRole("button", { name: "Select", exact: true })
    .click();
  const selected = await page.evaluate(
    () => window.testRuntime.snapshot()[0].instance,
  );
  expect(selected.model).toBe("nemotron-3.5");
  expect(selected.speechModel).toBe("magpie-tts");
  await page
    .getByRole("button", {
      name: "Download selected speech model",
      exact: true,
    })
    .click();
  await page.evaluate(() =>
    window.testRuntime.finishDownload("nemo-speech-cpp"),
  );
  await expect(guidance(page, "speech")).toContainText(
    "Download the remaining selected model",
  );
  await page
    .getByRole("button", {
      name: "Download selected transcription model",
      exact: true,
    })
    .click();
  await page.evaluate(() =>
    window.testRuntime.finishDownload("nemo-speech-cpp"),
  );
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await expect(guidance(page, "speech")).toContainText(
    "Choose its built-in connection in Speech settings",
  );
  await expect(
    page.getByRole("button", { name: "Use for Voice", exact: true }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Return to Speech settings", exact: true })
    .click();
  await expect(
    rail(page).getByRole("button", { name: "Local runtime", exact: true }),
  ).not.toHaveAttribute("aria-current", "page");
  expect((await settings(page)).savedConnections.selected).toEqual(
    before.savedConnections.selected,
  );
  expect((await settings(page)).textToSpeech.enabled).toBe(
    before.textToSpeech.enabled,
  );
  await page
    .getByRole("combobox", { name: "Choose connection", exact: true })
    .click();
  await page.getByRole("option", { name: /NeMo-Speech.cpp/ }).click();
  const save = await saves.waitForStart();
  await saves.complete(save, "success");
  const after = await settings(page);
  expect(after.savedConnections.selected?.speech).toBe(
    "builtin-fixture-nemo-speech-cpp",
  );
  expect(after.savedConnections.selected?.voice).toBe(
    before.savedConnections.selected?.voice,
  );
  expect(after.textToSpeech.enabled).toBe(before.textToSpeech.enabled);
  await rail(page)
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await expect(guidance(page, "speech")).toHaveCount(0);
});

test("setup intent is consumed once and never follows an unrelated provider choice", async ({
  page,
}) => {
  await openSetup(page, "cleanup");
  await expect(guidance(page, "cleanup")).toContainText(
    "Install the runtime for cleanup",
  );
  await expect(
    page.getByRole("button", {
      name: "Return to Cleanup settings",
      exact: true,
    }),
  ).toBeVisible();
  const inventory = page.getByRole("group", {
    name: "Runtime inventory",
    exact: true,
  });
  await inventory.getByRole("button", { name: /NeMo-Speech.cpp/ }).click();
  await expect(guidance(page, "cleanup")).toHaveCount(0);
  await expect(guidance(page, "speech")).toHaveCount(0);
  await inventory.getByRole("button", { name: /llama.cpp/ }).click();
  await expect(guidance(page, "cleanup")).toHaveCount(0);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("returning from speech setup and reopening runtime normally clears the setup purpose", async ({
  page,
}) => {
  await openSetup(page, "speech");
  const before = await settings(page);
  await page
    .getByRole("button", { name: "Return to Speech settings", exact: true })
    .click();
  await expect(
    page.getByRole("region", { name: "Local speech setup", exact: true }),
  ).toBeVisible();
  await rail(page)
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await expect(guidance(page, "speech")).toHaveCount(0);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  expect(await settings(page)).toEqual(before);
});
