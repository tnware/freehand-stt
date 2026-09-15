import type { SettingsSectionID } from "./navigation";
import type { PaneID, WorkflowPane } from "./panes";

export const GENERAL_SECTIONS: SettingsSectionID[] = [
  "general",
  "shortcuts",
  "overlay",
  "vocabulary",
];
export const WORKFLOW_SECTIONS: Record<WorkflowPane, SettingsSectionID[]> = {
  voice: [
    "voice-transcription",
    "audio",
    "processing",
    "vocabulary",
    "overlay",
    "general",
  ],
  file: ["server", "processing", "vocabulary"],
  tts: ["speech"],
};

/** Default section destinations; workflow sidebars can expose shared preferences locally. */
export function configurationRealm(
  section: SettingsSectionID,
  workflow: string,
): PaneID {
  switch (section) {
    case "voice-transcription":
    case "audio":
      return "voice";
    case "server":
      return "file";
    case "speech":
      return "tts";
    case "processing":
      return workflow === "file" ? "file" : "voice";
    case "history":
      return "history";
    case "connections":
      return "connections";
    case "local-runtime":
      return "runtimes";
    default:
      return "settings";
  }
}
