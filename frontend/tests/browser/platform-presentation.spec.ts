import { test, expect } from "@playwright/test";

// Load real components without a native bridge; only service metadata is mocked.
async function entry(page: import("@playwright/test").Page, script: string) {
  await page.route("**/presentation-fixture", (route) =>
    route.fulfill({
      contentType: "text/html",
      body: `<div id="app"></div><script type="module">${script}</script>`,
    }),
  );
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/presentation-fixture");
  await expect(page.locator("#app > *").first()).toBeVisible();
  expect(errors, "uncaught component errors").toEqual([]);
}

for (const platform of ["darwin", "windows"]) {
  test(`composer ${platform} shows the functional Control shortcut and activates once`, async ({
    page,
  }) => {
    await entry(
      page,
      `
      import { mount } from '/tests/browser/app/presentation-runtime.ts';
      import Composer from '/src/lib/components/home/TextToSpeech.svelte';
      import { session } from '/src/lib/stores/session.svelte.ts';
      import { settings } from '/src/lib/stores/session-fixtures-data.ts';
      session.editor.applySettingsSnapshot({...settings, platform: '${platform}'});
      window.speechCalls = 0;
      mount(Composer, {target: document.getElementById('app'), props: {
        text: 'A draft', settings: {...settings.textToSpeech, enabled: true, baseURL: 'https://tts.test/v1', model: 'speech', voice: 'voice'},
        status: {phase: 'idle'}, onSpeak: () => window.speechCalls++,
        onPause() {}, onResume() {}, onRestart() {}, onStop() {}, onSave() {}, onClear() {}, onOpenSettings() {}
      }});
    `,
    );
    const text = page.getByRole("textbox", { name: "Text to speak" });
    await expect(text).toHaveAttribute("aria-keyshortcuts", "Control+Enter");
    await expect(page.locator("#speech-compose-shortcut")).toContainText(
      platform === "darwin"
        ? "Press Control plus Enter"
        : "Press Ctrl plus Enter",
    );
    await expect(page.locator("button kbd")).toHaveText(
      platform === "darwin" ? "⌃+Enter" : "Ctrl+Enter",
    );
    for (const modifiers of [
      { metaKey: true },
      { ctrlKey: true, metaKey: true },
      { ctrlKey: true, shiftKey: true },
      { ctrlKey: true, altKey: true },
      { ctrlKey: true, repeat: true },
      { ctrlKey: true, isComposing: true },
    ]) {
      await text.dispatchEvent("keydown", {
        key: "Enter",
        bubbles: true,
        ...modifiers,
      });
    }
    expect(await page.evaluate(() => (window as any).speechCalls)).toBe(0);
    await text.press("Control+Enter");
    expect(await page.evaluate(() => (window as any).speechCalls)).toBe(1);
    await text.press("Enter");
    expect(await page.evaluate(() => (window as any).speechCalls)).toBe(1);
  });
}

for (const state of ["loading", "failure", "darwin", "windows"]) {
  test(`About ${state} uses authoritative or neutral credential copy`, async ({
    page,
  }) => {
    for (const service of ["settings", "buildinfo", "updates"]) {
      await page.route(
        `**/bindings/**/internal/${service}/service.*`,
        (route) => {
          const method = service === "settings" ? "GetSettings" : "Current";
          const value =
            service === "updates"
              ? "{}"
              : service === "settings"
                ? `{platform: '${state}', appearance: 'system'}`
                : `{platform: '${state}', windowsVersion: '11', wailsVersion: 'test', goVersion: 'test'}`;
          const response =
            state === "loading"
              ? "new Promise(() => {})"
              : state === "failure"
                ? 'Promise.reject(new Error("Metadata unavailable"))'
                : `Promise.resolve(${value})`;
          return route.fulfill({
            contentType: "application/javascript",
            body: `export function ${method}() { return ${response}; } export function CheckForUpdates() { return Promise.resolve(); }`,
          });
        },
      );
    }
    await entry(
      page,
      `import { mount } from '/tests/browser/app/presentation-runtime.ts'; import About from '/src/AboutWindow.svelte'; mount(About, {target: document.getElementById('app')});`,
    );
    const about = page.getByRole("main", { name: "About Freehand" });
    await expect(about).toBeVisible();
    if (state === "failure")
      await expect(page.getByRole("alert")).toContainText(
        "Metadata unavailable",
      );
    const store =
      state === "darwin"
        ? "macOS Keychain"
        : state === "windows"
          ? "Windows Credential Manager"
          : "your operating system’s credential store";
    await expect(about).toContainText(store);
    if (state !== "windows")
      await expect(about).not.toContainText("Windows Credential Manager");
    if (state === "windows") await expect(about).toContainText("Windows 11");
    if (state === "darwin") await expect(about).not.toContainText("Windows 11");
  });
}
