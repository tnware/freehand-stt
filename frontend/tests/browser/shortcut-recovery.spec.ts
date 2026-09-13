import { readFileSync } from "node:fs";
import { test, expect } from "./fixtures";

test("toggle can be cleared, saved unassigned, and explicitly replaced", async ({
  page,
  saves,
}) => {
  const bindings = readFileSync(
    new URL(
      "../../bindings/github.com/tnware/freehand-stt/internal/input/service.ts",
      import.meta.url,
    ),
    "utf8",
  );
  const captureID = Number(
    bindings.match(/function CaptureShortcut\([^]*?ByID\((\d+)/)![1],
  );
  let captures = 0;
  await page.route("**/wails/runtime", async (route) => {
    const request = route.request().postDataJSON();
    expect(request.args.methodID).toBe(captureID);
    expect(request.args.args[0]).toMatchObject({
      action: "toggle",
      assignments: { toggleRecording: "" },
    });
    captures++;
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        outcome: "captured",
        shortcut: "F14",
        changed: true,
      }),
    });
  });
  await page.goto("/tests/browser/app/?hold-degraded&general");
  await page.locator('[data-settings-section="shortcuts"]').click();
  const toggle = page.getByRole("group", {
    name: "Toggle recording",
    exact: true,
  });
  await expect(toggle.getByText("Optional", { exact: true })).toBeVisible();
  await toggle
    .getByRole("button", { name: "Clear Toggle recording shortcut" })
    .click();
  await expect(
    toggle.getByText("Not configured", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect
    .poll(() =>
      page.evaluate(
        () => window.testConnectionWindows.settings().toggleShortcut,
      ),
    )
    .toBe("");
  await expect(
    toggle.getByText("Not configured", { exact: true }),
  ).toBeVisible();
  expect(captures).toBe(0);

  const record = toggle.getByRole("button", { name: "Record", exact: true });
  await expect(record).toBeEnabled();
  await record.click();
  await expect(toggle.getByText("F14", { exact: true })).toBeVisible();
  expect(captures).toBe(1);
  expect(
    await page.evaluate(
      () => window.testConnectionWindows.settings().toggleShortcut,
    ),
  ).toBe("");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect
    .poll(() =>
      page.evaluate(
        () => window.testConnectionWindows.settings().toggleShortcut,
      ),
    )
    .toBe("F14");
});

test("unavailable native capture still permits Clear and explicit hold retry", async ({
  page,
  saves,
}) => {
  await page.goto("/tests/browser/app/?platform=darwin&hold-degraded");
  await page.locator('[data-settings-section="shortcuts"]').click();
  const hold = page.getByRole("group", { name: "Hold to talk", exact: true });
  await expect(
    hold.getByRole("button", { name: "Record", exact: true }),
  ).toBeDisabled();
  const clear = page.getByRole("button", {
    name: "Clear Hold to talk shortcut",
  });
  await expect(clear).toBeEnabled();
  const retry = page.getByRole("button", {
    name: "Retry hold-to-talk",
    exact: true,
  });
  await expect(retry).toBeEnabled();
  await expect(
    page.getByText("Secure Input is still active.", { exact: true }),
  ).toHaveCount(0);
  await retry.click();
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Secure Input is still active." }),
  ).toBeVisible();
  await retry.click();
  await expect(
    page.getByText("Hold-to-talk is ready.", { exact: true }),
  ).toBeVisible();
  // AX is still denied: permission-unavailable capture and clear are separate.
  await expect(
    hold.getByRole("button", { name: "Record", exact: true }),
  ).toBeDisabled();
  await expect(clear).toBeEnabled();
  await clear.click();
  await expect(hold.getByText("Not configured", { exact: true })).toBeVisible();
  await page
    .getByRole("button", { name: "Save and return", exact: true })
    .click();
  const save = await saves.waitForStart();
  await expect(retry).toBeDisabled();
  await saves.complete(save, "success");
});

// Exercise the production generated binding and Wails HTTP envelope, mocking
// only the native transport boundary. No keyboard or permission APIs run.
test("generated retry binding refreshes failure and preserves a dirty draft", async ({
  page,
}) => {
  const { readFileSync } = await import("node:fs");
  const bindings = readFileSync(
    new URL(
      "../../bindings/github.com/tnware/freehand-stt/internal/settings/service.ts",
      import.meta.url,
    ),
    "utf8",
  );
  const methodID = (name: string) =>
    Number(
      bindings.match(new RegExp(`function ${name}\\([^]*?ByID\\((\\d+)`))![1],
    );
  const retryID = methodID("RetryHoldShortcut");
  const snapshotID = methodID("GetSettings");
  const requests: number[] = [];
  let finish: (() => void) | undefined;
  let attempts = 0;
  await page.route("**/wails/runtime", async (route) => {
    const request = route.request().postDataJSON();
    expect(request.object).toBe(0);
    expect(request.method).toBe(0);
    expect(request.args.args).toEqual([]);
    requests.push(request.args.methodID);
    if (request.args.methodID === retryID) {
      attempts++;
      await new Promise<void>((resolve) => {
        finish = resolve;
      });
      if (attempts === 1) {
        await route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({
            kind: "RuntimeError",
            message: "Release all keys before retrying.",
          }),
        });
        return;
      }
    } else {
      expect(request.args.methodID).toBe(snapshotID);
    }
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        holdShortcut: "F13",
        holdAvailable: attempts > 1,
        holdAvailabilityReason:
          attempts > 1
            ? "Hold-to-talk is ready."
            : "The replacement tap is unavailable.",
      }),
    });
  });
  await page.goto(
    "/tests/browser/app/?platform=darwin&hold-degraded&hold-binding",
  );
  await page.locator('[data-settings-section="shortcuts"]').click();
  const retry = page.getByRole("button", {
    name: "Retry hold-to-talk",
    exact: true,
  });
  const hold = page.getByRole("group", { name: "Hold to talk", exact: true });
  await expect(retry).toBeEnabled();
  expect(requests).toEqual([]); // Mount never rearms or prompts.
  await page
    .getByRole("button", { name: "Clear Hold to talk shortcut" })
    .click();
  await retry.click();
  await expect(
    page.getByRole("button", { name: "Retrying hold-to-talk…", exact: true }),
  ).toBeDisabled();
  await expect.poll(() => requests).toEqual([retryID]);
  finish!();
  await expect(
    page
      .getByRole("alert")
      .filter({ hasText: "Release all keys before retrying." }),
  ).toBeVisible();
  await expect(
    page.getByText("The replacement tap is unavailable.", { exact: true }),
  ).toBeVisible();
  await expect(hold.getByText("Not configured", { exact: true })).toBeVisible();
  await retry.click();
  await expect.poll(() => attempts).toBe(2);
  finish!();
  await expect(
    page.getByText("Hold-to-talk is ready.", { exact: true }),
  ).toBeVisible();
  await expect(hold.getByText("Not configured", { exact: true })).toBeVisible();
  expect(requests).toEqual([retryID, snapshotID, retryID]);
});
