import PickerFeedbackFixture from "./PickerFeedbackFixture.svelte";
import LongContentFixture from "./LongContentFixture.svelte";
import TrayFixture from "./TrayFixture.svelte";
import VocabularyFixture from "./VocabularyFixture.svelte";
import TranscriptFixture from "./TranscriptFixture.svelte";
import { mount } from "svelte";
import SettingsFixture from "./SettingsFixture.svelte";
import WorkspaceFixture from "./WorkspaceFixture.svelte";
import "../../../src/app.css";
const params = new URLSearchParams(location.search);
document.documentElement.classList.toggle(
  "dark",
  params.get("theme") === "dark",
);
mount(
  params.get("view") === "picker-feedback"
    ? PickerFeedbackFixture
    : params.get("view") === "long-content"
      ? LongContentFixture
      : params.get("view") === "tray"
        ? TrayFixture
        : params.get("view") === "vocabulary"
          ? VocabularyFixture
          : params.get("view") === "transcript"
            ? TranscriptFixture
            : params.get("view") === "workspace"
              ? WorkspaceFixture
              : SettingsFixture,
  {
    target: document.getElementById("app")!,
  },
);
