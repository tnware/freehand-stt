import type { Component } from "svelte";
import HistoryIcon from "@lucide/svelte/icons/history";
import KeyboardIcon from "@lucide/svelte/icons/keyboard";
import MicIcon from "@lucide/svelte/icons/mic";
import PictureInPictureIcon from "@lucide/svelte/icons/picture-in-picture-2";
import FileAudioIcon from "@lucide/svelte/icons/file-audio";
import ServerIcon from "@lucide/svelte/icons/server";
import SettingsIcon from "@lucide/svelte/icons/settings";
import WandSparklesIcon from "@lucide/svelte/icons/wand-sparkles";
import Volume2Icon from "@lucide/svelte/icons/volume-2";

export type SettingsSectionID =
  | "general"
  | "shortcuts"
  | "audio"
  | "overlay"
  | "connections"
  | "server"
  | "processing"
  | "speech"
  | "history";

export type SettingsSection = {
  id: SettingsSectionID;
  label: string;
  blurb: string;
  icon: Component;
  group: "capture" | "features" | "application";
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
    blurb: "Control the passive status indicator shown above other applications.",
    icon: PictureInPictureIcon,
    group: "capture",
  },
  {
    id: "server",
    label: "Transcription",
    blurb: "Choose a connection, model, language, and transcription options.",
    icon: FileAudioIcon,
    group: "features",
  },
  {
    id: "processing",
    label: "Cleanup",
    blurb: "Optionally clean completed transcripts with a separate language model.",
    icon: WandSparklesIcon,
    group: "features",
  },
  {
    id: "speech",
    label: "Text to speech",
    blurb: "Write text to speak, or listen to completed transcripts.",
    icon: Volume2Icon,
    group: "features",
  },
  {
    id: "general",
    label: "General",
    blurb: "Startup, transcript delivery and appearance.",
    icon: SettingsIcon,
    group: "application",
  },
  {
    id: "connections",
    label: "Connections",
    blurb: "Create and manage saved server connections.",
    icon: ServerIcon,
    group: "application",
  },
  {
    id: "history",
    label: "History",
    blurb: "Optional recent transcripts, kept in memory until you quit.",
    icon: HistoryIcon,
    group: "application",
  },
];

export const GROUP_LABELS: Record<SettingsSection["group"], string> = {
  capture: "Capture",
  features: "Features",
  application: "Application",
};

export const sectionsInGroup = (group: SettingsSection["group"]): SettingsSection[] =>
  SETTINGS_SECTIONS.filter((section) => section.group === group);

export const sectionByID = (id: SettingsSectionID): SettingsSection =>
  SETTINGS_SECTIONS.find((section) => section.id === id) ?? SETTINGS_SECTIONS[0];
