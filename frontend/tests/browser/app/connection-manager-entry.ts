import { mount, tick } from "svelte";
import ConnectionManagerWindow from "../../../src/ConnectionManagerWindow.svelte";
import type { ConnectionWindowEvent } from "./connection-window-bridge";
import "../../../src/app.css";

mount(ConnectionManagerWindow, { target: document.getElementById("app")! });

// Receive the same event envelope as Wails, without emitting to a native app.
// Wait for onMount subscriptions before modelling WindowRuntimeReady.
void tick().then(() => {
  const runtime = window as typeof window & {
    _wails: {
      dispatchWailsEvent: (event: {
        name: ConnectionWindowEvent;
        data: null;
        sender?: string;
      }) => void;
    };
  };
  window.parent.testConnectionWindows.ready((name) => {
    runtime._wails.dispatchWailsEvent({
      name,
      data: null,
      ...(name === "common:WindowHide" ? {} : { sender: "connections" }),
    });
  });
});
