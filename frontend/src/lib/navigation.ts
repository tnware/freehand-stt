import type { Component } from "svelte";
import BookOpenIcon from "@lucide/svelte/icons/book-open";
import HistoryIcon from "@lucide/svelte/icons/history";
import KeyboardIcon from "@lucide/svelte/icons/keyboard";
import MicIcon from "@lucide/svelte/icons/mic";
import PictureInPictureIcon from "@lucide/svelte/icons/picture-in-picture-2";
import FileAudioIcon from "@lucide/svelte/icons/file-audio";
import ServerIcon from "@lucide/svelte/icons/server";
import SettingsIcon from "@lucide/svelte/icons/settings";
import WandSparklesIcon from "@lucide/svelte/icons/wand-sparkles";
import Volume2Icon from "@lucide/svelte/icons/volume-2";

export const SETTINGS_NAVIGATION = Symbol("settings-navigation");

export type SettingsSectionID =
  | "general"
  | "shortcuts"
  | "audio"
  | "overlay"
  | "connections"
  | "voice-transcription"
  | "server"
  | "processing"
  | "speech"
  | "history"
  | "vocabulary";

export type SettingsSection = {
  id: SettingsSectionID;
  label: string;
  blurb: string;
  icon: Component;
  group: "workflows" | "shared" | "capture" | "application";
};

/**
 * The settings navigation. New features add an entry here rather than another
 * card on an ever-growing page, which is the whole point of the two-pane shell.
 */
export const SETTINGS_SECTIONS: SettingsSection[] = [
  {
    id: "voice-transcription",
    label: "Voice transcription",
    blurb: "Choose a microphone transcription provider, model, and supported recording mode.",
    icon: MicIcon,
    group: "workflows",
  },
  {
    id: "server",
    label: "Audio-file transcription",
    blurb: "Choose a connection, model, language, and transcription options.",
    icon: FileAudioIcon,
    group: "workflows",
  },
  {
    id: "processing",
    label: "Cleanup",
    blurb: "Optionally clean completed transcripts with a separate language model.",
    icon: WandSparklesIcon,
    group: "workflows",
  },
  {
    id: "speech",
    label: "Text to speech",
    blurb: "Write text to speak, or listen to completed transcripts.",
    icon: Volume2Icon,
    group: "workflows",
  },
  {
    id: "connections",
    label: "Connections",
    blurb: "Create and manage saved server connections.",
    icon: ServerIcon,
    group: "shared",
  },
  {
    id: "vocabulary",
    label: "Vocabulary",
    blurb: "Names and terminology, shared across your transcription models.",
    icon: BookOpenIcon,
    group: "shared",
  },
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
    id: "general",
    label: "General",
    blurb: "Startup, transcript delivery and appearance.",
    icon: SettingsIcon,
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
  workflows: "Workflows",
  shared: "Connections & vocabulary",
  application: "Application",
};

export const SETTINGS_GROUPS = ["workflows", "shared", "capture", "application"] as const;

export const sectionsInGroup = (group: SettingsSection["group"]): SettingsSection[] =>
  SETTINGS_SECTIONS.filter((section) => section.group === group);

export const sectionByID = (id: SettingsSectionID): SettingsSection =>
  SETTINGS_SECTIONS.find((section) => section.id === id) ?? SETTINGS_SECTIONS[0];

/** Search terms mirror settings people look for, including controls below disclosures. */
const SETTINGS_KEYWORDS: Record<SettingsSectionID, string> = {
  "voice-transcription": "realtime live language context prompt model profile",
  server: "file upload language context prompt streaming timeout",
  processing: "post processing s1 mini instructions reasoning temperature timeout",
  speech: "tts voice preview speed language style instructions timeout",
  connections: "backend provider endpoint url api key authentication headers server",
  vocabulary: "names phrases terminology hotwords boosting",
  shortcuts: "hotkey keyboard hold toggle cancel record",
  audio:
    "microphone duration speech detection vad silence trim padding automatic stop segment split sensitivity",
  overlay: "layout position size surface glass glow opacity visualizer motion animation preview",
  general:
    "startup login tray updates theme light dark system mica clipboard direct input manual copy",
  history: "retention recent transcripts clear delete memory",
};

export function matchingSettingsSections(query: string): SettingsSection[] {
  const words = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return SETTINGS_SECTIONS.filter((section) => {
    const text =
      `${section.label} ${section.blurb} ${SETTINGS_KEYWORDS[section.id]}`.toLocaleLowerCase();
    return words.every((word) => text.includes(word));
  });
}
