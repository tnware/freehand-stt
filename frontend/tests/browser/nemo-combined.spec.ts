import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

test("combined runtime waits for both downloads and offers role-specific speech controls", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?main&runtime&runtime-ready&runtime-combined",
  );
  await openSection(page, "local-runtime");
  const catalog = page.getByRole("region", {
    name: "Model catalog",
    exact: true,
  });
  const speech = catalog.getByRole("article", {
    name: "Magpie TTS Multilingual",
    exact: true,
  });
  const asr = catalog.getByRole("article", {
    name: "Nemotron 3.5 Streaming",
    exact: true,
  });
  await expect(speech).toContainText("Selected");
  await expect(asr).toContainText("Selected");
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toHaveCount(0);
  const download = page.getByRole("button", {
    name: "Download selected model",
    exact: true,
  });
  await expect(download).toContainText("Get speech model");
  await download.click();
  await page.evaluate(() => window.testRuntime.finishDownload("nemo-default"));
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await expect(speech).toContainText("Loaded");
  await expect(asr).toContainText("Loaded");
  await expect(
    speech.getByRole("button", { name: "Disable", exact: true }),
  ).toBeDisabled();
  await openSection(page, "speech");
  const pane = page.locator('[data-pane="configuration"]');
  await expect(
    pane.getByLabel("Selected speech model", { exact: true }),
  ).toContainText("Magpie");
  await expect(pane.getByLabel("Speech language", { exact: true })).toHaveValue(
    "Server default",
  );
  await expect(pane).toContainText(
    "Start and Stop affect transcription and speech together",
  );
  const composer = page.getByRole("region", { name: "Speech composer" });
  await expect(composer).not.toContainText("Setup needed");
  await composer.getByRole("textbox").fill("A managed speech request");
  await expect(composer.getByRole("button", { name: /^Speak/ })).toBeEnabled();
});
