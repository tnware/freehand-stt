import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

test("an installed runtime with a missing model resumes setup without repeating the recommendation hero", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&blank&runtime&runtime-use-flow");
  await page
    .getByRole("button", { name: "Set up local speech", exact: true })
    .click();
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Install NeMo-Speech.cpp?", exact: true })
    .getByRole("button", { name: "Download and install", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Download selected model", exact: true }),
  ).toBeEnabled();
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  const continuation = page.getByRole("region", {
    name: "Continue local speech setup",
    exact: true,
  });
  await expect(continuation).toBeVisible();
  await expect(continuation).toContainText("Download selected model");
  await expect(continuation).toContainText("Nemotron 3.5 Streaming");
  await expect(
    page.getByRole("region", { name: "Local speech setup", exact: true }),
  ).toHaveCount(0);
  expect(
    await page.evaluate(
      () => window.testConnectionWindows.settings().savedConnections.selected,
    ),
  ).toEqual({});
  await continuation
    .getByRole("button", { name: "Continue local setup", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "NeMo-Speech.cpp", exact: true }),
  ).toBeVisible();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "SetInstance:nemo-speech-cpp",
    "Install:nemo-speech-cpp",
  ]);
});

for (const { section, role, provider } of [
  { section: "processing", role: "cleanup", provider: "llama.cpp" },
  { section: "speech", role: "speech", provider: "NeMo-Speech.cpp" },
] as const) {
  test(`empty ${role} settings expose a role-qualified local setup route without enabling or selecting it`, async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.goto(
      "/tests/browser/app/?main&blank&runtime&runtime-all-providers&runtime-combined",
    );
    const before = await page.evaluate(() =>
      window.testConnectionWindows.settings(),
    );
    await openSection(page, section);
    const choice = page.getByRole("region", {
      name: `Local ${role} setup`,
      exact: true,
    });
    await expect(choice).toBeVisible();
    await choice
      .getByRole("button", { name: `Set up local ${role}`, exact: true })
      .click();
    await expect(
      page.getByRole("heading", { name: provider, exact: true }),
    ).toBeVisible();
    const after = await page.evaluate(() =>
      window.testConnectionWindows.settings(),
    );
    expect(after.savedConnections.selected).toEqual(
      before.savedConnections.selected,
    );
    expect(after.postProcessing.enabled).toBe(before.postProcessing.enabled);
    expect(after.textToSpeech.enabled).toBe(before.textToSpeech.enabled);
    expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([]);
  });
}

test("continuing setup keeps the exact non-first duplicate installation", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&blank&runtime&runtime-use-flow");
  await page
    .getByRole("button", { name: "Set up local speech", exact: true })
    .click();
  await page.getByRole("button", { name: "Install", exact: true }).click();
  await page
    .getByRole("dialog", { name: "Install NeMo-Speech.cpp?", exact: true })
    .getByRole("button", { name: "Download and install", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Download selected model", exact: true }),
  ).toBeEnabled();
  await page.evaluate(() => {
    window.testRuntime.addDuplicate("zz-recovery");
    window.testRuntime.change("nemo-speech-cpp", { state: "not_installed" });
  });
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "Voice transcription", exact: true })
    .click();
  const continuation = page.getByRole("region", {
    name: "Continue local speech setup",
    exact: true,
  });
  await expect(continuation).toContainText("Download selected model");
  await continuation
    .getByRole("button", { name: "Continue local setup", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Duplicate speech · zz-recovery",
      exact: true,
    }),
  ).toHaveAttribute("aria-pressed", "true");
  await page
    .getByRole("button", { name: "Refresh inventory", exact: true })
    .click();
  await expect(
    page.getByRole("button", {
      name: "Duplicate speech · zz-recovery",
      exact: true,
    }),
  ).toHaveAttribute("aria-pressed", "true");
  // The next explicit edit must still target the requested installation.
  await page
    .getByRole("article", { name: "Parakeet TDT v3", exact: true })
    .getByRole("button", { name: "Select", exact: true })
    .click();
  expect(await page.evaluate(() => window.testRuntime.calls)).toEqual([
    "SetInstance:nemo-speech-cpp",
    "Install:nemo-speech-cpp",
    "SetInstance:zz-recovery",
  ]);
  const rows = await page.evaluate(() => window.testRuntime.snapshot());
  expect(
    rows.find((row) => row.instance.id === "zz-recovery")?.instance.model,
  ).toBe("parakeet-tdt");
  expect(
    rows.find((row) => row.instance.id === "nemo-speech-cpp")?.instance.model,
  ).toBe("nemotron-3.5");
});
