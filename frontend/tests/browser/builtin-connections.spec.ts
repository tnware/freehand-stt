import { test, expect } from "./fixtures";
import { installOutputFixture } from "./runtime-output-fixtures";
import { openSection } from "./context-navigation";

test("runtime connections have read-only details and navigate to their owners", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await openSection(page, "connections");
  await page
    .getByRole("navigation", { name: "Saved connections" })
    .getByRole("button", { name: /Local speech/ })
    .click();
  const details = page.getByRole("region", {
    name: "Connection details",
    exact: true,
  });
  await expect(details).toBeVisible();
  const connectionMark = page
    .getByRole("navigation", { name: "Saved connections" })
    .getByRole("button", { name: /Local speech/ })
    .locator("img");
  const markSource = await connectionMark.getAttribute("src");
  const detailMark = details.locator("img");
  await expect(detailMark).toBeVisible();
  await expect(detailMark).toHaveAttribute("src", markSource!);
  expect(
    await detailMark.evaluate(
      (img: HTMLImageElement) => img.complete && img.naturalWidth > 0,
    ),
  ).toBe(true);
  const ownership = details.locator("details").filter({
    has: page.getByText("Connection ownership & safety", { exact: true }),
  });
  await expect(ownership).not.toHaveAttribute("open", "");
  await ownership
    .getByText("Connection ownership & safety", { exact: true })
    .click();
  await expect(ownership).toContainText(
    "Requests never fall back to a remote server.",
  );
  await expect(details).toContainText("nemotron-3.5");
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", { backend: "cpu" }),
  );
  await expect(details.getByText("CPU", { exact: true })).toBeVisible();
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", { backend: "cuda" }),
  );
  await expect(
    details.getByText("NVIDIA GPU (CUDA)", { exact: true }),
  ).toBeVisible();
  await expect(details.locator("input")).toHaveCount(0);
  await expect(details.getByRole("switch")).toHaveCount(0);
  await expect(
    details.getByRole("button", { name: "Connection actions" }),
  ).toHaveCount(0);
  await expect(details.getByRole("button", { name: /^Save/ })).toHaveCount(0);
  await details
    .getByRole("button", {
      name: "Audio-file transcription settings",
      exact: true,
    })
    .click();
  await expect(page.locator("#saved-connection-stt")).toHaveValue(
    "Local speech",
  );
  await page
    .locator('[data-pane="configuration"]')
    .getByRole("button", { name: "Connection details", exact: true })
    .click();
  await details
    .getByRole("button", { name: "Manage runtime", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Local runtime", exact: true }),
  ).toHaveAttribute("aria-current", "page");
  await expect(
    page.locator('[aria-label="Runtime inventory"] img'),
  ).toHaveAttribute("src", markSource!);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("new manual connections never offer managed-target creation, while legacy aliases stay editable", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready&legacy-runtime");
  await openSection(page, "connections");
  await page
    .getByRole("navigation", { name: "Saved connections" })
    .getByRole("button", { name: /Local speech/ })
    .click();
  await expect(page.locator("#connection-name")).toHaveValue("Local speech");
  await expect(page.locator("#connection-instance")).toBeVisible();
  await expect(page.locator("#connection-url")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Save connection", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("button", { name: "Add connection", exact: true })
    .click();
  await expect(page.locator("#connection-url")).toBeVisible();
  await expect(page.locator("#connection-target")).toHaveCount(0);
  await expect(page.locator("#connection-instance")).toHaveCount(0);
});

for (const theme of ["dark", "light"] as const) {
  test(`resource views retain visible controls at compact width in ${theme} mode`, async ({
    page,
  }, testInfo) => {
    await installOutputFixture(page);
    await page.setViewportSize({ width: 860, height: 1000 });
    await page.emulateMedia({ colorScheme: theme });
    await page.goto(`/tests/browser/app/?runtime&runtime-ready&theme=${theme}`);
    await openSection(page, "connections");
    const connections = page.getByRole("navigation", {
      name: "Saved connections",
    });
    await expect(
      connections.getByText("Running", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Add connection", exact: true }),
    ).toBeInViewport();
    await page.screenshot({
      path: testInfo.outputPath(`connections-${theme}.png`),
      fullPage: true,
    });
    await connections.getByRole("button", { name: /Local speech/ }).click();
    await expect(
      connections.getByRole("button", { name: /Local speech/ }),
    ).toHaveAttribute("aria-current", "true");
    await expect(
      page.getByRole("button", { name: "Manage runtime", exact: true }),
    ).toBeInViewport();
    await page.screenshot({
      path: testInfo.outputPath(`connection-details-${theme}.png`),
      fullPage: true,
    });
    await page
      .getByRole("button", { name: "Manage runtime", exact: true })
      .click();
    await expect(page.getByText("Running", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Stop", exact: true }),
    ).toBeInViewport();
    await page
      .getByRole("button", { name: "View output", exact: true })
      .click();
    await expect(
      page.getByRole("region", {
        name: "Read-only process output",
        exact: true,
      }),
    ).toBeInViewport();
    await expect(
      page.getByRole("region", {
        name: "Read-only process output",
        exact: true,
      }),
    ).toContainText("Sensitive startup diagnostic");
    await page.screenshot({
      path: testInfo.outputPath(`runtime-list-${theme}.png`),
      fullPage: true,
    });
    const manage = page.getByRole("button", { name: /^Manage runtime/ });
    await manage.click();
    await expect(manage).toHaveAttribute("aria-expanded", "true");
    await expect(
      page.getByRole("region", { name: "Model catalog" }),
    ).toBeVisible();
    await page.screenshot({
      path: testInfo.outputPath(`runtime-expanded-${theme}.png`),
      fullPage: true,
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  });
}
