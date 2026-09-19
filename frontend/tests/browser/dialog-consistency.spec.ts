import { test, expect } from "./fixtures";

declare global {
  interface Window {
    testDialogs: { calls: string[]; complete: (success: boolean) => void };
  }
}

for (const theme of ["light", "dark"]) {
  test(`dialog actions stay reachable with long content in ${theme}`, async ({
    page,
  }, testInfo) => {
    await page.setViewportSize({ width: 380, height: 460 });
    await page.goto(`/tests/browser/app/?view=dialogs&long&theme=${theme}`);
    await page
      .getByRole("button", { name: "Open recovery", exact: true })
      .click();
    const dialog = page.getByRole("dialog", {
      name: "Saved settings need attention",
      exact: true,
    });
    const retry = dialog.getByRole("button", {
      name: "Retry loading",
      exact: true,
    });
    await expect(retry).toBeFocused();
    await expect(retry).toBeInViewport();
    await expect(
      dialog.getByRole("button", { name: "Reset to defaults", exact: true }),
    ).toBeInViewport();
    expect(
      await dialog.evaluate((el) => el.scrollWidth <= el.clientWidth),
    ).toBe(true);
    await page.keyboard.press("Escape");
    await page.mouse.click(2, 2);
    await expect(dialog).toBeVisible();
    await page.screenshot({
      path: testInfo.outputPath(`recovery-${theme}.png`),
    });
    await retry.click();
    await expect(
      dialog.getByRole("button", { name: "Loading…", exact: true }),
    ).toBeDisabled();
    await expect(
      dialog.getByRole("button", { name: "Reset to defaults", exact: true }),
    ).toBeDisabled();
    expect(await page.evaluate(() => window.testDialogs.calls)).toEqual([
      "retry",
    ]);
    await page.evaluate(() => window.testDialogs.complete(false));
    await expect(
      dialog.getByRole("alert").filter({ hasText: "Fixture recovery failed" }),
    ).toBeVisible();
    await dialog
      .getByRole("button", { name: "Reset to defaults", exact: true })
      .click();
    await expect(
      dialog.getByRole("button", { name: "Resetting…", exact: true }),
    ).toBeDisabled();
    await page.evaluate(() => window.testDialogs.complete(true));
    await expect(dialog).toBeHidden();
    expect(await page.evaluate(() => window.testDialogs.calls)).toEqual([
      "retry",
      "reset",
    ]);
  });
}

test("unsaved changes preserve focus, pending work, errors, and retry", async ({
  page,
}, testInfo) => {
  await page.goto("/tests/browser/app/?view=dialogs&theme=dark");
  const trigger = page.getByRole("button", {
    name: "Open unsaved changes",
    exact: true,
  });
  await trigger.click();
  const dialog = page.getByRole("dialog", {
    name: "Save settings changes?",
    exact: true,
  });
  await expect(
    dialog.getByRole("button", { name: "Keep editing" }),
  ).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(trigger).toBeFocused();
  await trigger.click();
  await page.screenshot({ path: testInfo.outputPath("unsaved-dark.png") });
  await dialog.getByRole("button", { name: "Save", exact: true }).click();
  await expect(
    dialog.getByRole("button", { name: "Saving…", exact: true }),
  ).toBeDisabled();
  await expect(
    dialog.getByRole("button", { name: "Close", exact: true }),
  ).toHaveCount(0);
  await page.keyboard.press("Escape");
  await page.mouse.click(2, 2);
  await expect(dialog).toBeVisible();
  expect(await page.evaluate(() => window.testDialogs.calls)).toEqual(["save"]);
  await page.evaluate(() => window.testDialogs.complete(false));
  await expect(dialog.getByRole("alert")).toContainText("Fixture save failed");
  await dialog.getByRole("button", { name: "Save", exact: true }).click();
  await page.evaluate(() => window.testDialogs.complete(true));
  await expect(dialog).toBeHidden();
  await expect(trigger).toBeFocused();
});

for (const kind of ["model", "files", "instance"] as const) {
  test(`${kind} removal stays open through failure and requires an explicit retry`, async ({
    page,
  }, testInfo) => {
    await page.goto(
      "/tests/browser/app/?main&runtime&runtime-ready&theme=dark",
    );
    await expect(
      page.getByRole("navigation", { name: "Workspace", exact: true }),
    ).toBeVisible();
    await page.evaluate(() => {
      window.testRuntime.change("nemo-default", { state: "stopped" });
      window.testRuntime.holdRemoval();
    });
    await page
      .getByRole("navigation", { name: "Workspace", exact: true })
      .getByRole("button", { name: "Local runtime", exact: true })
      .click();
    if (kind === "model") {
      await page
        .getByRole("button", { name: "Delete Parakeet TDT v3", exact: true })
        .click();
    } else {
      await page.getByRole("button", { name: /^Manage runtime/ }).click();
      await page
        .getByRole("button", {
          name: kind === "files" ? "Remove downloaded files" : "Delete runtime",
          exact: true,
        })
        .click();
    }
    const dialog = page.getByRole("dialog");
    const label =
      kind === "model"
        ? "Delete model"
        : kind === "files"
          ? "Remove runtime files"
          : "Delete runtime";
    await expect(
      dialog.getByRole("button", { name: "Cancel", exact: true }),
    ).toBeFocused();
    await page.screenshot({
      path: testInfo.outputPath(`remove-${kind}-dark.png`),
    });
    await dialog.getByRole("button", { name: label, exact: true }).click();
    await expect(
      dialog.getByRole("button", { name: "Removing…", exact: true }),
    ).toBeDisabled();
    await expect(
      dialog.getByRole("button", { name: "Cancel", exact: true }),
    ).toBeDisabled();
    await page.keyboard.press("Escape");
    await page.mouse.click(2, 2);
    await expect(dialog).toBeVisible();
    expect(await page.evaluate(() => window.testRuntime.calls.length)).toBe(1);
    if (kind !== "instance") {
      await page.evaluate(() => window.testRuntime.acknowledgeRemoval());
      await expect(
        dialog.getByRole("button", { name: "Removing…", exact: true }),
      ).toBeDisabled();
      await page.keyboard.press("Escape");
      await expect(dialog).toBeVisible();
    }
    await page.evaluate(() => window.testRuntime.completeRemoval(false));
    await expect(dialog.getByRole("alert")).toBeVisible();
    await expect(
      dialog.getByRole("button", { name: label, exact: true }),
    ).toBeEnabled();
    await dialog.getByRole("button", { name: label, exact: true }).click();
    if (kind !== "instance") {
      await page.evaluate(() => window.testRuntime.acknowledgeRemoval());
      await expect(
        dialog.getByRole("button", { name: "Removing…", exact: true }),
      ).toBeDisabled();
    }
    await page.evaluate(() => window.testRuntime.completeRemoval(true));
    await expect(dialog).toBeHidden();
    expect(await page.evaluate(() => window.testRuntime.calls.length)).toBe(2);
  });
}

test("connection deletion prevents dismissal while saving and keeps failure recoverable", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?main&unused-connection");
  await page
    .getByRole("navigation", { name: "Workspace", exact: true })
    .getByRole("button", { name: "Connections", exact: true })
    .click();
  await page.getByRole("button", { name: /Unused server/ }).click();
  await page
    .getByRole("button", { name: "Connection actions", exact: true })
    .click();
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  const dialog = page.getByRole("dialog", {
    name: "Delete connection?",
    exact: true,
  });
  await expect(
    dialog.getByRole("button", { name: "Cancel", exact: true }),
  ).toBeFocused();
  await dialog
    .getByRole("button", { name: "Delete connection", exact: true })
    .click();
  const first = await saves.waitForStart();
  await expect(
    dialog.getByRole("button", { name: "Deleting…", exact: true }),
  ).toBeDisabled();
  await expect(
    dialog.getByRole("button", { name: "Close", exact: true }),
  ).toHaveCount(0);
  await page.keyboard.press("Escape");
  await page.mouse.click(2, 2);
  await expect(dialog).toBeVisible();
  await saves.complete(first, "failure");
  await expect(dialog.getByRole("alert")).toContainText("Fixture save failed");
  await dialog
    .getByRole("button", { name: "Delete connection", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(dialog).toBeHidden();
  await expect(page.getByRole("button", { name: /Unused server/ })).toHaveCount(
    0,
  );
});
