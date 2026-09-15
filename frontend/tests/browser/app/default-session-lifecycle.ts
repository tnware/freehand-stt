import { mount, tick, unmount } from "svelte";
import App from "../../../src/App.svelte";
import "../../../src/app.css";
import { session } from "$lib/stores/session.svelte";
import { idle, settings } from "$lib/stores/session-fixtures-data";
import type { InstanceRequest, InstanceStatus } from "$bindings/managedruntime";
import { createRuntimeFixture } from "./runtime-fixture";
import { installConnectionWindows } from "./connection-window-bridge";

type RuntimeMethod = "GetInstances" | "GetProviders" | "Start" | "Stop";

declare global {
  interface Window {
    testDefaultSessionRuntime: {
      invoke: (
        method: RuntimeMethod,
        request?: InstanceRequest,
      ) => Promise<unknown>;
    };
  }
}

/** The real default singleton and generated runtime bindings, across App mounts. */
export async function mountDefaultSessionApp() {
  const current = structuredClone(settings);
  current.setupCompleted = true;
  const runtime = createRuntimeFixture(true, (instances) => {
    current.managedRuntimes = instances;
  });
  const emitStatus = (row: InstanceStatus) => {
    (window as any)._wails.dispatchWailsEvent({
      name: "managed-runtime:status",
      data: row,
    });
  };
  runtime.subscribe(emitStatus);
  window.testDefaultSessionRuntime = {
    invoke: async (method, request) => {
      if (method === "GetInstances" || method === "GetProviders")
        return runtime.service[method]();
      if (!request) throw new Error("Missing runtime request");
      return runtime.service[method](request);
    },
  };
  installConnectionWindows(
    {
      settings: () => current,
      save: async () => current,
      apply: (snapshot) => session.editor.applySettingsSnapshot(snapshot),
    },
    true,
  );

  // Other feature services are outside this lifecycle regression. Keep Session's
  // real load orchestration and the runtime's real generated bridge untouched.
  session.dictation.load = async () => {
    session.dictation.applyStatus(idle);
  };
  session.speech.load = async () => {};
  session.files.refresh = async () => {};
  session.editor.load = async () =>
    session.editor.applySettingsSnapshot(current);
  session.editor.refreshDevices = async () => {};
  session.history.refresh = async () => {};
  session.resources.setVisible = () => {};

  let statusDeliveries = 0;
  const applyStatus = session.runtime.applyStatus.bind(session.runtime);
  session.runtime.applyStatus = (row) => {
    statusDeliveries++;
    applyStatus(row);
  };
  const target = document.createElement("div");
  target.id = "app";
  document.body.append(target);
  let app: ReturnType<typeof mount> | undefined;
  const render = async () => {
    if (app) throw new Error("App is already mounted");
    // Deliberately omit a session prop: imported fallback consumers must share
    // this same module-owned default session after App is remounted by HMR.
    app = mount(App, { target });
    await tick();
  };
  const teardown = async () => {
    if (app) await unmount(app);
    app = undefined;
    await tick();
  };
  await render();
  return {
    render,
    teardown,
    calls: () => [...runtime.control.calls],
    deliveries: () => statusDeliveries,
    emit: async (state: string) => {
      const row = runtime.control.snapshot()[0];
      row.status.state = state;
      emitStatus(row);
      await tick();
    },
    state: () => session.runtime.instances[0]?.status.state,
    enterCredentials: () => {
      session.editor.apiKey = "fixture-stt-secret";
      session.editor.processingAPIKey = "fixture-cleanup-secret";
      session.editor.ttsAPIKey = "fixture-speech-secret";
    },
    credentialsCleared: () =>
      !session.editor.apiKey &&
      !session.editor.processingAPIKey &&
      !session.editor.ttsAPIKey,
    run: () => session.runtime.run("nemo-default", "Stop"),
    destroy: async () => {
      await teardown();
      session.dispose();
      target.remove();
    },
  };
}
