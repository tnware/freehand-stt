import { test, expect } from "./fixtures";
import type { Page } from "@playwright/test";

test("quick runtime controls expose startup elapsed, output and cancel", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?main&runtime&runtime-ready");
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "starting",
      phase: "start",
      startupProgress: {
        phase: "loading_warming",
        startedAt: Date.now() - 5000,
      },
    }),
  );
  await page
    .getByRole("button", { name: "Transcription settings", exact: true })
    .click();
  const panel = page.getByRole("dialog", { name: "Transcription settings" });
  await expect(
    panel
      .getByRole("status")
      .filter({ hasText: "Loading and warming up selected model" }),
  ).toContainText("in this stage");
  await panel.getByRole("button", { name: "View output", exact: true }).click();
  await panel.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "OpenProcessOutput:nemo-default",
    "Cancel:nemo-default",
  ]);
});

async function installRuntime(page: Page) {
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Download selected model", exact: true }),
  ).toBeVisible();
  return page.evaluate(() => {
    const call = window.testRuntime.calls.find((c) =>
      c.startsWith("SetInstance:"),
    );
    if (!call) throw new Error("Instance was not saved");
    return call.slice("SetInstance:".length);
  });
}

for (const backend of ["cuda", "cpu"]) {
  test(`GGML setup probes before explicit ${backend} installation`, async ({
    page,
  }) => {
    await page.goto("/tests/browser/app/?runtime&runtime-provider=whisper-cpp");
    await page.locator('[data-settings-section="local-runtime"]').click();
    await page.getByRole("button", { name: "Install", exact: true }).click();
    await expect(
      page.getByText("Recommended: NVIDIA GPU (CUDA)", { exact: true }),
    ).toBeVisible();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      "GetBinaryOptions:whisper-cpp",
    ]);
    if (backend === "cpu")
      await page.getByRole("button", { name: "CPU", exact: true }).click();
    await page
      .getByRole("button", { name: "Download and install", exact: true })
      .click();
    await expect(
      page.getByRole("button", {
        name: "Download selected model",
        exact: true,
      }),
    ).toBeVisible();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      "GetBinaryOptions:whisper-cpp",
      "SetInstance:whisper-cpp",
      `InstallBackend:whisper-cpp:${backend}`,
    ]);
  });
}

test("startup stage is visible collapsed and expanded with output and cancellation", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?runtime&runtime-ready&runtime-provider=whisper-cpp",
  );
  await page.evaluate(() =>
    window.testRuntime.change("whisper-cpp-default", {
      state: "starting",
      phase: "start",
      startupProgress: { phase: "warming_up", startedAt: Date.now() - 5000 },
    }),
  );
  await page.locator('[data-settings-section="local-runtime"]').click();
  await expect(
    page.getByText(/Warming up selected model · \d+s in this stage/),
  ).toBeVisible();
  await page.getByRole("button", { name: "View output", exact: true }).click();
  await page.getByRole("button", { name: "Manage", exact: true }).click();
  await expect(
    page.getByText(/Warming up selected model · \d+s in this stage/),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Cancel operation", exact: true })
    .click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "OpenProcessOutput:whisper-cpp-default",
    "Cancel:whisper-cpp-default",
  ]);
});

for (const provider of ["llama-cpp", "whisper-cpp"]) {
  test(`${provider} switches CPU to CUDA and back without replacing its selected model`, async ({
    page,
  }, testInfo) => {
    await page.emulateMedia({ colorScheme: "dark" });
    await page.goto(
      `/tests/browser/app/?runtime&runtime-ready&runtime-provider=${provider}&theme=dark`,
    );
    await page.locator('[data-settings-section="local-runtime"]').click();
    const before = await page.evaluate(() => window.testRuntime.snapshot()[0]);
    const connections = await page.evaluate(
      () => window.testConnectionWindows.settings().savedConnections,
    );
    const manage = page.getByRole("button", { name: "Manage", exact: true });
    await manage.click();
    const cpu = page.getByRole("button", { name: "CPU", exact: true });
    const cuda = page.getByRole("button", {
      name: "NVIDIA GPU (CUDA)",
      exact: true,
    });
    await expect(cpu).toHaveAttribute("aria-pressed", "true");
    await expect(cpu).toBeDisabled();
    await expect(cuda).toBeDisabled();
    await expect(
      page.getByRole("button", { name: "Stop", exact: true }),
    ).toBeInViewport();
    if (provider === "llama-cpp") {
      await page.screenshot({
        path: testInfo.outputPath("expanded-running-llama.png"),
        fullPage: true,
      });
    }
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await expect(cuda).toBeEnabled();
    await cuda.click();
    await expect(cuda).toHaveAttribute("aria-pressed", "true");
    await expect(cpu).toHaveAttribute("aria-pressed", "false");
    await manage.click();
    await expect(page.getByText(/NVIDIA GPU \(CUDA\) binary/)).toBeVisible();
    await manage.click();
    await page.getByRole("button", { name: "Start", exact: true }).click();
    await expect(cpu).toBeDisabled();
    await expect(cuda).toBeDisabled();
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await cpu.click();
    await expect(cpu).toHaveAttribute("aria-pressed", "true");
    await expect(cuda).toHaveAttribute("aria-pressed", "false");
    const after = await page.evaluate(() => window.testRuntime.snapshot()[0]);
    expect(after.instance).toEqual(before.instance);
    expect(after.status.selectedModel).toBe(before.status.selectedModel);
    expect(after.status.models).toEqual(before.status.models);
    expect(after.status.backend).toBe("cpu");
    expect(
      await page.evaluate(
        () => window.testConnectionWindows.settings().savedConnections,
      ),
    ).toEqual(connections);
    const selectedModel = page.getByRole("article", {
      name: before.status.models![0].name,
      exact: true,
    });
    await expect(
      selectedModel.getByText("Selected", { exact: true }),
    ).toHaveCount(1);
    await expect(
      selectedModel.getByRole("button", {
        name: "Delete " + before.status.models![0].name,
        exact: true,
      }),
    ).toBeEnabled();
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      `Stop:${before.instance.id}`,
      `InstallBackend:${before.instance.id}:cuda`,
      `Start:${before.instance.id}`,
      `Stop:${before.instance.id}`,
      `InstallBackend:${before.instance.id}:cpu`,
    ]);
  });
}

test("binary switching respects ongoing runtime work and unsaved settings", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?runtime&runtime-ready&runtime-provider=whisper-cpp",
  );
  await page.evaluate(() =>
    window.testRuntime.change("whisper-cpp-default", {
      state: "installing",
      phase: "install",
    }),
  );
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page.getByRole("button", { name: "Manage", exact: true }).click();
  const cuda = page.getByRole("button", {
    name: "NVIDIA GPU (CUDA)",
    exact: true,
  });
  await expect(cuda).toBeDisabled();
  await page.evaluate(() =>
    window.testRuntime.change("whisper-cpp-default", {
      state: "stopped",
      phase: "",
    }),
  );
  await expect(cuda).toBeEnabled();
  await page.locator('[data-settings-section="audio"]').click();
  await page.locator("#max-duration").fill("90");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page.getByRole("button", { name: "Manage", exact: true }).click();
  await cuda.click();
  await expect(page.getByRole("dialog")).toContainText(
    "Save settings before continuing?",
  );
  await page.getByRole("button", { name: "Keep editing", exact: true }).click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  expect(
    await page.evaluate(() => window.testRuntime.snapshot()[0].status.backend),
  ).toBe("cpu");
});

test("active dictation blocks a stopped runtime's binary switch", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?runtime&runtime-ready&runtime-provider=llama-cpp&work-busy",
  );
  await page.evaluate(() =>
    window.testRuntime.change("llama-cpp-default", { state: "stopped" }),
  );
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page.getByRole("button", { name: "Manage", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "NVIDIA GPU (CUDA)", exact: true }),
  ).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("collapsed runtime controls can stop, download and restart without opening details", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="local-runtime"]').click();
  const id = await installRuntime(page);
  const toggle = page.getByRole("button", { name: "Manage", exact: true });
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await page
    .getByRole("button", { name: "Download selected model", exact: true })
    .click();
  await expect(
    page.getByRole("progressbar", { name: "Runtime download progress" }),
  ).toBeInViewport();
  await page
    .getByRole("button", { name: "Cancel operation", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Download selected model", exact: true })
    .click();
  await page.evaluate((id) => window.testRuntime.finishDownload(id), id);
  await page.getByRole("button", { name: "Start", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Stop", exact: true }),
  ).toBeEnabled();
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toBeEnabled();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
});

test("catalog downloads keep progress, cancellation and completion at the model", async ({
  page,
}) => {
  await page.setViewportSize({ width: 900, height: 640 });
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="local-runtime"]').click();
  const id = await installRuntime(page);
  const model = page.getByRole("article", {
    name: "Parakeet TDT v3",
    exact: true,
  });
  await model.scrollIntoViewIfNeeded();
  await model.getByRole("button", { name: "Download", exact: true }).click();
  const progress = model.getByRole("progressbar");
  await expect(progress).toBeInViewport();
  await page.evaluate(
    (id) =>
      window.testRuntime.change(id, {
        acquisition: {
          phase: "downloading",
          bytes: 500_000_000,
          totalBytes: 1_000_000_000,
        },
      }),
    id,
  );
  await expect(progress).toHaveAttribute("value", "50");
  const cancel = model.getByRole("button", { name: "Cancel", exact: true });
  await expect(cancel).toBeInViewport();
  await cancel.click();
  await expect(model.getByRole("status")).toContainText("cancelled");
  await model.getByRole("button", { name: "Download", exact: true }).click();
  await page.evaluate((id) => window.testRuntime.finishDownload(id), id);
  await expect(model.getByRole("status")).toContainText(
    "downloaded and verified",
  );
  await expect(model.getByRole("status")).toBeInViewport();
});

for (const viewport of [
  { width: 1280, height: 720 },
  { width: 900, height: 640 },
]) {
  test(`setup advances in place without scrolling at ${viewport.width}x${viewport.height}`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await page.goto("/tests/browser/app/?runtime");
    await page.locator('[data-settings-section="local-runtime"]').click();
    await expect(
      page.getByRole("button", { name: "Install", exact: true }),
    ).toBeInViewport();
    const instanceID = await installRuntime(page);
    const download = page.getByRole("button", {
      name: "Download selected model",
      exact: true,
    });
    await expect(download).toBeInViewport();
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toHaveCount(0);
    await download.click();
    await expect(
      page.getByRole("progressbar", { name: "Runtime operation progress" }),
    ).toBeInViewport();
    const cancel = page.getByRole("button", {
      name: "Cancel operation",
      exact: true,
    });
    await expect(cancel).toBeInViewport();
    const progress = page.getByRole("progressbar", {
      name: "Runtime operation progress",
    });
    for (const bytes of [250_000_000, 750_000_000]) {
      await page.evaluate(
        ({ id, bytes }) =>
          window.testRuntime.change(id, {
            acquisition: {
              phase: "downloading",
              bytes,
              totalBytes: 1_000_000_000,
            },
          }),
        { id: instanceID, bytes },
      );
      await expect(progress).toHaveAttribute(
        "value",
        String(bytes / 10_000_000),
      );
    }
    await cancel.click();
    const setup = page.getByRole("region", {
      name: "Runtime setup",
      exact: true,
    });
    await expect(setup.getByText(/Operation cancelled\./)).toBeInViewport();
    await expect(download).toBeInViewport();
    await download.click();
    await page.evaluate(
      (id) =>
        window.testRuntime.change(id, {
          acquisition: {
            phase: "verifying",
            bytes: 1_000_000_000,
            totalBytes: 1_000_000_000,
          },
        }),
      instanceID,
    );
    await expect(progress).toHaveCount(0);
    await expect(page.getByText(/Model downloaded and verified\./)).toHaveCount(
      0,
    );
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toHaveCount(0);
    await page.evaluate(
      (id) => window.testRuntime.finishDownload(id),
      instanceID,
    );
    await expect(
      setup.getByText(/Model downloaded and verified\./),
    ).toBeInViewport();
    expect(await page.evaluate(() => window.testRuntime.calls)).not.toContain(
      `Start:${instanceID}`,
    );
    const start = page.getByRole("button", {
      name: "Start",
      exact: true,
    });
    await expect(start).toBeInViewport();
    await start.click();
    await expect(
      page.getByRole("button", { name: "Stop", exact: true }),
    ).toBeInViewport();
  });
}

test("local runtime is discoverable and browsing never downloads", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await expect(
    page.getByRole("heading", { name: "Managed runtimes", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  await installRuntime(page);
  await expect(
    page.getByRole("region", { name: "Model catalog" }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Refresh catalog", exact: true })
    .click();
  expect(
    await page.evaluate(() =>
      window.testRuntime.calls.some(
        (call) =>
          call.startsWith("DownloadModel:") || call.startsWith("Start:"),
      ),
    ),
  ).toBe(false);
});

test("unsaved drafts guard immediate runtime operations", async ({ page }) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.locator('[data-settings-section="audio"]').click();
  await page.locator("#max-duration").fill("90");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await page.getByRole("button", { name: "Manage", exact: true }).click();
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(page.getByRole("dialog")).toContainText(
    "Save settings before continuing?",
  );
  await page.getByRole("button", { name: "Keep editing", exact: true }).click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  await page.locator('[data-settings-section="audio"]').click();
  await expect(page.locator("#max-duration")).toHaveValue("90");
});

test("unsupported hosts never offer an enabled Windows runtime", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&platform=darwin");
  await page.locator('[data-settings-section="local-runtime"]').click();
  await expect(
    page.getByText("Unavailable on this platform", {
      exact: false,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", {
      name: "Install",
      exact: true,
    }),
  ).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});
