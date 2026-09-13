import { test, expect } from "@playwright/test";
// This surface has native fixed dimensions, including in the full browser suite.
test.use({ viewport: { width: 360, height: 500 } });
test("recording guidance promises dismissal, not external focus restoration", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=tray");
  await expect(page.getByText("Hides this panel before recording.", { exact: true })).toBeVisible();
});

test("compact task surface dismisses before start and stop", async ({ page }) => {
  await page.goto("/tests/browser/app/?view=tray");
  await page.getByRole("button", {name:"Start recording", exact:true}).click();
  await expect(page.getByRole("button", {name:"Stop recording", exact:true})).toBeEnabled();
  await page.getByRole("button", {name:"Stop recording", exact:true}).click();
  await expect.poll(() => page.evaluate(() => (window as any).trayCalls)).toEqual(["hide", "start", "hide", "stop"]);
  await expect(page.getByRole("button", {name:"Start recording", exact:true})).toBeDisabled();
});

test("bounded result, native copy, and task-scoped navigation", async ({page}, testInfo) => {
  await page.goto("/tests/browser/app/?view=tray&result");
  await page.getByRole("button", {name:"Copy", exact:true}).click();
  await expect(page.getByRole("button", {name:"Copied", exact:true})).toBeVisible();
  await page.getByRole("button", {name:"Settings", exact:true}).click();
  await page.getByRole("button", {name:"Open Freehand"}).click();
  await expect.poll(() => page.evaluate(() => (window as any).trayCalls)).toEqual(["copy", "settings:voice-transcription", "main"]);
  expect(await page.evaluate(() => ({width:document.documentElement.scrollWidth,height:document.documentElement.scrollHeight}))).toEqual({width:360,height:500});
  expect((await page.locator(".tray header").boundingBox())!.y).toBeGreaterThanOrEqual(12);
  await page.screenshot({path:testInfo.outputPath("tray-popover.png")});
});
test("loading and clipboard errors stay explicit", async ({page}) => {
  await page.goto("/tests/browser/app/?view=tray&loading");
  await expect(page.getByRole("button", {name:"Start recording", exact:true})).toBeDisabled();
  await expect(page.getByRole("status")).toContainText("Loading");
  await page.evaluate(() => (window as any).trayFixture.release());
  await expect(page.getByRole("button", {name:"Start recording", exact:true})).toBeEnabled();
  await page.goto("/tests/browser/app/?view=tray&result&copy-error");
  await page.getByRole("button", {name:"Copy", exact:true}).click();
  await expect(page.getByRole("alert")).toHaveText("Clipboard unavailable");
  await expect(page.getByRole("button", {name:"Copied", exact:true})).toHaveCount(0);
  await page.getByRole("button", {name:"Settings", exact:true}).click();
  expect((await page.locator(".tray header").boundingBox())!.y).toBeGreaterThanOrEqual(12);
});
test("failed hide prevents capture", async ({page}) => {
  await page.goto("/tests/browser/app/?view=tray&hide-error");
  await page.getByRole("button", {name:"Start recording", exact:true}).click();
  await expect(page.getByRole("alert")).toHaveText("Cannot dismiss panel");
  expect(await page.evaluate(() => (window as any).trayCalls)).toEqual(["hide"]);
});
test("realtime-capable backend retains its model picker in completed mode", async ({page}) => {
  await page.goto("/tests/browser/app/?view=tray");
  await expect(page.locator("#tray-model")).toBeVisible();
  await page.evaluate(() => {
    const editor = (window as any).trayFixture.session.editor;
    const settings = JSON.parse(JSON.stringify(editor.applied));
    settings.compatibilityProfiles.transcription = [{
      id: settings.voiceTranscription.compatibilityProfile,
      capabilities: {serverLoadedModel: true, realtime: true},
    }];
    settings.voiceTranscription.realtime = false;
    editor.applySettingsSnapshot(settings);
  });
  await expect(page.locator("#tray-model")).toBeVisible();
  await page.locator("#tray-model").fill("local/completed");
  await expect(page.getByRole("option", {name:/local\/completed/})).toBeVisible();
});

test("model choice commits without creating a settings draft", async ({page}) => {
  await page.goto("/tests/browser/app/?view=tray");
  await page.locator("#tray-model").fill("local/alternate");
  await page.getByRole("option", {name:/local\/alternate/}).click();
  await expect.poll(() => page.evaluate(() => (window as any).trayFixture.session.editor.applied.voiceTranscription.model)).toBe("local/alternate");
  expect(await page.evaluate(() => (window as any).trayFixture.session.editor.runtimeDirty)).toBe(false);
});
