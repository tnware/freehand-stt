import { test, expect } from "./fixtures";

test("host metrics keep the status bar stable and expose accessible detail", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  const trigger = page.getByRole("button", {
    name: "This computer: CPU 42%, RAM 75%, GPU 88%",
    exact: true,
  });
  await expect(trigger).toBeVisible();
  const before = await trigger.boundingBox();
  await trigger.click();
  const detail = page.getByRole("dialog", {
    name: "This computer’s resources",
  });
  await expect(detail.getByRole("meter", { name: "CPU use" })).toHaveAttribute(
    "aria-valuenow",
    "42",
  );
  await expect(detail.getByText("12.0 GiB / 16.0 GiB used")).toBeVisible();
  await expect(detail.getByText(/Remote servers/)).toBeVisible();
  await page.evaluate(() => {
    window.testResources.sample.cpuPercent = 100;
    window.testResources.sample.memoryAvailableBytes = 0;
  });
  await expect(detail.getByText(/High CPU use/)).toBeVisible();
  await expect(detail.getByText(/10% or less remaining/)).toBeVisible();
  const after = await page
    .getByRole("button", {
      name: "This computer: CPU 100%, RAM 100%, GPU 88%",
      exact: true,
    })
    .boundingBox();
  expect(after?.width).toBe(before?.width);
  await page.keyboard.press("Escape");
  await expect(detail).not.toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "This computer: CPU 100%, RAM 100%, GPU 88%",
      exact: true,
    }),
  ).toBeFocused();
});

test("sampling stops on minimize and hide and resumes on reveal", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  await expect(
    page.getByRole("button", {
      name: "This computer: CPU 42%, RAM 75%, GPU 88%",
      exact: true,
    }),
  ).toBeVisible();
  for (const [hide, show] of [
    ["WindowMinimise", "WindowUnMinimise"],
    ["WindowHide", "WindowShow"],
  ]) {
    await page.evaluate(
      (name) => window.testWindow.emit(`common:${name}`),
      hide,
    );
    const reads = await page.evaluate(() => window.testResources.reads);
    await expect(
      page.getByRole("button", {
        name: "This computer: CPU —, RAM —, GPU —",
        exact: true,
      }),
    ).toBeVisible();
    await page.waitForTimeout(2300);
    expect(await page.evaluate(() => window.testResources.reads)).toBe(reads);
    await page.evaluate(
      (name) => window.testWindow.emit(`common:${name}`),
      show,
    );
    await expect(
      page.getByRole("button", {
        name: "This computer: CPU 42%, RAM 75%, GPU 88%",
        exact: true,
      }),
    ).toBeVisible();
  }
});

test("failed samples show unavailable and recover without zero readings", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  await page
    .getByRole("button", {
      name: "This computer: CPU 42%, RAM 75%, GPU 88%",
      exact: true,
    })
    .click();
  const detail = page.getByRole("dialog", {
    name: "This computer’s resources",
  });
  await page.evaluate(() => {
    window.testResources.fail = true;
  });
  await expect(detail.getByText("Unavailable", { exact: true })).toHaveCount(3);
  await expect(detail.getByRole("meter")).toHaveCount(0);
  await page.evaluate(() => {
    window.testResources.fail = false;
  });
  await expect(detail.getByRole("meter")).toHaveCount(4);
});

test("resource details fit in a compact workspace", async ({ page }) => {
  await page.setViewportSize({ width: 520, height: 640 });
  await page.goto("/tests/browser/app/?main&setup-ready");
  await page
    .getByRole("button", {
      name: "This computer: CPU 42%, RAM 75%, GPU 88%",
      exact: true,
    })
    .click();
  const detail = page.getByRole("dialog", {
    name: "This computer’s resources",
  });
  await expect(detail).toBeVisible();
  const bounds = await detail.boundingBox();
  expect(bounds!.x).toBeGreaterThanOrEqual(0);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(520);
  expect(bounds!.y).toBeGreaterThanOrEqual(0);
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth),
  ).toBeLessThanOrEqual(520);
});

test("GPU activity and dedicated memory report high usage independently", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready");
  await page
    .getByRole("button", {
      name: "This computer: CPU 42%, RAM 75%, GPU 88%",
      exact: true,
    })
    .click();
  const detail = page.getByRole("dialog", {
    name: "This computer’s resources",
  });
  await expect(
    detail.getByRole("meter", { name: "Fixture GPU GPU use" }),
  ).toHaveAttribute("aria-valuenow", "88");
  await expect(detail.getByText("10.0 GiB / 12.0 GiB")).toBeVisible();
  await page.evaluate(() => {
    window.testResources.sample.gpus![0].utilizationPercent = 96;
    window.testResources.sample.gpus![0].memoryUsedBytes = 11.5 * 1024 ** 3;
  });
  await expect(
    detail.getByText("High GPU activity · 90% or more."),
  ).toBeVisible();
  await expect(
    detail.getByText("Dedicated GPU memory is nearly full."),
  ).toBeVisible();
  await page.evaluate(() => {
    window.testResources.sample.gpus![0].utilizationAvailable = false;
  });
  await expect(
    detail.getByRole("meter", { name: "Fixture GPU GPU use" }),
  ).toHaveCount(0);
  await expect(
    detail.getByRole("meter", { name: "Fixture GPU dedicated memory use" }),
  ).toBeVisible();
});

test("Apple GPU shared memory has no invented VRAM capacity", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&setup-ready&platform=darwin");
  await page.evaluate(() => {
    const gpu = window.testResources.sample.gpus![0];
    gpu.name = "Apple GPU";
    gpu.unifiedMemory = true;
    gpu.memoryTotalBytes = 0;
    gpu.memoryUsedBytes = 6 * 1024 ** 3;
    gpu.utilizationPercent = 0;
  });
  await page
    .getByRole("button", {
      name: "This computer: CPU 42%, RAM 75%, GPU 0%",
      exact: true,
    })
    .click();
  const detail = page.getByRole("dialog", {
    name: "This computer’s resources",
  });
  await expect(detail.getByText("GPU memory · shared RAM")).toBeVisible();
  await expect(detail.getByText("6.0 GiB", { exact: true })).toBeVisible();
  await expect(detail.getByText(/no separate VRAM pool/)).toBeVisible();
  await expect(
    detail.getByRole("meter", { name: /dedicated memory/ }),
  ).toHaveCount(0);
  await expect(
    detail.getByRole("meter", { name: "Apple GPU GPU use" }),
  ).toHaveAttribute("aria-valuenow", "0");
});
