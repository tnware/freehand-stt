import { test, expect } from "./fixtures";

for (const width of [760, 1100]) {
  test(`runtime toolbar keeps actions together and moves raw IDs into preferences at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 820 });
    await page.goto(
      "/tests/browser/app/?main&runtime&runtime-ready&runtime-combined",
    );
    await page
      .getByRole("button", { name: "Local runtime", exact: true })
      .click();
    const raw =
      "provider-private-model-identifier-that-should-not-widen-the-runtime-toolbar";
    await page.evaluate(
      (id) =>
        window.testRuntime.change("nemo-default", {
          selectedModel: id,
          selectedSpeechModel: `${id}-speech`,
        }),
      raw,
    );
    const toolbar = page.getByRole("group", {
      name: "Runtime status and actions",
      exact: true,
    });
    await expect(toolbar).not.toContainText(raw);
    const output = toolbar.getByRole("button", {
      name: "View output",
      exact: true,
    });
    const connections = toolbar.getByRole("button", {
      name: "Open Connections",
      exact: true,
    });
    await expect(output).toBeVisible();
    await expect(connections).toBeVisible();
    const outputBox = await output.boundingBox();
    const connectionBox = await connections.boundingBox();
    expect(outputBox && connectionBox).toBeTruthy();
    expect(Math.abs(outputBox!.y - connectionBox!.y)).toBeLessThan(2);
    await page.getByRole("button", { name: /^Runtime preferences/ }).click();
    const identifiers = page.locator('dl[aria-label="Runtime identifiers"]');
    await expect(identifiers).toContainText(raw);
    await expect(identifiers).toContainText(`${raw}-speech`);
    await expect(identifiers).toContainText("nemotron-3.5");
    await expect(identifiers).toContainText("magpie-tts");
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  });
}

test("CPU remains current while Metal is recommended until an explicit switch", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?main&runtime&runtime-ready&runtime-provider=llama-cpp&runtime-macos",
  );
  await page
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  const toolbar = page.getByRole("group", {
    name: "Runtime status and actions",
    exact: true,
  });
  await expect(toolbar).toContainText("CPU");
  await expect(toolbar).not.toContainText("Metal");
  await page.getByRole("button", { name: /^Runtime preferences/ }).click();
  await expect(
    page.getByLabel("Current runtime backend", { exact: true }),
  ).toContainText("Running with: CPU");
  await expect(
    page.getByText("Recommended for this computer: Apple GPU (Metal).", {
      exact: true,
    }),
  ).toBeVisible();
  const cpu = page.getByRole("button", { name: "CPU", exact: true });
  const metal = page.getByRole("button", {
    name: "Apple GPU (Metal)",
    exact: true,
  });
  await expect(cpu).toHaveAttribute("aria-pressed", "true");
  await expect(metal).toHaveAttribute("aria-pressed", "false");
  await expect(metal).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "GetBinaryOptions:llama-cpp",
  ]);
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await metal.click();
  await expect(metal).toHaveAttribute("aria-pressed", "true");
  await expect(toolbar).toContainText("Apple GPU (Metal)");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "GetBinaryOptions:llama-cpp",
    "Stop:llama-cpp-default",
    "InstallBackend:llama-cpp-default:metal",
  ]);
});

test("installation preview distinguishes explicit CPU choice from host recommendation", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?main&runtime&runtime-provider=llama-cpp&runtime-macos",
  );
  await page
    .getByRole("button", { name: "Local runtime", exact: true })
    .click();
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await expect(
    page.getByText("Recommended: Apple GPU (Metal)", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "CPU", exact: true }).click();
  await expect(
    page.getByLabel("Selected installation backend", { exact: true }),
  ).toContainText("Will install: CPU");
  await expect(
    page.getByText("Recommended: Apple GPU (Metal)", { exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "GetBinaryOptions:llama-cpp",
  ]);
  expect(await page.evaluate(() => window.testRuntime.snapshot())).toEqual([]);
});
