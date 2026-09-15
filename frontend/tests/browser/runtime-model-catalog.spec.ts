import { test, expect } from "./fixtures";

for (const { width, ready } of [
  { width: 1280, ready: false },
  { width: 560, ready: true },
]) {
  test(`grouped Whisper catalog remains explicit and searchable at ${width}px`, async ({
    page,
  }) => {
    // Load inventory at desktop width before exercising the compact content.
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(
      `/tests/browser/app/?main&setup-ready&runtime&runtime-provider=whisper-cpp${ready ? "&runtime-ready" : ""}`,
    );
    await page.waitForFunction(() => !!window.testRuntime);
    await page.evaluate(() => window.testRuntime.useWhisperVariantCatalog());
    await page
      .getByRole("button", { name: "Local runtime", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Refresh inventory", exact: true })
      .click();
    const catalog = page.getByRole("region", {
      name: "Model catalog",
      exact: true,
    });
    const filter = page.getByRole("searchbox", {
      name: "Filter models",
      exact: true,
    });
    await expect(filter).toBeVisible();
    await page.setViewportSize({ width, height: 900 });

    const base = catalog.getByRole("button", {
      name: "Base models",
      exact: true,
    });
    const tiny = catalog.getByRole("button", {
      name: "Tiny models",
      exact: true,
    });
    await expect(base).toHaveAttribute("aria-expanded", "true");
    await expect(tiny).toHaveAttribute("aria-expanded", "false");
    await expect(catalog.getByRole("article")).toHaveCount(3);
    const recommended = catalog.getByRole("article", {
      name: "Whisper Base",
      exact: true,
    });
    await expect(
      recommended.getByText("Recommended", { exact: true }),
    ).toBeVisible();
    await expect(
      recommended.getByText(
        "Multilingual baseline for completed transcription, balancing size and accuracy.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(recommended.getByRole("link")).toHaveCount(0);
    const details = recommended.getByRole("button", {
      name: "Model details: Whisper Base",
      exact: true,
    });
    await expect(details).toHaveAttribute("aria-expanded", "false");
    await details.focus();
    await page.keyboard.press("Enter");
    await expect(details).toHaveAttribute("aria-expanded", "true");
    await expect(recommended.getByText("Model ID:")).toBeVisible();
    await expect(
      recommended.getByRole("link", {
        name: "ggerganov/whisper.cpp",
        exact: true,
      }),
    ).toBeVisible();
    // The action guards are identical while uninstalled and while running.
    for (const action of await catalog
      .getByRole("button", { name: /^(Get|Select|Delete Whisper)/ })
      .all())
      await expect(action).toBeDisabled();
    if (ready)
      await expect(
        recommended.getByText("Loaded", { exact: true }),
      ).toBeVisible();
    else
      await expect(
        page.getByRole("button", { name: "Refresh catalog", exact: true }),
      ).toBeDisabled();

    await tiny.focus();
    await page.keyboard.press("Enter");
    await expect(tiny).toHaveAttribute("aria-expanded", "true");
    await expect(catalog.getByRole("article")).toHaveCount(5);
    await page.keyboard.press("Space");
    await expect(tiny).toHaveAttribute("aria-expanded", "false");

    await filter.fill("TINY");
    await expect(tiny).toHaveAttribute("aria-expanded", "true");
    await expect(catalog.getByRole("article")).toHaveCount(2);
    await tiny.click();
    await expect(tiny).toHaveAttribute("aria-expanded", "false");
    await expect(catalog.getByRole("article")).toHaveCount(0);
    // A changed search reopens matching groups; ID searches are case-insensitive.
    await filter.fill("TINY.EN-Q5_1");
    await expect(tiny).toHaveAttribute("aria-expanded", "true");
    await expect(catalog.getByRole("article")).toHaveCount(1);
    await expect(
      catalog.getByText(
        "English-only Q5_1 variant with a smaller download and memory footprint.",
        { exact: true },
      ),
    ).toBeVisible();
    await filter.fill("Whisper Large v3 Turbo");
    await expect(
      catalog.getByRole("article", {
        name: "Whisper Large v3 Turbo Q8_0",
        exact: true,
      }),
    ).toBeVisible();
    await filter.fill("not-a-model");
    await expect(catalog.getByRole("status")).toHaveText(
      "No models match your filter.",
    );
    await page
      .getByRole("button", { name: "Clear model filter", exact: true })
      .click();
    await expect(filter).toHaveValue("");
    await expect(filter).toBeFocused();
    await expect(base).toHaveAttribute("aria-expanded", "true");
    await expect(tiny).toHaveAttribute("aria-expanded", "false");
    await expect(catalog.getByRole("article")).toHaveCount(3);
    expect(
      await catalog.evaluate(
        (element) => element.scrollWidth <= element.clientWidth,
      ),
    ).toBe(true);
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  });
}
