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
      `/tests/browser/app/?main&setup-ready&runtime&runtime-provider=whisper-cpp&runtime-whisper-variants${ready ? "&runtime-ready" : ""}`,
    );
    await page.waitForFunction(() => !!window.testRuntime);
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
    await expect(
      recommended.getByText("Model ID", { exact: true }),
    ).toBeVisible();
    await expect(
      recommended.getByRole("link", {
        name: "ggerganov/whisper.cpp",
        exact: true,
      }),
    ).toBeVisible();
    // The action guards are identical while uninstalled and while running.
    for (const action of await catalog
      .getByRole("button", { name: /^(Download|Select|Delete Whisper)/ })
      .all())
      await expect(action).toBeDisabled();
    if (ready) {
      await expect(
        recommended.getByText("Loaded", { exact: true }),
      ).toBeVisible();
      await expect(
        recommended.getByText("Downloaded", { exact: true }),
      ).toBeVisible();
      await expect(
        page.getByText(
          "Stop the runtime to download, remove, or select a model. Start it again when ready.",
          { exact: true },
        ),
      ).toBeVisible();
    } else
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

for (const width of [1280, 560]) {
  test(`selection and download status stay distinct through cancellation and startup at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(
      "/tests/browser/app/?main&setup-ready&runtime&runtime-provider=whisper-cpp&runtime-whisper-variants&runtime-ready",
    );
    await page.waitForFunction(() => !!window.testRuntime);
    await page
      .getByRole("button", { name: "Local runtime", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Refresh inventory", exact: true })
      .click();
    await page.setViewportSize({ width, height: 900 });
    const catalog = page.getByRole("region", {
      name: "Model catalog",
      exact: true,
    });
    const model = catalog.getByRole("article", {
      name: "Whisper Base Q8_0",
      exact: true,
    });
    const select = model.getByRole("button", {
      name: "Select",
      exact: true,
    });
    const download = model.getByRole("button", {
      name: "Download",
      exact: true,
    });
    await expect(select).toBeDisabled();
    await expect(download).toBeDisabled();
    await expect(
      model.getByText("Not downloaded", { exact: true }),
    ).toBeVisible();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);

    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await expect(select).toBeEnabled();
    const selectBox = await select.boundingBox();
    const downloadBox = await download.boundingBox();
    expect(selectBox).not.toBeNull();
    expect(downloadBox).not.toBeNull();
    await select.click();
    await expect(model.getByText("Selected", { exact: true })).toBeVisible();
    await expect(
      model.getByText("Not downloaded", { exact: true }),
    ).toBeVisible();
    await expect(download).toBeEnabled();
    const downloadAfterSelection = await download.boundingBox();
    expect(downloadAfterSelection?.x).toBeCloseTo(downloadBox!.x, 0);
    expect(downloadAfterSelection?.y).toBeCloseTo(downloadBox!.y, 0);
    // A second click at Select's old coordinates must not become Download.
    await page.mouse.click(
      selectBox!.x + selectBox!.width / 2,
      selectBox!.y + selectBox!.height / 2,
    );
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toHaveCount(0);
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      "Stop:whisper-cpp-default",
      "SetInstance:whisper-cpp-default",
    ]);

    await download.click();
    await expect(model.getByText("Selected", { exact: true })).toBeVisible();
    await expect(model.getByText("Downloading", { exact: true })).toBeVisible();
    // Cancel lives with progress, never under the just-clicked Download button.
    await page.mouse.click(
      downloadBox!.x + downloadBox!.width / 2,
      downloadBox!.y + downloadBox!.height / 2,
    );
    expect(await page.evaluate(() => window.testRuntime.calls)).not.toContain(
      "Cancel:whisper-cpp-default",
    );
    await model.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(model.getByText("Selected", { exact: true })).toBeVisible();
    await expect(
      model.getByText("Not downloaded", { exact: true }),
    ).toBeVisible();
    await expect(download).toBeEnabled();

    await download.click();
    await page.evaluate(() =>
      window.testRuntime.finishDownload("whisper-cpp-default"),
    );
    await expect(model.getByText("Selected", { exact: true })).toBeVisible();
    await expect(model.getByText("Downloaded", { exact: true })).toBeVisible();
    await expect(download).toHaveCount(0);
    await page.getByRole("button", { name: "Start", exact: true }).click();
    await expect(model.getByText("Loaded", { exact: true })).toBeVisible();
    await expect(model.getByText("Downloaded", { exact: true })).toBeVisible();
    await expect(model.getByText("Selected", { exact: true })).toHaveCount(0);
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      "Stop:whisper-cpp-default",
      "SetInstance:whisper-cpp-default",
      "DownloadModel:whisper-cpp-default:base-q8_0",
      "Cancel:whisper-cpp-default",
      "DownloadModel:whisper-cpp-default:base-q8_0",
      "Start:whisper-cpp-default",
    ]);
    expect(
      await catalog.evaluate(
        (element) => element.scrollWidth <= element.clientWidth,
      ),
    ).toBe(true);
  });
}

for (const width of [1280, 560]) {
  test(`speech selection cannot turn a repeated click into Disable at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto(
      "/tests/browser/app/?main&runtime&runtime-ready&runtime-combined",
    );
    await page
      .getByRole("button", { name: "Local runtime", exact: true })
      .click();
    const speech = page.getByRole("article", {
      name: "Magpie TTS Multilingual",
      exact: true,
    });
    await speech.getByRole("button", { name: "Disable", exact: true }).click();
    await page.setViewportSize({ width, height: 900 });
    const select = speech.getByRole("button", { name: "Select", exact: true });
    await expect(select).toBeEnabled();
    await select.scrollIntoViewIfNeeded();
    const selectBox = await select.boundingBox();
    const download = speech.getByRole("button", {
      name: "Download",
      exact: true,
    });
    const beforeDownload = await download.boundingBox();
    await select.click();
    await expect(speech.getByText("Selected", { exact: true })).toBeVisible();
    await expect(select).toHaveCount(0);
    const afterDownload = await download.boundingBox();
    expect(afterDownload?.x).toBeCloseTo(beforeDownload!.x, 0);
    expect(afterDownload?.y).toBeCloseTo(beforeDownload!.y, 0);
    const calls = await page.evaluate(() => [...window.testRuntime.calls]);
    await page.mouse.click(
      selectBox!.x + selectBox!.width / 2,
      selectBox!.y + selectBox!.height / 2,
    );
    await expect(speech.getByText("Selected", { exact: true })).toBeVisible();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual(calls);
    await expect(
      speech.getByRole("button", { name: "Disable", exact: true }),
    ).toBeEnabled();
    expect(
      await speech.evaluate(
        (element) => element.scrollWidth <= element.clientWidth,
      ),
    ).toBe(true);
  });
}
