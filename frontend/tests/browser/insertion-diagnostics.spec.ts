import { test, expect } from "./fixtures";
for (const scenario of ["capture", "dispatch", "unknown", "manual"]) {
  test(`copy-required ${scenario} preserves its reason without inventing focus loss`, async ({
    page,
  }) => {
    await page.goto(
      `/tests/browser/app/?view=workspace&copy-outcome=${scenario}`,
    );
    await expect(
      page
        .getByRole("region", { name: "Voice capture" })
        .getByText("Ready to copy", { exact: true }),
    ).toBeVisible();
    await expect(
      page.getByText("Focus moved before insertion.", { exact: true }),
    ).toHaveCount(0);
    await expect(page.getByText(/nothing was typed/i)).toHaveCount(0);
    const result = page.getByRole("region", {
      name: "Current result",
      exact: true,
    });
    await expect(
      result.getByRole("button", { name: "Copy", exact: true }),
    ).toBeEnabled();
    if (scenario === "manual")
      await expect(result).toContainText("Transcript ready to copy");
    if (scenario === "capture")
      await expect(result).toContainText("capture: value_not_settable");
    if (scenario === "dispatch")
      await expect(result).toContainText(
        "Check the target for any text already inserted",
      );
  });
}
