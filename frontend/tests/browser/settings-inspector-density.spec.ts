import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

for (const width of [1280, 560]) {
  test(`Voice inspector keeps every section visible and keyboard reachable at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 820 });
    await page.goto("/tests/browser/app/?main&workflows&runtime&runtime-ready");
    await openSection(page, "voice-transcription");
    const inspector = page.locator('[data-pane="configuration"]');
    const tabs = inspector.getByRole("tablist", { name: "Context settings" });
    await expect(tabs.getByRole("tab")).toHaveCount(6);
    expect(
      await tabs.evaluate((node) => node.scrollWidth <= node.clientWidth),
    ).toBe(true);
    const bounds = await tabs.boundingBox();
    expect(bounds).not.toBeNull();
    for (const tab of await tabs.getByRole("tab").all()) {
      const box = await tab.boundingBox();
      expect(box).not.toBeNull();
      expect(box!.x).toBeGreaterThanOrEqual(bounds!.x);
      expect(box!.x + box!.width).toBeLessThanOrEqual(
        bounds!.x + bounds!.width + 1,
      );
      expect(box!.y + box!.height).toBeLessThanOrEqual(
        bounds!.y + bounds!.height + 1,
      );
    }
    const voice = tabs.getByRole("tab", {
      name: "Voice transcription settings",
      exact: true,
    });
    await voice.focus();
    await voice.press("End");
    const delivery = tabs.getByRole("tab", {
      name: "Transcript delivery settings",
      exact: true,
    });
    await expect(delivery).toBeFocused();
    await expect(delivery).toHaveAttribute("aria-selected", "true");
    await delivery.press("Home");
    await expect(voice).toBeFocused();
    await voice.press("ArrowRight");
    await expect(
      tabs.getByRole("tab", { name: "Audio settings", exact: true }),
    ).toBeFocused();
    await expect(inspector.locator("#microphone-select")).toBeVisible();
  });
}

test("managed Voice presents one engine/model card with runtime actions and advanced controls", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&workflows&runtime&runtime-ready");
  await openSection(page, "voice-transcription");
  const inspector = page.locator('[data-pane="configuration"]');
  await expect(
    inspector.getByRole("group", {
      name: "Selected engine and model",
      exact: true,
    }),
  ).toHaveCount(0);
  const runtime = inspector.getByRole("group", {
    name: "Local runtime Local speech",
    exact: true,
  });
  await expect(runtime).toHaveCount(1);
  await expect(runtime).toContainText("NeMo-Speech.cpp");
  await expect(
    runtime.locator('[data-provider="nemo-speech-cpp"] img'),
  ).toBeVisible();
  await expect(
    runtime.getByRole("button", { name: "Selected model", exact: true }),
  ).toContainText("Nemotron 3.5 Streaming");
  await expect(
    runtime.getByRole("button", { name: "Stop", exact: true }),
  ).toBeVisible();
  await expect(
    runtime.getByRole("button", { name: "Manage runtime", exact: true }),
  ).toBeVisible();
  await expect(
    runtime.getByRole("button", { name: "View output", exact: true }),
  ).toBeVisible();
  await expect(
    inspector
      .locator("summary")
      .filter({ hasText: "NeMo transcription controls" }),
  ).toBeVisible();
});
