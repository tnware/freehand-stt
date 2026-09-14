import { test, expect } from "@playwright/test";

// Load real components without a native bridge; only service metadata is mocked.
async function entry(
  page: import("@playwright/test").Page,
  script: string,
  ready = "#app > *",
) {
  await page.route("**/presentation-fixture", (route) =>
    route.fulfill({
      contentType: "text/html",
      body: `<div id="app"></div><script type="module">${script}</script>`,
    }),
  );
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/presentation-fixture");
  await expect(page.locator(ready).first()).toBeVisible();
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

test("shortcut accessibility exposes one spoken label without decorative glyphs", async ({
  page,
}) => {
  await entry(
    page,
    `
    import { mount } from '/tests/browser/app/presentation-runtime.ts';
    import ShortcutKeys from '/src/lib/components/common/ShortcutKeys.svelte';
    mount(ShortcutKeys, {target: document.getElementById('app'), props: {value: 'Cmd+Alt+Ctrl+Shift+Space', platform: 'darwin'}});
  `,
  );
  await expect(page.locator("#app")).toMatchAriaSnapshot(`
    - 'img "Keyboard shortcut: Command plus Option plus Control plus Shift plus Space"'
  `);
});

for (const [platform, credentialStore] of [
  ["windows", "Windows Credential Manager"],
  ["darwin", "macOS Keychain"],
]) {
  test(`configuration recovery ${platform} requires an explicit action and keeps credentials in place`, async ({
    page,
  }) => {
    await entry(
      page,
      `
      import { mount } from '/tests/browser/app/presentation-runtime.ts';
      import Recovery from '/src/lib/components/settings/ConfigurationRecoveryDialog.svelte';
      import { Session } from '/src/lib/stores/session.svelte.ts';
      import { settings, idle, serviceWithStatus } from '/src/lib/stores/session-fixtures-data.ts';
      const invalid = {...structuredClone(settings), platform: '${platform}', configuration: {recoveryRequired: true, errorKind: 'database_corrupt', message: 'Fixture database is unavailable.'}};
      window.recoveryCalls = [];
      const request = kind => new Promise((resolve, reject) => {
        window.recoveryCalls.push(kind);
        window.finishRecovery = success => success ? resolve({...invalid, configuration: {recoveryRequired: false}}) : reject(new Error('Fixture recovery failed'));
      });
      const session = new Session(serviceWithStatus(() => Promise.resolve(idle), {settings: {RetryConfiguration: () => request('retry'), ResetConfiguration: () => request('reset')}}));
      session.editor.applySettingsSnapshot(invalid);
      mount(Recovery, {target: document.getElementById('app'), props: {session}});
    `,
      '[role="dialog"]',
    );
    const recovery = page.getByRole("dialog", {
      name: "Saved settings need attention",
    });
    await expect(recovery).toContainText("Fixture database is unavailable.");
    await expect(recovery).toContainText("current-version database backup");
    await expect(recovery).toContainText(
      `credentials stored in ${credentialStore} are not deleted`,
    );
    await expect(recovery).toContainText(
      "Reconfigure connections and enter API keys again after resetting.",
    );
    expect(await page.evaluate(() => (window as any).recoveryCalls)).toEqual(
      [],
    );
    await page.keyboard.press("Escape");
    await page.mouse.click(5, page.viewportSize()!.height - 5);
    await expect(recovery).toBeVisible();
    await expect(
      recovery.getByRole("button", { name: "Close", exact: true }),
    ).toHaveCount(0);
    await recovery
      .getByRole("button", { name: "Retry loading", exact: true })
      .click();
    await expect(
      recovery.getByRole("button", { name: "Loading…", exact: true }),
    ).toBeDisabled();
    await expect(
      recovery.getByRole("button", { name: "Reset to defaults", exact: true }),
    ).toBeDisabled();
    await page.evaluate(() => (window as any).finishRecovery(false));
    await expect(recovery).toContainText("Fixture recovery failed");
    await recovery
      .getByRole("button", { name: "Reset to defaults", exact: true })
      .click();
    await expect(
      recovery.getByRole("button", { name: "Resetting…", exact: true }),
    ).toBeDisabled();
    await expect(
      recovery.getByRole("button", { name: "Retry loading", exact: true }),
    ).toBeDisabled();
    expect(await page.evaluate(() => (window as any).recoveryCalls)).toEqual([
      "retry",
      "reset",
    ]);
    await page.evaluate(() => (window as any).finishRecovery(true));
    await expect(recovery).toBeHidden();
  });
}

test("compatibility picker retains unsupported selections and offers only available profiles", async ({
  page,
}) => {
  await entry(
    page,
    `
    import { mount } from '/tests/browser/app/presentation-runtime.ts';
    import Picker from '/src/lib/components/settings/CompatibilityProfilePicker.svelte';
    mount(Picker, {target: document.getElementById('app'), props: {id: 'compatibility-test', value: 'localai', profiles: [
      {id: 'generic', label: 'Generic', available: true, description: 'Compatible API', capabilities: {}},
      {id: 'localai', label: 'LocalAI', available: false, description: 'Not implemented.', capabilities: {}},
      {id: 'future', label: 'Future backend', available: false, description: 'Planned only.', capabilities: {}}
    ]}});
  `,
  );
  const picker = page.getByRole("button", { name: "Backend", exact: true });
  await expect(picker).toContainText("LocalAI");
  await expect(
    page.getByText(/This saved profile is unavailable/),
  ).toBeVisible();
  await picker.click();
  await expect(page.getByRole("option")).toHaveCount(1);
  await expect(
    page.getByRole("option", { name: "Generic", exact: true }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(picker).toContainText("LocalAI");
  await picker.click();
  await page.getByRole("option", { name: "Generic", exact: true }).click();
  await expect(picker).toContainText("Generic");
  await expect(page.getByText(/This saved profile is unavailable/)).toHaveCount(
    0,
  );
});
