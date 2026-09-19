import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

for (const theme of ["light", "dark"] as const) {
  test(`transcript toolbar retains its established heading style in ${theme}`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.emulateMedia({ colorScheme: theme });
    await page.goto(
      `/tests/browser/app/?view=workspace&history=off&theme=${theme}`,
    );
    const transcript = page.getByRole("region", { name: "Current result" });
    const heading = transcript.getByRole("heading", {
      name: "Transcript",
      exact: true,
    });
    await expect(heading).toHaveCSS("font-size", "16px");
    await expect(heading).toHaveCSS("font-weight", "600");
    await expect(transcript.locator(".workbench-toolbar").first()).toHaveCSS(
      "background-color",
      "rgba(0, 0, 0, 0)",
    );
    await page.screenshot({ path: info.outputPath(`headings-${theme}.png`) });
  });

  test(`palette commands carry icons and remain keyboard usable in ${theme}`, async ({
    page,
  }, info) => {
    await page.emulateMedia({ colorScheme: theme });
    await page.goto(
      `/tests/browser/app/?main&setup-ready&workflows&theme=${theme}`,
    );
    await page.getByRole("button", { name: /^Run a command/ }).click();
    const search = page.getByRole("combobox", { name: "Command", exact: true });
    for (const name of [
      "Toggle primary sidebar",
      "Toggle bottom panel",
      "Toggle secondary sidebar",
      "Start recording",
      "Turn cleanup",
    ]) {
      await search.fill(name);
      const option = page.getByRole("option").first();
      await expect(option).toBeVisible();
      await expect(option.locator('svg[aria-hidden="true"]')).toHaveCount(1);
    }
    await search.fill("");
    await page.screenshot({ path: info.outputPath(`palette-${theme}.png`) });
    await search.fill("Toggle primary sidebar");
    await search.press("Enter");
    await expect(
      page.getByRole("dialog", { name: "Commands", exact: true }),
    ).toBeHidden();
    await expect(page.locator("#workbench-primary-sidebar")).toBeHidden();
  });

  test(`segmented settings retain equal columns in a narrow ${theme} pane`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width: 560, height: 740 });
    await page.emulateMedia({ colorScheme: theme });
    await page.goto(`/tests/browser/app/?theme=${theme}`);
    await openSection(page, "overlay");
    const group = page.getByRole("group", {
      name: "Overlay layout",
      exact: true,
    });
    await expect(group).toHaveCSS("display", "grid");
    const widths = await group
      .locator("button")
      .evaluateAll((nodes) =>
        nodes.map((node) => node.getBoundingClientRect().width),
      );
    expect(widths).toHaveLength(4);
    expect(Math.max(...widths) - Math.min(...widths)).toBeLessThan(1);
    await group.locator("button").last().focus();
    await page.keyboard.press("Space");
    await expect(group.locator("button").last()).toHaveAttribute(
      "aria-checked",
      "true",
    );
    await page.screenshot({ path: info.outputPath(`segments-${theme}.png`) });
  });
}

test("history facts stack with aligned values and wrapping literals in the inspector", async ({
  page,
}, info) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto(
    "/tests/browser/app/?main&setup-ready&history-workbench&theme=dark",
  );
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "History", exact: true })
    .click();
  const facts = page.getByRole("region", {
    name: "Run information",
    exact: true,
  });
  await expect(facts).toBeVisible();
  await expect(facts.locator("dd").first()).toHaveCSS("text-align", "left");
  await expect(facts.locator(".mono-chip").first()).toHaveCSS(
    "overflow-wrap",
    "anywhere",
  );
  expect(
    await facts.evaluate((node) => node.scrollWidth <= node.clientWidth),
  ).toBe(true);
  await page.screenshot({ path: info.outputPath("history-facts.png") });
});

test("connection picker footer remains reachable within a short viewport", async ({
  page,
}, info) => {
  await page.setViewportSize({ width: 860, height: 500 });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto(
    "/tests/browser/app/?main&setup-ready&workflows&pickers&theme=dark",
  );
  await page
    .locator("#workbench-primary-sidebar")
    .getByRole("button", { name: "Show connections" })
    .first()
    .click();
  const popup = page.locator('[data-slot="combobox-content"]');
  await expect(popup).toBeVisible();
  await expect(
    popup.getByRole("button", { name: "Manage connections…", exact: true }),
  ).toBeInViewport();
  expect(
    await popup.evaluate((node) => node.scrollHeight <= node.clientHeight + 1),
  ).toBe(true);
  const bounds = (await popup.boundingBox())!;
  expect(bounds.y).toBeGreaterThanOrEqual(0);
  expect(bounds.y + bounds.height).toBeLessThanOrEqual(500);
  await page.screenshot({ path: info.outputPath("connection-picker.png") });
});
