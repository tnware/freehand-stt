import { test, expect } from "./fixtures";

for (const width of [1280, 560]) {
  test(`disabled History has one privacy notice and explicit opt-in at ${width}px`, async ({
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 820 });
    await page.goto("/tests/browser/app/?main&setup-ready");
    await page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: "History", exact: true })
      .click();
    const history = page.getByRole("region", {
      name: "Transcript history",
      exact: true,
    });
    await expect(history).toHaveCount(1);
    await expect(
      history.getByText("History is turned off.", { exact: true }),
    ).toBeInViewport();
    await expect(
      page.getByText("History is turned off.", { exact: true }),
    ).toHaveCount(1);
    await expect(
      page.getByRole("complementary", { name: "History browser", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", {
        name: "Show transcript details",
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(page.locator("#workbench-secondary-sidebar")).toBeHidden();
    await expect(
      page.getByRole("button", { name: "Toggle bottom panel", exact: true }),
    ).toBeDisabled();
    await page.screenshot({
      path: info.outputPath(`history-off-${width}.png`),
    });

    await history
      .getByRole("button", { name: "History settings", exact: true })
      .click();
    const settings = page.locator('[data-pane="configuration"]');
    const retention = settings.getByRole("switch", {
      name: "Keep transcript history",
      exact: true,
    });
    await expect(retention).toHaveAttribute("aria-checked", "false");
    await expect(settings).toContainText("Retention is off.");
    await expect(settings).toContainText("History is never written to disk.");
    await expect(
      settings.getByText("The next finalized transcript will appear here.", {
        exact: true,
      }),
    ).toHaveCount(0);
    await expect(
      settings.getByRole("button", { name: "Clear history", exact: true }),
    ).toHaveCount(0);

    // A draft opt-in does not claim that the applied privacy setting changed.
    await retention.click();
    await expect(retention).toHaveAttribute("aria-checked", "true");
    await expect(settings).toContainText("Retention is off.");
    await expect(
      settings.getByText("The next finalized transcript will appear here.", {
        exact: true,
      }),
    ).toHaveCount(0);
    const close = page.getByRole("button", {
      name: "Close secondary sidebar",
      exact: true,
    });
    await close.click();
    const decision = page.getByRole("dialog", {
      name: "Save changes?",
      exact: true,
    });
    await decision
      .getByRole("button", { name: "Keep editing", exact: true })
      .click();
    await expect(retention).toHaveAttribute("aria-checked", "true");
    await close.click();
    await decision
      .getByRole("button", { name: "Discard", exact: true })
      .click();
    await expect(settings).toHaveCount(0);
    await expect(
      history.getByText("History is turned off.", { exact: true }),
    ).toBeInViewport();
    await history
      .getByRole("button", { name: "History settings", exact: true })
      .click();
    await expect(retention).toHaveAttribute("aria-checked", "false");
    await page.screenshot({
      path: info.outputPath(`history-settings-off-${width}.png`),
    });
  });
}

test("History promises future entries only after retention is successfully saved", async ({
  page,
  saves,
}) => {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto("/tests/browser/app/?main&setup-ready");
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "History", exact: true })
    .click();
  await page
    .getByRole("region", { name: "Transcript history", exact: true })
    .getByRole("button", { name: "History settings", exact: true })
    .click();
  const settings = page.locator('[data-pane="configuration"]');
  const retention = settings.getByRole("switch", {
    name: "Keep transcript history",
    exact: true,
  });
  const nextTranscript = settings.getByText(
    "The next finalized transcript will appear here.",
    { exact: true },
  );
  await retention.click();
  await settings.getByRole("button", { name: "Save", exact: true }).click();
  const pending = await saves.waitForStart();
  await expect(settings).toContainText("Retention is off.");
  await expect(nextTranscript).toHaveCount(0);
  await saves.complete(pending, "failure");
  await expect(settings).toContainText("Retention is off.");
  await expect(nextTranscript).toHaveCount(0);
  await settings.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(settings).toContainText("Retention is active.");
  await expect(nextTranscript).toBeVisible();
  await expect(
    page.getByRole("complementary", { name: "History browser", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("History is turned off.", { exact: true }),
  ).toHaveCount(0);

  await retention.click();
  await expect(settings).toContainText("Retention is active.");
  await settings.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(settings).toContainText("Retention is off.");
  await expect(nextTranscript).toHaveCount(0);
  await expect(
    page.getByRole("complementary", { name: "History browser", exact: true }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Close secondary sidebar", exact: true })
    .click();
  await expect(page.locator("#workbench-secondary-sidebar")).toBeHidden();
});
