import { mount } from "svelte";
import AboutWindow from "./AboutWindow.svelte";
import ConnectionManagerWindow from "./ConnectionManagerWindow.svelte";
import App from "./App.svelte";
import HistoryDetailsWindow from "./HistoryDetailsWindow.svelte";
import SettingsWindow from "./SettingsWindow.svelte";
import "./app.css";

const Root = window.location.hash.startsWith("#connections")
  ? ConnectionManagerWindow
  : window.location.hash.startsWith("#settings")
    ? SettingsWindow
    : window.location.hash.startsWith("#about")
      ? AboutWindow
      : window.location.hash.startsWith("#transcription-details")
        ? HistoryDetailsWindow
        : App;

mount(Root, { target: document.getElementById("app")! });
