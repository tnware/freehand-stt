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
  const empty = (): SettingsRequest => ({ section: "", origin: "" });
  const emitMain: Dispatch = (name, data = null) => {
    (window as any)._wails.dispatchWailsEvent({ name, data: wire(data) });
  };
  const reveal = async (request: SettingsRequest) => {
    const bridge = window.testConnectionWindows;
    // Native publishes the latest request; the renderer owns draft guards.
    bridge.requestState = { pending: true, request: wire(request) };
    bridge.visible = true;
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
      if (window.testConnectionWindows.requestState.pending)
        dispatch("settings:open");
    },
    open: async (connection, origin = "") =>
      reveal({ section: "connections", origin, connection: wire(connection) }),
    hide: async () => {
      window.testConnectionWindows.visible = false;
      window.testConnectionWindows.requestState = {
        pending: false,
        request: empty(),
      };
      dispatch?.("shell:hidden");
    },
    finish: async (origin) => {
      if (origin) emitMain("workspace:select-task", origin);
    },
    requestClose: () => dispatch?.("shell:close-requested"),
    openSettings: async (section, origin = "") => reveal({ section, origin }),
  };
}
