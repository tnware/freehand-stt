import type { Component } from "svelte";
import FileAudioIcon from "@lucide/svelte/icons/file-audio";
import MicIcon from "@lucide/svelte/icons/mic";
import CpuIcon from "@lucide/svelte/icons/cpu";
import HistoryIcon from "@lucide/svelte/icons/history";
import SlidersIcon from "@lucide/svelte/icons/sliders-horizontal";
import Volume2Icon from "@lucide/svelte/icons/volume-2";

/**
 * The main window is a set of places reached from one activity rail, rather
 * than a task workspace with a separate configuration window beside it. A
 * workflow pane owns a signal chain; an auxiliary pane owns an inventory.
 */
export const WORKFLOW_PANES = ["voice", "file", "tts"] as const;
export type WorkflowPane = (typeof WORKFLOW_PANES)[number];

export const AUXILIARY_PANES = ["runtimes", "history", "settings"] as const;
export type AuxiliaryPane = (typeof AUXILIARY_PANES)[number];

export type PaneID = WorkflowPane | AuxiliaryPane;

export function isWorkflowPane(value: string): value is WorkflowPane {
  return (WORKFLOW_PANES as readonly string[]).includes(value);
}

export type Pane = {
  id: PaneID;
  /** The rail is icon-only, so this is the accessible name and the tooltip. */
  label: string;
  icon: Component;
  /** Workflow panes sit at the top of the rail; configuration sits at the foot. */
  place: "top" | "foot";
};

export const PANES: Pane[] = [
  { id: "voice", label: "Voice transcription", icon: MicIcon, place: "top" },
  { id: "file", label: "Audio file", icon: FileAudioIcon, place: "top" },
  { id: "tts", label: "Text to speech", icon: Volume2Icon, place: "top" },
  { id: "runtimes", label: "Local runtime", icon: CpuIcon, place: "top" },
  { id: "history", label: "History", icon: HistoryIcon, place: "top" },
  { id: "settings", label: "Settings", icon: SlidersIcon, place: "foot" },
];

export function paneByID(id: PaneID): Pane {
  return PANES.find((pane) => pane.id === id) ?? PANES[0];
}

/**
 * Switching workflow mid-run would leave a recorder or an upload without the
 * surface that owns it, so the rail states the reason rather than going quiet.
 */
export function workflowBlockedReason(
  id: PaneID,
  voiceActive: boolean,
  fileWorking: boolean,
  ttsWorking = false,
): string {
  if (id === "voice" && fileWorking) return "Finish the audio file first";
  if (id === "file" && voiceActive) return "Finish the recording first";
  if (id === "tts" && (voiceActive || fileWorking))
    return voiceActive
      ? "Finish the recording first"
      : "Finish the audio file first";
  if (id === "voice" && ttsWorking) return "";
  return "";
}
