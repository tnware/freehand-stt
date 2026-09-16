import { test, expect } from "./fixtures";

for (const theme of ["dark", "light"] as const) {
  test(`runtime indicators agree across Connections, inventory and workflow controls in ${theme}`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
    await page.goto(
      `/tests/browser/app/?main&runtime&runtime-ready&theme=${theme}`,
    );
    const rail = page.getByRole("navigation", {
      name: "Workspace",
      exact: true,
    });
    const indicators = '[data-slot="runtime-status"]';
    for (const [state, tone, busy] of [
      ["stopped", "neutral", "false"],
      ["starting", "accent", "true"],
      ["error", "danger", "false"],
      ["running", "success", "false"],
    ]) {
      await page.evaluate(
        (state) => window.testRuntime.change("nemo-default", { state }),
        state,
      );
      await rail
        .getByRole("button", { name: "Connections", exact: true })
        .click();
      const row = page
        .getByRole("navigation", { name: "Saved connections" })
        .getByRole("button", { name: /Local speech/ });
      await row.click();
      const details = page.getByRole("region", {
        name: "Connection details",
        exact: true,
      });
      for (const surface of [row, details]) {
        await expect(surface.locator(indicators)).toHaveAttribute(
          "data-tone",
          tone,
        );
        await expect(surface.locator(indicators)).toHaveAttribute(
          "data-busy",
          busy,
        );
      }
      expect(
        await row
          .locator('[data-slot="status-badge"]')
          .evaluate((el) => el.getBoundingClientRect().height),
      ).toBeLessThanOrEqual(24);
      expect(await row.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(
        true,
      );
      if (state === "starting") {
        await expect(
          details.locator('[data-slot="status-indicator"] svg'),
        ).toHaveCSS("animation-name", "none");
        await page.screenshot({
          path: info.outputPath(`connections-starting-${theme}.png`),
        });
      }
      await rail
        .getByRole("button", { name: "Local runtime", exact: true })
        .click();
      const inventory = page.getByRole("group", {
        name: "Runtime inventory",
        exact: true,
      });
      await expect(inventory.locator(indicators)).toHaveAttribute(
        "data-tone",
        tone,
      );
      await expect(
        page.locator(".workbench-toolbar").locator(indicators),
      ).toHaveAttribute("data-tone", tone);
      await rail
        .getByRole("button", { name: "Voice transcription", exact: true })
        .click();
      const quick = page
        .locator("#workbench-primary-sidebar")
        .getByRole("status", { name: "Local runtime status", exact: true });
      await expect(quick.locator(indicators)).toHaveAttribute(
        "data-tone",
        tone,
      );
      await expect(quick.locator(indicators)).toHaveAttribute(
        "data-busy",
        busy,
      );
      if (state === "error")
        await page.screenshot({
          path: info.outputPath(`voice-error-${theme}.png`),
        });
    }
    await page.setViewportSize({ width: 560, height: 900 });
    await page.evaluate(() =>
      window.testRuntime.change("nemo-default", { state: "starting" }),
    );
    await rail
      .getByRole("button", { name: "Connections", exact: true })
      .click();
    const list = page.getByRole("navigation", { name: "Saved connections" });
    if (!(await list.isVisible()))
      await page
        .getByRole("button", { name: "Toggle primary sidebar", exact: true })
        .click();
    const compactRow = list.getByRole("button", { name: /Local speech/ });
    await expect(compactRow.locator(indicators)).toContainText("Starting");
    expect(
      await compactRow.evaluate((el) => el.scrollWidth <= el.clientWidth),
    ).toBe(true);
    const summary = compactRow.locator("span[title]").last();
    await expect(summary).toHaveCSS("text-overflow", "clip");
    expect(
      await summary.evaluate((el) => el.scrollWidth <= el.clientWidth),
    ).toBe(true);
    await page.screenshot({
      path: info.outputPath(`connections-compact-${theme}.png`),
    });
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  });
}

test("pending Stop stays active when navigating to Connections before backend admission", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page.evaluate(() => window.testRuntime.holdLifecycle());
  const rail = page.getByRole("navigation", { name: "Workspace", exact: true });
  await rail
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await rail.getByRole("button", { name: "Connections", exact: true }).click();
  await page
    .getByRole("navigation", { name: "Saved connections" })
    .getByRole("button", { name: /Local speech/ })
    .click();
  const badge = page
    .getByRole("region", { name: "Connection details", exact: true })
    .locator('[data-slot="runtime-status"]');
  await expect(badge).toContainText("Stopping runtime");
  await expect(badge).toHaveAttribute("data-tone", "accent");
  await expect(badge).toHaveAttribute("data-busy", "true");
  await page.evaluate(() => {
    window.testRuntime.acknowledgeLifecycle("nemo-default");
    window.testRuntime.finishLifecycle("nemo-default");
  });
  await expect(badge).toContainText("Stopped");
  await expect(badge).toHaveAttribute("data-tone", "neutral");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
  ]);
});
