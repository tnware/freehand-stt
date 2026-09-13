import type { Page } from "@playwright/test";
import {
  test,
  expect,
  manager,
  requestNativeClose,
  currentManagerState,
} from "./connection-window-fixtures";

const setup = (page: Page) =>
  manager(page).getByRole("region", { name: "Connection editor", exact: true });
const decision = (page: Page) =>
  page.getByRole("dialog", {
    name: "Save settings before changing connections?",
    exact: true,
  });
const selection = (page: Page) => page.locator("#saved-connection-stt");
async function addConnection(page: Page) {
  await page
    .getByRole("button", { name: "Show connections", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Add connection…", exact: true })
    .click();
}

test("native close-request browser proxy retains the same document across discard and reopen", async ({
  page,
}) => {
  await addConnection(page);
  const frame = page.locator('iframe[title="Connections window"]');
  const document = await manager(page)
    .locator("body")
    .evaluateHandle(() => window.document);
  await manager(page).locator("#connection-name").fill("Discard this draft");
  await requestNativeClose(page);
  const prompt = manager(page).getByRole("dialog", {
    name: "Save connection changes?",
    exact: true,
  });
  await expect(prompt).toBeVisible();
  await expect(frame).toBeVisible();
  await prompt
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(manager(page).locator("#connection-name")).toHaveValue(
    "Discard this draft",
  );
  await requestNativeClose(page);
  await prompt.getByRole("button", { name: "Discard", exact: true }).click();
  await expect(frame).toBeHidden();
  await expect(
    frame,
    "hide must retain the WebView, not destroy it",
  ).toHaveCount(1);
  expect(await currentManagerState(page)).toEqual({
    visible: false,
    request: { id: "", purpose: "", create: false },
  });
  await addConnection(page);
  expect(await currentManagerState(page)).toEqual({
    visible: true,
    // Purpose.Transcription is serialized as "stt" by the generated binding.
    request: { id: "", purpose: "stt", create: true },
  });
  expect(
    await document.evaluate((original) => original === window.document),
  ).toBe(true);
  await expect(manager(page).locator("#connection-name")).toHaveValue("");
  await expect(prompt).toBeHidden();
  await expect(selection(page)).toHaveValue("Original server");
  // Clean native close must not reuse the stale discard action or dirty state.
  await requestNativeClose(page);
  await expect(frame).toBeHidden();
  await expect(frame).toHaveCount(1);
});

test("WindowHide clears a pending discard in the retained manager before a different open request", async ({
  page,
}) => {
  await addConnection(page);
  const document = await manager(page)
    .locator("body")
    .evaluateHandle(() => window.document);
  await manager(page).locator("#connection-name").fill("Stale new connection");
  await requestNativeClose(page);
  await expect(
    manager(page).getByRole("dialog", { name: "Save connection changes?" }),
  ).toBeVisible();
  // Unlike close(), an external hide does not call clear() itself: the actual
  // common:WindowHide subscription must clear the dialog and editor state.
  await page.evaluate(() => window.testConnectionWindows.hide());
  await expect(page.locator('iframe[title="Connections window"]')).toBeHidden();
  expect(
    await page.evaluate(() => window.testConnectionWindows.requestState),
  ).toEqual({
    visible: false,
    request: { id: "", purpose: "", create: false },
  });
  await page.locator('[data-settings-section="connections"]').click();
  await expect(
    manager(page).getByRole("dialog", { name: "Save connection changes?" }),
  ).toBeHidden();
  await expect(setup(page)).toBeHidden();
  await manager(page)
    .getByRole("button", { name: /Original server/ })
    .click();
  expect(
    await document.evaluate((original) => original === window.document),
  ).toBe(true);
  await expect(manager(page).locator("#connection-name")).toHaveValue(
    "Original server",
  );
  await expect(
    manager(page).getByRole("button", { name: "Save connection", exact: true }),
  ).toBeVisible();
  await requestNativeClose(page);
  await expect(page.locator('iframe[title="Connections window"]')).toBeHidden();
});

test("revealing a visible manager preserves its request and unsaved draft", async ({
  page,
}) => {
  await addConnection(page);
  await manager(page).locator("#connection-name").fill("Owned draft");
  const before = await page.evaluate(
    () => window.testConnectionWindows.requestState,
  );
  await page.evaluate(() =>
    window.testConnectionWindows.open({
      ...window.testConnectionWindows.requestState.request,
      id: "original",
      create: false,
    }),
  );
  expect(
    await page.evaluate(() => window.testConnectionWindows.requestState),
  ).toEqual(before);
  await expect(page.locator('iframe[title="Connections window"]')).toHaveCount(
    1,
  );
  await expect(manager(page).locator("#connection-name")).toHaveValue(
    "Owned draft",
  );
});

test("native close-request browser proxy leaves a pending save available for failure and retry", async ({
  page,
  saves,
}) => {
  await addConnection(page);
  await manager(page)
    .locator("#connection-name")
    .fill("Native close pending server");
  await manager(page)
    .locator("#connection-url")
    .fill("https://fixture.example.test/v1");
  await setup(page)
    .getByRole("button", { name: "Save and set up", exact: true })
    .click();
  const first = await saves.waitForStart();
  await requestNativeClose(page);
  await expect(setup(page)).toBeVisible();
  await expect(
    manager(page).getByRole("dialog", { name: "Save connection changes?" }),
  ).toBeHidden();
  await saves.complete(first, "failure");
  await expect(setup(page).getByRole("alert")).toContainText(
    "Fixture save failed",
  );
  await requestNativeClose(page);
  await manager(page)
    .getByRole("button", { name: "Keep editing", exact: true })
    .click();
  await expect(manager(page).locator("#connection-name")).toHaveValue(
    "Native close pending server",
  );
  await setup(page)
    .getByRole("button", { name: "Save and set up", exact: true })
    .click();
  await saves.complete(await saves.waitForStart(), "success");
  await expect(setup(page)).toBeHidden();
  await expect(selection(page)).toHaveValue("Native close pending server");
});

for (const gesture of ["Escape", "Close / outside click"] as const) {
  test.describe(gesture, () => {
    async function dismiss(page: Page) {
      if (
        await page.locator('iframe[title="Connections window"]').isVisible()
      ) {
        // Native windows have no outside-click dismissal: use their Close control.
        if (gesture === "Escape")
          // Focus a focusable element in this WebView, not the outer picker/body.
          await manager(page).locator("#connection-name").press("Escape");
        else {
          const close = manager(page).getByRole("button", {
            name: "Close",
            exact: true,
          });
          // A real pointer gesture also exercises the disabled control while saving.
          const box = await close.boundingBox();
          expect(box).not.toBeNull();
          await page.mouse.click(
            box!.x + box!.width / 2,
            box!.y + box!.height / 2,
            { delay: 50 },
          );
        }
      } else if (gesture === "Escape") await page.keyboard.press("Escape");
      else await page.mouse.click(5, 5, { delay: 50 });
    }

    test("Keep editing preserves the visible connection draft", async ({
      page,
    }) => {
      await addConnection(page);
      await manager(page).locator("#connection-name").fill("Unfinished server");
      await dismiss(page);
      await manager(page)
        .getByRole("button", { name: "Keep editing", exact: true })
        .click();
      await expect(setup(page)).toBeVisible();
      await expect(manager(page).locator("#connection-name")).toHaveValue(
        "Unfinished server",
      );
      await manager(page).locator("#connection-name").fill("Still editable");
      await expect(manager(page).locator("#connection-name")).toHaveValue(
        "Still editable",
      );
      await expect(selection(page)).toHaveValue("Original server");
    });

    test("Discard clears the connection draft and preserves the selection", async ({
      page,
    }) => {
      await addConnection(page);
      await manager(page).locator("#connection-name").fill("Unfinished server");
      await dismiss(page);
      await manager(page)
        .getByRole("button", { name: "Discard", exact: true })
        .click();
      await expect(setup(page)).toBeHidden();
      await expect(selection(page)).toHaveValue("Original server");
      await addConnection(page);
      await expect(manager(page).locator("#connection-name")).toHaveValue("");
    });

    test("An unchanged connection editor dismisses normally", async ({
      page,
    }) => {
      await addConnection(page);
      await expect(setup(page)).toBeVisible();
      await dismiss(page);
      await expect(setup(page)).toBeHidden();
      await expect(selection(page)).toHaveValue("Original server");
    });

    test("A pending connection save retains failure and retry actions", async ({
      page,
      saves,
    }) => {
      await addConnection(page);
      await manager(page).locator("#connection-name").fill("Pending server");
      await manager(page)
        .locator("#connection-url")
        .fill("https://fixture.example.test/v1");
      await setup(page)
        .getByRole("button", { name: "Save and set up", exact: true })
        .click();
      const first = await saves.waitForStart();
      await expect(
        setup(page).getByRole("button", { name: "Cancel", exact: true }),
      ).toBeDisabled();
      await dismiss(page);
      await expect(setup(page)).toBeVisible();
      await saves.complete(first, "failure");
      await expect(setup(page).getByRole("alert")).toContainText(
        "Fixture save failed",
      );
      await expect(manager(page).locator("#connection-name")).toHaveValue(
        "Pending server",
      );
      await expect(selection(page)).toHaveValue("Original server");
      await setup(page)
        .getByRole("button", { name: "Save and set up", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(setup(page)).toBeHidden();
      await expect(selection(page)).toHaveValue("Pending server");
    });

    test("A pending settings save retains the prompt through failure and retry", async ({
      page,
      saves,
    }) => {
      await page.locator("summary", { hasText: "Request settings" }).click();
      await page.locator("#file-transcription-timeout").fill("75");
      await expect(
        page.getByText("Unsaved changes", { exact: true }),
      ).toBeVisible();
      await addConnection(page);
      await decision(page)
        .getByRole("button", { name: "Save and continue", exact: true })
        .click();
      const first = await saves.waitForStart();
      await expect(
        decision(page).getByRole("button", {
          name: "Keep editing",
          exact: true,
        }),
      ).toBeDisabled();
      await dismiss(page);
      await expect(decision(page)).toBeVisible();
      await saves.complete(first, "failure");
      await expect(decision(page).getByRole("alert")).toContainText(
        "Fixture save failed",
      );
      await expect(page.locator("#file-transcription-timeout")).toHaveValue(
        "75",
      );
      await expect(selection(page)).toHaveValue("Original server");
      await decision(page)
        .getByRole("button", { name: "Save and continue", exact: true })
        .click();
      await saves.complete(await saves.waitForStart(), "success");
      await expect(setup(page)).toBeVisible();
      await manager(page)
        .getByRole("button", { name: "Close", exact: true })
        .click();
      await expect(setup(page)).toBeHidden();
      await expect(page.locator("#file-transcription-timeout")).toHaveValue(
        "75",
      );
      await expect(
        page.getByText("All changes saved", { exact: true }),
      ).toBeVisible();
    });

    test("An idle save decision dismisses without losing settings edits", async ({
      page,
    }) => {
      await page.locator("summary", { hasText: "Request settings" }).click();
      await page.locator("#file-transcription-timeout").fill("75");
      await addConnection(page);
      await expect(decision(page)).toBeVisible();
      await dismiss(page);
      await expect(decision(page)).toBeHidden();
      await expect(page.locator("#file-transcription-timeout")).toHaveValue(
        "75",
      );
      await expect(
        page.getByText("Unsaved changes", { exact: true }),
      ).toBeVisible();
      await expect(selection(page)).toHaveValue("Original server");
    });
  });
}
