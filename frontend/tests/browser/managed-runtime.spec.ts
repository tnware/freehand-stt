import { openSection } from "./context-navigation";
import { test, expect } from "./fixtures";
import { installOutputFixture } from "./runtime-output-fixtures";
import type { Page } from "@playwright/test";

const openRuntimes = (page: Page) =>
  page.getByRole("button", { name: "Local runtime", exact: true }).click();
const preferences = (page: Page) =>
  page.getByRole("button", { name: /^Runtime preferences/ });

async function installRuntime(page: Page) {
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Download selected model", exact: true }),
  ).toBeVisible();
  return page.evaluate(() => window.testRuntime.snapshot()[0].instance.id);
}

for (const backend of ["cuda", "cpu"]) {
  test(`GGML setup probes before explicit ${backend} installation`, async ({
    page,
  }) => {
    await page.goto("/tests/browser/app/?runtime&runtime-provider=whisper-cpp");
    await openRuntimes(page);
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

test("startup stage, elapsed time and cancellation remain visible", async ({
  page,
}) => {
  await installOutputFixture(page);
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
  await openRuntimes(page);
  await expect(
    page.getByText(/Warming up selected model · \d+s in this stage/),
  ).toBeVisible();
  await page.getByRole("button", { name: "View output", exact: true }).click();
  await expect(
    page.getByRole("region", { name: "Read-only process output", exact: true }),
  ).toContainText("whisper-cpp-default");
  await page
    .getByRole("button", { name: "Cancel operation", exact: true })
    .click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Cancel:whisper-cpp-default",
  ]);
});

for (const provider of ["llama-cpp", "whisper-cpp"]) {
  test(`${provider} switches CPU and CUDA while retaining models and Connections`, async ({
    page,
  }) => {
    await page.goto(
      `/tests/browser/app/?runtime&runtime-ready&runtime-provider=${provider}`,
    );
    await openRuntimes(page);
    const before = await page.evaluate(() => window.testRuntime.snapshot()[0]);
    const connections = await page.evaluate(
      () => window.testConnectionWindows.settings().savedConnections,
    );
    await preferences(page).click();
    const cpu = page.getByRole("button", { name: "CPU", exact: true });
    const cuda = page.getByRole("button", {
      name: "NVIDIA GPU (CUDA)",
      exact: true,
    });
    await expect(cpu).toHaveAttribute("aria-pressed", "true");
    await expect(cpu).toBeDisabled();
    await expect(cuda).toBeDisabled();
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await cuda.click();
    await expect(cuda).toHaveAttribute("aria-pressed", "true");
    await page.getByRole("button", { name: "Start", exact: true }).click();
    await expect(cpu).toBeDisabled();
    await expect(cuda).toBeDisabled();
    await page.getByRole("button", { name: "Stop", exact: true }).click();
    await cpu.click();
    await expect(cpu).toHaveAttribute("aria-pressed", "true");
    const after = await page.evaluate(() => window.testRuntime.snapshot()[0]);
    expect(after.instance).toEqual(before.instance);
    expect(after.status.models).toEqual(before.status.models);
    expect(after.status.backend).toBe("cpu");
    expect(
      await page.evaluate(
        () => window.testConnectionWindows.settings().savedConnections,
      ),
    ).toEqual(connections);
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
      `Stop:${before.instance.id}`,
      `InstallBackend:${before.instance.id}:cuda`,
      `Start:${before.instance.id}`,
      `Stop:${before.instance.id}`,
      `InstallBackend:${before.instance.id}:cpu`,
    ]);
    if (provider === "llama-cpp")
      await expect(
        page
          .getByRole("article", { name: "S1-mini", exact: true })
          .getByText("Cleanup", { exact: true }),
      ).toBeVisible();
  });
}

test("preferences expose autostart without starting or downloading", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await openRuntimes(page);
  await preferences(page).click();
  await page
    .getByRole("switch", { name: "Start when Freehand launches" })
    .click();
  await expect(
    page.getByRole("switch", { name: "Start when Freehand launches" }),
  ).toBeChecked();
  expect(
    await page.evaluate(
      () => window.testRuntime.snapshot()[0].instance.autoStart,
    ),
  ).toBe(true);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "SetInstance:nemo-default",
  ]);
});

test("runtime mutations are disabled during active dictation", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?runtime&runtime-ready&runtime-provider=llama-cpp&work-busy",
  );
  await page.evaluate(() =>
    window.testRuntime.change("llama-cpp-default", { state: "stopped" }),
  );
  await openRuntimes(page);
  await preferences(page).click();
  await expect(
    page.getByRole("button", { name: "NVIDIA GPU (CUDA)", exact: true }),
  ).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("catalog download progress, cancellation and completion stay at the model", async ({
  page,
}) => {
  await page.setViewportSize({ width: 900, height: 640 });
  await page.goto("/tests/browser/app/?runtime");
  await openRuntimes(page);
  const id = await installRuntime(page);
  const model = page.getByRole("article", {
    name: "Parakeet TDT v3",
    exact: true,
  });
  await model.getByRole("button", { name: "Get", exact: true }).click();
  const progress = model.getByRole("progressbar");
  await expect(progress).toBeVisible();
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
  await model.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(model.getByRole("status")).toContainText("cancelled");
  await model.getByRole("button", { name: "Get", exact: true }).click();
  await page.evaluate((id) => window.testRuntime.finishDownload(id), id);
  await expect(model.getByRole("status")).toContainText(
    "downloaded and verified",
  );
});

for (const viewport of [
  { width: 1280, height: 720 },
  { width: 900, height: 640 },
]) {
  test(`explicit setup stages remain reachable at ${viewport.width}x${viewport.height}`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    await page.goto("/tests/browser/app/?runtime");
    await openRuntimes(page);
    await expect(
      page.getByRole("button", { name: "Install", exact: true }),
    ).toBeInViewport();
    const id = await installRuntime(page);
    const download = page.getByRole("button", {
      name: "Download selected model",
      exact: true,
    });
    await expect(download).toBeInViewport();
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toHaveCount(0);
    await download.click();
    const progress = page.getByRole("progressbar", {
      name: "Runtime operation progress",
      exact: true,
    });
    await expect(progress).toBeInViewport();
    await page
      .getByRole("button", { name: "Cancel operation", exact: true })
      .click();
    await expect(
      page.getByRole("status", { name: "Runtime operation", exact: true }),
    ).toContainText("Operation cancelled.");
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
      id,
    );
    await expect(progress).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Start", exact: true }),
    ).toHaveCount(0);
    await page.evaluate((id) => window.testRuntime.finishDownload(id), id);
    expect(await page.evaluate(() => window.testRuntime.calls)).not.toContain(
      `Start:${id}`,
    );
    const start = page.getByRole("button", { name: "Start", exact: true });
    await expect(start).toBeInViewport();
    await start.click();
    await expect(
      page.getByRole("button", { name: "Stop", exact: true }),
    ).toBeInViewport();
  });
}

test("browsing the inventory and catalog never downloads or starts models", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime");
  await openRuntimes(page);
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  await installRuntime(page);
  await page
    .getByRole("button", { name: "Refresh catalog", exact: true })
    .click();
  expect(
    await page.evaluate(() =>
      window.testRuntime.calls.some((call) =>
        /^(DownloadModel|Start):/.test(call),
      ),
    ),
  ).toBe(false);
});

test("leaving edited settings for runtime operations resolves the draft first", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await openSection(page, "audio");
  await page.locator("#max-duration").fill("90");
  await openRuntimes(page);
  await expect(page.getByRole("dialog")).toContainText("Save changes?");
  await page.getByRole("button", { name: "Keep editing", exact: true }).click();
  await expect(page.locator("#max-duration")).toHaveValue("90");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("unsupported hosts never offer an enabled runtime installation", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&platform=darwin");
  await openRuntimes(page);
  await expect(
    page.getByRole("button", { name: "Install", exact: true }),
  ).toBeDisabled();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
});

test("macOS llama installation and switching use Metal without CUDA", async ({
  page,
}) => {
  await page.goto(
    "/tests/browser/app/?runtime&runtime-provider=llama-cpp&runtime-macos",
  );
  await openRuntimes(page);
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await expect(
    page.getByText("Recommended: Apple GPU (Metal)", { exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "NVIDIA GPU (CUDA)", exact: true }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Download and install", exact: true })
    .click();
  await preferences(page).click();
  await expect(
    page.getByRole("button", { name: "Apple GPU (Metal)", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("button", { name: "CPU", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "CPU", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "GetBinaryOptions:llama-cpp",
    "SetInstance:llama-cpp",
    "InstallBackend:llama-cpp:metal",
    "InstallBackend:llama-cpp:cpu",
  ]);
});

test("all downloaded models require fresh confirmation before deletion", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await openRuntimes(page);
  const selected = page.getByRole("article", {
    name: "Nemotron 3.5 Streaming",
    exact: true,
  });
  const other = page.getByRole("article", {
    name: "Parakeet TDT v3",
    exact: true,
  });
  await expect(
    selected.getByRole("button", {
      name: "Delete Nemotron 3.5 Streaming",
      exact: true,
    }),
  ).toBeDisabled();
  await expect(
    other.getByRole("button", { name: "Select", exact: true }),
  ).toBeDisabled();
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await other
    .getByRole("button", { name: "Delete Parakeet TDT v3", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toContainText(
    "Deletes this downloaded model",
  );
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
  ]);
  await page.getByRole("button", { name: "Keep", exact: true }).click();
  await selected
    .getByRole("button", { name: "Delete Nemotron 3.5 Streaming", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm removal", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Download selected model", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
    "RemoveModel:nemo-default:nemotron-3.5",
  ]);
});

test("failed installations remain repairable and show their error immediately", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.evaluate(() =>
    window.testRuntime.change("nemo-default", {
      state: "error",
      backend: "",
      error: "Runtime installation failed.",
    }),
  );
  await openRuntimes(page);
  await expect(
    page.getByRole("alert").filter({ hasText: "Runtime installation failed." }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Install", exact: true }),
  ).toBeEnabled();
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toHaveCount(0);
});

test("restart does not start again when stopping failed", async ({ page }) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await openRuntimes(page);
  await page.evaluate(() => window.testRuntime.failNextStop());
  await page.getByRole("button", { name: "Restart", exact: true }).click();
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "The runtime operation did not finish." }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
  ]);
});

test("stopping the initial runtime keeps its selection when another provider is running", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.evaluate(() => window.testRuntime.addSecondProvider());
  await openRuntimes(page);
  await page
    .getByRole("button", { name: "Refresh inventory", exact: true })
    .click();
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Start", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "NeMo-Speech.cpp", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "Stop:nemo-default",
  ]);
  expect(
    await page.evaluate(
      () =>
        window.testRuntime
          .snapshot()
          .find((row) => row.instance.id === "other-speech")?.status.state,
    ),
  ).toBe("running");
});

test("duplicate runtime recovery preserves identity and resets pending removal", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?runtime&runtime-ready");
  await page.evaluate(() => window.testRuntime.addDuplicate("nemo-extra"));
  await openRuntimes(page);
  await expect(
    page.getByText(/Multiple saved installations need review/),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Duplicate speech · nemo-extra", exact: true })
    .click();
  await page.getByRole("button", { name: /^Manage runtime/ }).click();
  await page
    .getByRole("button", { name: "Delete runtime", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toContainText("Duplicate speech");
  await page.getByRole("button", { name: "Keep", exact: true }).click();
  await page
    .getByRole("button", { name: "Delete runtime", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm removal", exact: true })
    .click();
  await expect(
    page.getByText(/Multiple saved installations need review/),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Stop", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "DeleteInstance:nemo-extra",
  ]);
  expect(
    await page.evaluate(() =>
      window.testRuntime.snapshot().map((row) => row.instance.id),
    ),
  ).toEqual(["nemo-default"]);
});
