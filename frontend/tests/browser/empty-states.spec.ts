import { test, expect } from "./fixtures";

for (const theme of ["light", "dark"] as const) {
  for (const width of [1280, 560]) {
    test(`empty transcript and recent history fit at ${width}px in ${theme}`, async ({
      page,
    }, info) => {
      await page.setViewportSize({ width, height: 740 });
      await page.emulateMedia({ colorScheme: theme });
      await page.goto(
        `/tests/browser/app/?view=workspace&history=off&theme=${theme}`,
      );
      const result = page.getByRole("region", {
        name: "Current result",
        exact: true,
      });
      await result.getByRole("button", { name: "Clear", exact: true }).click();
      const pane = result.locator('[data-slot="empty-state"]');
      await expect(
        pane.getByText("Speak into the application you’re using"),
      ).toBeInViewport();
      await expect(pane.locator("svg")).toHaveCSS("width", "24px");
      const recent = page.getByRole("region", {
        name: "Transcript history",
        exact: true,
      });
      await expect(recent.getByText("Nothing is being kept")).toBeInViewport();
      await expect(recent.locator('[data-slot="empty-state"] svg')).toHaveCSS(
        "width",
        "16px",
      );
      await page.screenshot({
        path: info.outputPath(`empty-states-${width}-${theme}.png`),
      });

      // A short bottom panel must scroll its action into view without losing its title.
      const splitter = page.getByRole("separator", {
        name: "Resize editor and bottom panel",
        exact: true,
      });
      await splitter.focus();
      await splitter.press("Home");
      await expect(splitter).toHaveAttribute("aria-valuenow", "18");
      const action = recent.getByRole("button", { name: "Turn history on" });
      // The divider value updates before the workbench's next layout frame.
      await expect
        .poll(async () => (await recent.boundingBox())!.height)
        .toBeLessThan(100);
      await action.focus();
      await expect(action).toBeInViewport({ ratio: 1 });
      await page.screenshot({
        path: info.outputPath(`empty-short-${width}-${theme}.png`),
      });
      await action.press("Enter");
      await expect(
        page.getByRole("status").filter({ hasText: /^History settings$/ }),
      ).toHaveCount(1);
      await recent.getByText("Nothing is being kept").scrollIntoViewIfNeeded();
      await expect(recent.getByText("Nothing is being kept")).toBeInViewport();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    });
  }

  test(`History empty search can reset both filters in ${theme}`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.emulateMedia({ colorScheme: theme });
    await page.goto(
      `/tests/browser/app/?main&setup-ready&history-workbench&theme=${theme}`,
    );
    await page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: "History", exact: true })
      .click();
    const sidebar = page.getByRole("complementary", {
      name: "History browser",
      exact: true,
    });
    await sidebar
      .getByRole("button", { name: "Audio files", exact: true })
      .click();
    const search = sidebar.getByRole("searchbox", { name: "Search history" });
    await search.fill("no such transcript");
    const empty = page
      .getByRole("region", { name: "Transcript history", exact: true })
      .locator('[data-slot="empty-state"]');
    await expect(empty.getByText("No matching transcripts.")).toBeInViewport();
    await page.screenshot({
      path: info.outputPath(`history-empty-${theme}.png`),
    });
    await empty.getByRole("button", { name: "Reset filters" }).focus();
    await page.keyboard.press("Enter");
    await expect(search).toHaveValue("");
    await expect(
      sidebar.getByRole("button", { name: "All sources", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
    await expect(
      page.getByRole("region", { name: "Selected transcript", exact: true }),
    ).toBeVisible();
    await expect(empty).toHaveCount(0);
  });
}
