import { mount } from "svelte";
import AboutWindow from "./AboutWindow.svelte";
import App from "./App.svelte";
import SettingsHost from "./SettingsHost.svelte";
import TrayPopover from "./TrayPopover.svelte";
import HistoryDetailsWindow from "./HistoryDetailsWindow.svelte";
import ProcessOutputWindow from "./ProcessOutputWindow.svelte";
import "./app.css";
const target = document.getElementById("app")!;
if (
  new URLSearchParams(window.location.search).get("window") === "tray-popover"
)
  mount(TrayPopover, { target });
else if (
  new URLSearchParams(window.location.search).get("window") === "settings"
)
  mount(SettingsHost, { target });
else if (
  new URLSearchParams(window.location.search).get("window") === "process-output"
)
  mount(ProcessOutputWindow, { target });
else if (window.location.hash.startsWith("#about"))
  mount(AboutWindow, { target });
else if (window.location.hash.startsWith("#transcription-details"))
  mount(HistoryDetailsWindow, { target });
else mount(App, { target });
