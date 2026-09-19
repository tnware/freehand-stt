import { test, expect } from "./fixtures";
import { openSection } from "./context-navigation";

for (const theme of ["light", "dark"] as const) {
  test(`quick-save feedback reserves space and explains failures in ${theme}`, async ({
    page,
    saves,
  }, info) => {
    await page.setViewportSize({ width: 1280, height: 820 });
    await page.emulateMedia({ colorScheme: theme });
    await page.goto(
      `/tests/browser/app/?main&workflows&pickers&setup-ready&theme=${theme}`,
    );
    const sidebar = page.locator("#workbench-primary-sidebar");
    const model = sidebar.locator('input[id$="-voice-model"]');
    const cleanup = sidebar.getByRole("switch", {
      name: "Post-process transcripts",
      exact: true,
    });
    const top = (await cleanup.boundingBox())!.y;
    const feedback = sidebar.getByRole("status").first();
    const feedbackHeight = (await feedback.boundingBox())!.height;
    await model.fill("voice/updated");
    await page.getByRole("option", { name: /voice\/updated/ }).click();
    const first = await saves.waitForStart();
    await expect(
      sidebar.getByRole("status").filter({ hasText: "Saving…" }),
    ).toBeVisible();
    expect((await cleanup.boundingBox())!.y).toBeCloseTo(top, 0);
    await saves.complete(first, "success");
    await expect(
      sidebar.getByRole("status").filter({ hasText: /\bSaved\b/ }),
    ).toBeVisible();
    // The selected model can change which capability controls are shown.
    // Feedback itself keeps the same reserved line before and after saving.
    expect((await feedback.boundingBox())!.height).toBe(feedbackHeight);
    await page.screenshot({ path: info.outputPath(`saved-${theme}.png`) });
    await model.fill("voice/failed");
    await page.getByRole("option", { name: /voice\/failed/ }).click();
    await saves.complete(await saves.waitForStart(), "failure");
    const failure = sidebar.getByRole("status").filter({
      hasText: "Could not save. Your previous settings are still active.",
    });
    await expect(failure).toBeVisible();
    await expect(failure.locator("svg")).toHaveCount(1);
    await expect(model).toHaveValue("voice/updated");
    expect(
      await sidebar.evaluate((el) => el.scrollWidth <= el.clientWidth),
    ).toBe(true);
    await page.screenshot({ path: info.outputPath(`failed-${theme}.png`) });
  });
}

test("narrow settings rows retain label focus and an associated wrapping validation message", async ({
  page,
  saves,
}, info) => {
  await page.setViewportSize({ width: 520, height: 740 });
  await openSection(page, "audio");
  const input = page.locator("#max-duration");
  await page.locator('label[for="max-duration"]').click();
  await expect(input).toBeFocused();
  await input.fill("0");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "invalid-duration");
  await expect(input).toBeFocused();
  await expect(input).toHaveAttribute("aria-invalid", "true");
  const row = input.locator('xpath=ancestor::div[@role="group"][1]');
  const error = row.locator('p[id$="-error"]');
  await expect(error).toContainText(
    "Enter a recording limit from 1 to 262 seconds.",
  );
  await expect(row).toHaveAttribute(
    "aria-describedby",
    new RegExp(
      (await error.getAttribute("id"))!.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
    ),
  );
  await expect(error.locator('svg[aria-hidden="true"]')).toHaveCount(1);
  await input.hover();
  const errorColor = await error.evaluate((el) => getComputedStyle(el).color);
  await expect
    .poll(() => input.evaluate((el) => getComputedStyle(el).borderColor))
    .toBe(errorColor);
  await page.screenshot({ path: info.outputPath("field-error-narrow.png") });
  expect(await row.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(
    true,
  );
  await input.fill("60");
  await expect(error).toHaveCount(0);
  await expect(input).not.toHaveAttribute("aria-invalid", "true");
});
