import { Purpose, type Connection } from "$bindings/savedconnection";
import type { SettingsSectionID } from "$lib/navigation";

export const connectionWorkflows: { id: Purpose; label: string; section: SettingsSectionID }[] = [
  { id: Purpose.Voice, label: "Voice transcription", section: "voice-transcription" },
  { id: Purpose.Transcription, label: "Audio-file transcription", section: "server" },
  { id: Purpose.Cleanup, label: "Cleanup", section: "processing" },
  { id: Purpose.Speech, label: "Text to speech", section: "speech" },
];
export function connectionMatches(connection: Connection, query: string): boolean {
  const words = query.trim().toLocaleLowerCase().split(/\s+/);
  const text =
    `${connection.name} ${connection.details.baseURL} ${connection.details.compatibilityProfile}`.toLocaleLowerCase();
  return words.every((word) => text.includes(word));
}
export function connectionSection(purpose: Purpose): SettingsSectionID {
  return connectionWorkflows.find((workflow) => workflow.id === purpose)?.section ?? "connections";
}
