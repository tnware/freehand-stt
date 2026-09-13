import type {
  ConnectionManagerRequest,
  ConnectionManagerState,
} from "$bindings/windowing";
import type { SettingsDTO, SaveSettingsRequest } from "$bindings/settings";
import { Purpose } from "$bindings/savedconnection";

export type ConnectionWindowEvent =
  "connections:open" | "connections:close-requested" | "common:WindowHide";

declare global {
  interface Window {
    testConnectionWindows: {
      requestState: ConnectionManagerState;
      open: (request: ConnectionManagerRequest) => Promise<void>;
      hide: () => Promise<void>;
      requestClose: () => void;
      ready: (dispatch: (name: ConnectionWindowEvent) => void) => void;
      settings: () => SettingsDTO;
      save: (request: SaveSettingsRequest) => Promise<SettingsDTO>;
      openSettings: (section: string) => Promise<void>;
    };
  }
}

// One retained iframe models one native WebView/Session for the entire test.
// Only service/window boundaries are substituted; editor and lifecycle handlers
// are real. This browser proxy cannot establish native window/focus acceptance.
export function installConnectionWindows(backend: {
  settings: () => SettingsDTO;
  save: (request: SaveSettingsRequest) => Promise<SettingsDTO>;
  apply: (settings: SettingsDTO) => void;
  select: (section: string) => void;
}) {
  let frame: HTMLIFrameElement | undefined;
  let dispatch: ((name: ConnectionWindowEvent) => void) | undefined;
  let origin: HTMLElement | null = null;
  const emptyRequest = (): ConnectionManagerRequest => ({
    id: "",
    purpose: Purpose.$zero,
    create: false,
  });
  window.testConnectionWindows = {
    requestState: { visible: false, request: emptyRequest() },
    settings: backend.settings,
    save: async (request) => {
      const saved = await backend.save(request);
      backend.apply(saved);
      return saved;
    },
    ready: (emit) => {
      dispatch = emit;
      // Mirrors WindowRuntimeReady; an early reveal event may precede listeners.
      dispatch("connections:open");
    },
    open: async (request) => {
      const state = window.testConnectionWindows.requestState;
      // Like the Go service, revealing an already-owned window preserves drafts.
      if (!state.visible) {
        state.request = JSON.parse(JSON.stringify(request));
        origin = document.activeElement as HTMLElement;
      }
      state.visible = true;
      if (!frame) {
        frame = document.createElement("iframe");
        frame.title = "Connections window";
        frame.src = "/connection-manager-fixture";
        frame.style.cssText =
          "position:fixed;inset:0;width:100%;height:100%;border:0;z-index:100;background:var(--background)";
        document.body.append(frame);
      }
      frame.hidden = false;
      dispatch?.("connections:open");
    },
    requestClose: () => {
      // Native WindowClosing is cancelled. The real manager decides whether to
      // prompt, ignore a busy close request, or call HideConnectionManager.
      if (window.testConnectionWindows.requestState.visible)
        dispatch?.("connections:close-requested");
    },
    hide: async () => {
      if (frame) frame.hidden = true;
      window.testConnectionWindows.requestState = {
        visible: false,
        request: emptyRequest(),
      };
      dispatch?.("common:WindowHide");
      // Explicit browser focus proxy only, not evidence of native focus restore.
      origin?.focus();
    },
    openSettings: async (section) => backend.select(section),
  };
}
