import { mount } from "svelte";
import SettingsFixture from "./SettingsFixture.svelte";
import "../../../src/app.css";
mount(SettingsFixture, { target: document.getElementById("app")! });
