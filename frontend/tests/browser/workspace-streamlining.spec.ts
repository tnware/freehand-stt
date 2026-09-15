import { test, expect } from "./fixtures";
for (const width of [560, 1156]) {
  test(`history keeps common actions compact and secondary actions accessible at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await page.goto(
      "/tests/browser/app/?view=workspace&theme=dark&history=review",
    );
    await page.getByRole("button", { name: "History", exact: true }).click();
    const sidebar = page.getByRole("complementary", {
      name: "History browser",
      exact: true,
    });
    const entries = sidebar.getByRole("button", {
      name: /^View Voice transcript from /,
    });
    async function selectEntry(index: number) {
      if (width < 700) {
        await page
          .getByRole("button", { name: "Toggle primary sidebar", exact: true })
          .click();
      }
      await entries.nth(index).click();
    }
    const first = page.getByRole("region", {
      name: "Selected transcript",
      exact: true,
    });
    const footer = first.locator(".history-footer");
    await expect(
      first.getByRole("button", {
        name: "Copy cleaned transcript",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      first.getByRole("button", { name: "Listen to transcript", exact: true }),
    ).toBeVisible();
    await selectEntry(1);
    await expect(
      first.getByText("Cleanup could not finish", { exact: false }),
    ).toBeVisible();
    await selectEntry(0);
    const metadata = await footer.locator(":scope > div").first().boundingBox();
    const actions = await footer.locator(".history-actions").boundingBox();
    expect(
      Math.abs(
        metadata!.y + metadata!.height / 2 - actions!.y - actions!.height / 2,
      ),
    ).toBeLessThan(2);
    await first
      .getByRole("button", { name: "Compare raw and cleaned transcripts" })
      .click();
    await expect(
      first.getByRole("region", { name: "Raw transcript", exact: true }),
    ).toBeVisible();
    await first
      .getByRole("button", { name: "Copy raw transcript", exact: true })
      .click();
    await expect(
      first.getByRole("button", { name: "Raw transcript copied", exact: true }),
    ).toBeVisible();
    if (width < 1100)
      await page
        .getByRole("button", { name: "Toggle secondary sidebar", exact: true })
        .click();
    await expect(
      page.getByRole("region", { name: "Run information", exact: true }),
    ).toContainText("speech/stt");
    if (width < 1100) {
      await page.keyboard.press("Escape");
      await expect(
        page.getByRole("region", { name: "Run information", exact: true }),
      ).toHaveCount(0);
    }
    const menu = first.getByRole("button", {
      name: "Transcript actions",
      exact: true,
    });
    await menu.focus();
    await menu.press("Enter");
    await expect(
      page.getByRole("menuitem", {
        name: "Transcription details",
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("menuitem", { name: "Remove from history", exact: true }),
    ).toBeInViewport();
    await page.screenshot({
      path: info.outputPath(`history-menu-${width}.png`),
    });
    await page.keyboard.press("Escape");
    await expect(menu).toBeFocused();
    await menu.click();
    await page
      .getByRole("menuitem", { name: "Remove from history", exact: true })
      .click();
    await expect(
      first.getByText("Cleanup could not finish", { exact: false }),
    ).toBeVisible();
    await expect(
      first.getByRole("button", { name: "Copy transcript", exact: true }),
    ).toBeVisible();
    if (width < 700) {
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();
    }
    await expect(entries).toHaveCount(1);
    await expect(entries.first()).toHaveAttribute("aria-current", "true");
    if (width < 700) await entries.first().click();
    await page.screenshot({
      path: info.outputPath(`history-warning-${width}.png`),
    });
  });

  test(`file error details are readable without shifting the workspace at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await page.goto("/tests/browser/app/?view=workspace&theme=dark&file-error");
    await page.getByRole("button", { name: "Audio file", exact: true }).click();
    const transport = page.getByRole("region", {
      name: "Audio file",
      exact: true,
    });
    const before = await transport.boundingBox();
    const trigger = page.getByRole("button", {
      name: "File transcription error details",
      exact: true,
    });
    await trigger.focus();
    await trigger.press("Enter");
    const details = page.getByRole("dialog", {
      name: "File transcription error details",
      exact: true,
    });
    await expect(details).toBeInViewport();
    await expect(details).toContainText("then retry this file.");
    expect(await transport.boundingBox()).toEqual(before);
    await page.screenshot({ path: info.outputPath(`file-error-${width}.png`) });
    await page.keyboard.press("Escape");
    await expect(trigger).toBeFocused();
    await page.getByRole("button", { name: "Retry", exact: true }).click();
    await expect(trigger).toHaveCount(0);
    await expect(transport).toHaveAttribute("data-state", "uploading");
    await expect(
      page.getByRole("region", { name: "Current result" }),
    ).toBeInViewport();
  });
}
