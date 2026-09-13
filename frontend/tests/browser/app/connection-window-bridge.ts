import type {
  ConnectionManagerRequest,
  SettingsRequest,
  SettingsRequestState,
} from "$bindings/windowing";
import type { SettingsDTO, SaveSettingsRequest } from "$bindings/settings";

// All DTOs cross the same JSON boundary as the native bridge. Sessions do not.
export const wire = <T>(value: T): T => JSON.parse(JSON.stringify(value));
type Dispatch = (name: string, data?: unknown) => void;
declare global {
  interface Window {
    testConnectionWindows: {
      requestState: SettingsRequestState;
      visible: boolean;
      open: (
        request: ConnectionManagerRequest,
        origin?: string,
      ) => Promise<void>;
      hide: () => Promise<void>;
      requestClose: () => void;
      ready: (dispatch: Dispatch) => Promise<void>;
      take: () => SettingsRequestState | Promise<SettingsRequestState>;
      finish: (origin: string) => Promise<void>;
      settings: () => SettingsDTO;
      save: (request: SaveSettingsRequest) => Promise<SettingsDTO>;
      openSettings: (section: string, origin?: string) => Promise<void>;
    };
  }
}
export function installConnectionWindows(
  backend: {
    settings: () => SettingsDTO;
    save: (request: SaveSettingsRequest) => Promise<SettingsDTO>;
    apply: (settings: SettingsDTO) => void;
  },
  integrated = false,
  general = false,
) {
  let dispatch: Dispatch | undefined;
  let frame: HTMLIFrameElement | undefined;
  const empty = (): SettingsRequest => ({ section: "", origin: "" });
  const emitMain: Dispatch = (name, data = null) => {
    (window as any)._wails.dispatchWailsEvent({ name, data: wire(data) });
  };
  const reveal = async (request: SettingsRequest) => {
    const bridge = window.testConnectionWindows;
    // Native publishes the latest request; the renderer owns draft guards.
    bridge.requestState = { pending: true, request: wire(request) };
    bridge.visible = true;
    if (integrated && !frame) {
      frame = document.createElement("iframe");
      frame.dataset.window = "settings";
      frame.title = "Settings";
      frame.style.cssText =
        "position:fixed;inset:0;width:100%;height:100%;border:0;z-index:1000;background:white";
      const url = new URL(location.href);
      url.searchParams.set("settings-frame", "1");
      frame.src = url.href;
      document.body.append(frame);
    }
    if (frame) frame.hidden = false;
    dispatch?.("settings:visibility", true);
    dispatch?.("settings:open");
  };
  window.testConnectionWindows = {
    requestState: integrated
      ? { pending: false, request: empty() }
      : {
          pending: true,
          request: { section: "server", origin: general ? "" : "file" },
        },
    visible: !integrated,
    settings: () => wire(backend.settings()),
    save: async (request) => {
      const saved = wire(await backend.save(wire(request)));
      backend.apply(wire(saved));
      return saved;
    },
    take: () => {
      const state = wire(window.testConnectionWindows.requestState);
      window.testConnectionWindows.requestState = {
        pending: false,
        request: empty(),
      };
      return state;
    },
    ready: async (callback) => {
      dispatch = callback;
    },
    open: async (connection, origin = "") =>
      reveal({ section: "connections", origin, connection: wire(connection) }),
    hide: async () => {
      window.testConnectionWindows.visible = false;
      window.testConnectionWindows.requestState = {
        pending: false,
        request: empty(),
      };
      dispatch?.("settings:visibility", false);
      if (frame) frame.hidden = true;
    },
    finish: async (origin) => {
      await window.testConnectionWindows.hide();
      if (integrated && origin) emitMain("workspace:select-task", origin);
    },
    requestClose: () => dispatch?.("settings:close-requested"),
    openSettings: async (section, origin = "") => reveal({ section, origin }),
  };
}
