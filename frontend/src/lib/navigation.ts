import type { Component } from "svelte";
import HistoryIcon from "@lucide/svelte/icons/history";
import KeyboardIcon from "@lucide/svelte/icons/keyboard";
import MicIcon from "@lucide/svelte/icons/mic";
import PictureInPictureIcon from "@lucide/svelte/icons/picture-in-picture-2";
import ServerIcon from "@lucide/svelte/icons/server";
import SettingsIcon from "@lucide/svelte/icons/settings";
import WandSparklesIcon from "@lucide/svelte/icons/wand-sparkles";
import Volume2Icon from "@lucide/svelte/icons/volume-2";

export type SettingsSectionID =
  | "general"
  | "shortcuts"
  | "audio"
  | "overlay"
  | "server"
  | "processing"
  | "speech"
  | "history";

export type SettingsSection = {
  id: SettingsSectionID;
  label: string;
  blurb: string;
  icon: Component;
  group: "capture" | "delivery" | "server" | "data";
};

/**
 * The settings navigation. New features add an entry here rather than another
 * card on an ever-growing page, which is the whole point of the two-pane shell.
 */
export const SETTINGS_SECTIONS: SettingsSection[] = [
  {
    id: "shortcuts",
    label: "Shortcuts",
    blurb: "Global keys. They work while any application has focus.",
    icon: KeyboardIcon,
    group: "capture",
  },
  {
    id: "audio",
    label: "Audio",
    blurb: "Capture device and recording limits.",
    icon: MicIcon,
    group: "capture",
  },
  {
    id: "overlay",
    label: "Overlay",
    blurb:
      "Control the passive status indicator shown above other applications.",
    icon: PictureInPictureIcon,
    group: "capture",
  },
  {
    id: "general",
    label: "General",
    blurb: "Startup, transcript delivery and appearance.",
    icon: SettingsIcon,
    group: "delivery",
  },
  {
    id: "server",
    label: "Transcription",
    blurb: "The OpenAI-compatible endpoint that transcribes your speech.",
    icon: ServerIcon,
    group: "server",
  },
  {
    id: "processing",
    label: "Post-processing",
    blurb:
      "Optionally clean completed transcripts with a separate language model.",
    icon: WandSparklesIcon,
    group: "server",
  },
  {
    id: "speech",
    label: "Speech playback",
    blurb:
      "Optionally listen to completed transcripts through a separate TTS endpoint.",
    icon: Volume2Icon,
    group: "delivery",
  },
  {
    id: "history",
    label: "History",
    blurb: "An in-memory safety net for transcripts that did not land.",
    icon: HistoryIcon,
    group: "data",
  },
];

export const GROUP_LABELS: Record<SettingsSection["group"], string> = {
  capture: "Capture",
  delivery: "Delivery",
  server: "Server",
  data: "Data",
};

export const sectionsInGroup = (
  group: SettingsSection["group"],
): SettingsSection[] =>
  SETTINGS_SECTIONS.filter((section) => section.group === group);

export const sectionByID = (id: SettingsSectionID): SettingsSection =>
  SETTINGS_SECTIONS.find((section) => section.id === id) ??
  SETTINGS_SECTIONS[0];
