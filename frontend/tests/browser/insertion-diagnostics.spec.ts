import { test, expect } from "./fixtures";

for (const scenario of ["capture", "dispatch", "unknown", "manual"]) {
  test(`copy-required ${scenario} preserves the reason without inventing focus loss`, async ({
    page,
  }) => {
    await page.goto(`/tests/browser/app/?view=workspace&copy-outcome=${scenario}`);
    await expect(
      page.locator(".transport .stage").getByText("Ready to copy", { exact: true }),
    ).toBeVisible();
    await expect(page.getByText("Focus moved before insertion.", { exact: true })).toHaveCount(0);
    await expect(page.getByText(/nothing was typed/i)).toHaveCount(0);
    if (scenario === "manual") {
      await expect(
        page.getByText("Manual copy is selected for this profile.", {
          exact: true,
        }),
      ).toBeVisible();
    } else {
      await expect(
        page.getByText("Automatic insertion needs attention.", { exact: true }),
      ).toBeVisible();
    }
    if (scenario === "capture") {
      await expect(page.getByRole("region", { name: "Current result", exact: true })).toContainText(
        "capture: value_not_settable",
      );
    }
    if (scenario === "dispatch") {
      await expect(page.getByRole("region", { name: "Current result", exact: true })).toContainText(
        "Check the target for any text already inserted",
      );
    }
  });
}
