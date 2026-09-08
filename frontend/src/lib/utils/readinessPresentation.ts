import type { Readiness, ReadinessStep } from "./readiness";
import type { SettingsSectionID } from "$lib/navigation";

export type ReadinessAction =
  | { kind: "settings"; label: string; section: SettingsSectionID; step: ReadinessStep["id"] }
  | { kind: "check" | "complete" | "wait"; label: string };

/** Presentation priority only: appReadiness remains the readiness authority. */
export function nextReadinessAction(readiness: Readiness): ReadinessAction {
  if (readiness.canComplete) return { kind: "complete", label: "Finish setup" };
  const prerequisite = readiness.steps.find(
    (step) => step.id !== "connection" && step.blocking && step.status !== "complete",
  );
  if (prerequisite?.status === "pending")
    return { kind: "wait", label: "Looking for a microphone…" };
  if (prerequisite?.settingsSection) {
    const labels = {
      server: "Configure transcription",
      credential: "Review authentication",
      microphone: "Choose microphone",
      shortcut: "Choose shortcut",
      connection: "Review connection",
    };
    return {
      kind: "settings",
      label: labels[prerequisite.id],
      section: prerequisite.settingsSection,
      step: prerequisite.id,
    };
  }
  if (readiness.canTestConnection)
    return {
      kind: "check",
      label: readiness.steps.some((step) => step.id === "connection" && step.status === "attention")
        ? "Check again"
        : "Check connection",
    };
  return { kind: "wait", label: "Waiting for settings…" };
}
