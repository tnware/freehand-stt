import { test, expect } from "./fixtures";
async function selectSection(
  page: import("@playwright/test").Page,
  id: string,
) {
  const toggle = page.getByRole("button", {
    name: "Toggle primary sidebar",
    exact: true,
  });
  await expect(toggle).toBeVisible();
  if ((await toggle.getAttribute("aria-pressed")) === "false")
    await toggle.click();
  await page.locator(`[data-settings-section="${id}"]`).click();
}
const requestSettings = (page: import("@playwright/test").Page) =>
  page
    .locator("details")
    .filter({ has: page.locator("summary", { hasText: "Request settings" }) });

for (const width of [860, 520]) {
  test(`workflow request options preserve edits at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 740 });
    await page.goto("/tests/browser/app/?workflows&theme=dark");
    for (const [id, control, value] of [
      ["server", "file-transcription-timeout", "45"],
      ["voice-transcription", "voice-timeout", "150"],
      ["processing", "cleanup-timeout", "160"],
      ["speech", "tts-timeout", "170"],
    ]) {
      await selectSection(page, id);
      const request = requestSettings(page);
      await expect(page.locator(`#${control}`)).toBeHidden();
      if (id === "speech")
        await expect(
          page.getByRole("button", { name: "Preview", exact: true }),
        ).toBeVisible();
      await page.screenshot({ path: info.outputPath(`${id}-${width}.png`) });
      await request.locator("summary").click();
      await page.locator(`#${control}`).fill(value);
      await request.locator("summary").click();
      await selectSection(page, "general");
      await selectSection(page, id);
      await request.locator("summary").click();
      await expect(page.locator(`#${control}`)).toHaveValue(value);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    }
  });
}

test("save validation reveals a collapsed request limit", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?workflows");
  await requestSettings(page).locator("summary").click();
  await page.locator("#file-transcription-timeout").fill("45");
  await requestSettings(page).locator("summary").click();
  await selectSection(page, "general");
  await page
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "invalid-file-timeout");
  await expect(page.locator("#file-transcription-timeout")).toBeFocused();
  await expect(page.locator("#file-transcription-timeout")).toHaveValue("45");
});

test("voice controls survive collapsing and workflow validation reveals them", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?workflows");
  await selectSection(page, "voice-transcription");
  await expect(page.locator("#voice-prompt")).toBeVisible();
  await requestSettings(page).locator("summary").click();
  await page.locator("#voice-temperature-override").click();
  await page
    .getByRole("spinbutton", { name: "Temperature", exact: true })
    .fill("0.4");
  await requestSettings(page).locator("summary").click();
  await selectSection(page, "general");
  await page
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "invalid-voice");
  await expect(
    page.getByRole("spinbutton", { name: "Temperature", exact: true }),
  ).toHaveValue("0.4");
  await expect(page.locator("#voice-timeout")).toBeVisible();
});

test("connection attention stays visible with diagnostics collapsed", async ({
  page,
}) => {
  await page.goto("/tests/browser/app/?workflows&attention");
  await page.getByRole("button", { name: "Refresh models" }).click();
  await expect(requestSettings(page).locator("summary")).toContainText(
    "Connection needs attention",
  );
  await expect(
    page.getByText("The selected model was not listed."),
  ).toBeHidden();
  await requestSettings(page).locator("summary").click();
  await expect(
    page.getByText("The selected model was not listed."),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Check again" })).toBeVisible();
});
