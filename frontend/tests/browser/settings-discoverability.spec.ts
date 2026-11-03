import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

test("general settings search reaches model controls in another workflow", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&workflows&setup-ready&runtime");
  await openSection(page, "general");
  await page.getByPlaceholder("Find settings…").fill("model");
  await expect(
    page.locator('[data-settings-section="voice-transcription"]'),
  ).toBeVisible();
  await expect(
    page.locator('[data-settings-section="local-runtime"]'),
  ).toBeVisible();
  await page.locator('[data-settings-section="local-runtime"]').click();
  await expect(
    page.getByRole("heading", { name: "NeMo-Speech.cpp", exact: true }),
  ).toBeVisible();
});

test("unconfigured model card does not claim a provider", async ({ page }) => {
  await page.goto("/tests/browser/app/?main&blank&setup-ready");
  const chooser = page.getByRole("button", {
    name: "Change engine or model",
    exact: true,
  });
  await expect(chooser).toContainText("No connection selected");
  await expect(chooser).not.toContainText("Generic OpenAI-compatible");
  await expect(chooser.locator(".model-identity")).toHaveCount(0);
});
