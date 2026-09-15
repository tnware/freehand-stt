import { expect, type Page } from "@playwright/test";
import type { SettingsSectionID } from "../../src/lib/navigation";

const labels: Record<SettingsSectionID, string> = {
  "voice-transcription": "Voice transcription",
  server: "Audio-file transcription",
  processing: "Cleanup",
  speech: "Text to speech",
  audio: "Audio",
  connections: "Connections",
  "local-runtime": "Local runtime",
  history: "History",
  general: "General",
  shortcuts: "Shortcuts",
  overlay: "Overlay",
  vocabulary: "Vocabulary",
};

/** Follow the public command route, including draft guards and global shared sections. */
export async function openSection(page: Page, id: SettingsSectionID) {
  await page.getByRole("button", { name: /^Run a command/ }).click();
  const command = page.getByRole("combobox", { name: "Command", exact: true });
  await expect(command).toBeVisible();
  await command.fill(labels[id]);
  await page.locator(`[id="command-settings:${id}"]`).click();
}
