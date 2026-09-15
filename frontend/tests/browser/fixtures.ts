import { test as base, expect } from "@playwright/test";
import type { SaveControl } from "./app/save-control";

type Saves = {
  waitForStart: () => Promise<number>;
  complete: (...args: Parameters<SaveControl["complete"]>) => Promise<void>;
};

type WindowMethod =
  | "IsMaximised"
  | "IsMinimised"
  | "Minimise"
  | "ToggleMaximise"
  | "Close"
  | "Hide";

declare global {
  interface Window {
    testWindow: {
      calls: WindowMethod[];
      maximised: boolean;
      minimised: boolean;
      shown: boolean;
      pendingStateReads: number;
      pendingVisibilityReads: number;
      failNext: (method: WindowMethod) => void;
      deferNextStateRead: () => void;
      releaseStateRead: () => void;
      deferNextVisibilityRead: () => void;
      releaseVisibilityRead: () => void;
      readVisibility: () => Promise<boolean>;
      emit: (name: string, maximised?: boolean) => void;
      invoke: (method: number) => Promise<{ status: number; body: string }>;
    };
  }
}

export const test = base.extend<{ saves: Saves }>({
  page: async ({ page, baseURL }, use) => {
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.addInitScript(() => {
      const methods: Record<number, WindowMethod> = {
        2: "Close",
        11: "Hide",
        14: "IsMaximised",
        15: "IsMinimised",
        17: "Minimise",
        39: "ToggleMaximise",
      };
      const failures = new Set<WindowMethod>();
      let deferRead = false;
      let releaseRead: (() => void) | undefined;
      let deferVisibility = false;
      let releaseVisibility: (() => void) | undefined;
      const state: Window["testWindow"] = {
        calls: [],
        maximised: new URLSearchParams(location.search).has("window-maximised"),
        minimised: false,
        shown: true,
        pendingStateReads: 0,
        pendingVisibilityReads: 0,
        failNext: (method) => void failures.add(method),
        deferNextStateRead: () => {
          deferRead = true;
        },
        releaseStateRead: () => releaseRead?.(),
        deferNextVisibilityRead: () => {
          deferVisibility = true;
        },
        releaseVisibilityRead: () => releaseVisibility?.(),
        readVisibility: async () => {
          const shown = state.shown;
          if (deferVisibility) {
            deferVisibility = false;
            state.pendingVisibilityReads++;
            await new Promise<void>((resolve) => {
              releaseVisibility = resolve;
            });
            releaseVisibility = undefined;
            state.pendingVisibilityReads--;
          }
          return shown;
        },
        emit: (name, maximised) => {
          if (maximised !== undefined) state.maximised = maximised;
          const event = name.replace(/^common:/, "");
          if (
            ["WindowShow", "WindowRestore", "WindowUnMinimise"].includes(event)
          ) {
            state.shown = true;
            state.minimised = false;
          } else if (event === "WindowHide") state.shown = false;
          else if (event === "WindowMinimise") state.minimised = true;
          (window as any)._wails?.dispatchWailsEvent({
            name: name.startsWith("common:") ? name : `common:${name}`,
            data: null,
          });
        },
        invoke: async (id) => {
          const method = methods[id];
          state.calls.push(method);
          if (failures.delete(method))
            return {
              status: 500,
              body: JSON.stringify({
                kind: "RuntimeError",
                message: `Window ${method} failed in the fixture.`,
              }),
            };
          let result: boolean | null = null;
          if (method === "IsMaximised") {
            // Capture the native answer before delaying its delivery, so a later
            // event can invalidate this read rather than changing its answer.
            result = state.maximised;
            if (deferRead) {
              deferRead = false;
              state.pendingStateReads++;
              await new Promise<void>((resolve) => {
                releaseRead = resolve;
              });
              releaseRead = undefined;
              state.pendingStateReads--;
            }
          } else if (method === "IsMinimised") {
            result = state.minimised;
          } else if (method === "Minimise") {
            state.minimised = true;
            state.emit("WindowMinimise");
          } else if (method === "ToggleMaximise") {
            state.maximised = !state.maximised;
            state.emit(state.maximised ? "WindowMaximise" : "WindowUnMaximise");
          } else if (method === "Close") {
            // Native Close requests the renderer's draft decision. Hide is a
            // separate transport call, made only after that decision succeeds.
            window.testConnectionWindows?.requestClose();
          } else if (method === "Hide") {
            state.shown = false;
            await window.testConnectionWindows?.hide();
          }
          return { status: 200, body: JSON.stringify(result) };
        },
      };
      window.testWindow = state;
    });
    await page.route("**/*", (route) =>
      new URL(route.request().url()).origin === new URL(baseURL!).origin
        ? route.continue()
        : route.abort(),
    );
    await page.route("**/bindings/**/internal/windowing/service.*", (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `
      const bridge = () => window.testConnectionWindows;
      export const OpenConnectionManager = request => bridge().open(request);
      export const OpenTaskConnection = (request, origin) => bridge().open(request, origin);
      export const TakeSettingsRequest = async () => bridge().take();
      export const SettingsReady = async () => bridge().ready((name, data = null) => window._wails.dispatchWailsEvent({ name, data }));
      export const SettingsVisible = async () => new URLSearchParams(location.search).has('main') ? window.testWindow.readVisibility() : bridge().visible;
      export const FinishSettings = origin => bridge().finish(origin);
      export const HideSettings = () => bridge().hide();
      export const OpenSettings = section => bridge().openSettings(section);
      export const OpenTaskSettings = (section, origin) => bridge().openSettings(section, origin);
      export const ShellReady = async () => bridge().ready((name, data = null) => window._wails.dispatchWailsEvent({ name, data }));
      export const AboutVisible = async () => false;
      export const OpenAbout = async () => {};
      export const HideAbout = async () => {};
      `,
      }),
    );
    await page.route("**/bindings/**/internal/buildinfo/service.*", (route) =>
      route.fulfill({
        contentType: "application/javascript",
        body: `export const Current = async () => ({version: "fixture"});`,
      }),
    );
    await page.route("**/wails/runtime", async (route) => {
      const call = route.request().postDataJSON();
      if (call?.object === 6 && [2, 11, 14, 15, 17, 39].includes(call.method)) {
        const response = await page.evaluate(
          (method) => window.testWindow.invoke(method),
          call.method,
        );
        await route.fulfill({ contentType: "application/json", ...response });
      } else await route.continue();
    });
    await page.goto("/tests/browser/app/");
    await expect(page.locator("#saved-connection-stt")).toHaveValue(
      "Original server",
    );
    await use(page);
    expect(errors, "uncaught browser errors").toEqual([]);
  },
  saves: async ({ page }, use) => {
    let last = 0;
    await use({
      waitForStart: async () => {
        last = await page.evaluate(
          (after) => window.testSaves.waitForStart(after),
          last,
        );
        return last;
      },
      complete: async (id, outcome) => {
        await page.evaluate(
          ({ id, outcome }) => window.testSaves.complete(id, outcome),
          {
            id,
            outcome,
          },
        );
      },
    });
  },
});
export { expect } from "@playwright/test";
