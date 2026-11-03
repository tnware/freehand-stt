import { Action, Purpose } from "$bindings/savedconnection";
import { Role } from "$bindings/compatibility";
import type { WorkflowPane } from "$lib/panes";
import type { Session } from "$lib/stores/session.svelte";
import { TTSPhase } from "$lib/state";

export const runtimeWorkflowTargets: Record<
  WorkflowPane,
  { purpose: Purpose; label: string; role: Role }
> = {
  voice: { purpose: Purpose.Voice, label: "Voice", role: Role.Transcription },
  file: {
    purpose: Purpose.Transcription,
    label: "Audio file",
    role: Role.Transcription,
  },
  tts: { purpose: Purpose.Speech, label: "Text to speech", role: Role.Speech },
};

/** Only offer the backend-owned connection for the current, qualified model. */
export function runtimeWorkflowConnection(
  session: Session,
  instanceID: string,
  workflow: WorkflowPane,
) {
  const row = session.runtime.statusFor(instanceID);
  if (
    !row?.status.supported ||
    row.status.state !== "running" ||
    row.status.selectedModel !== row.instance.model ||
    (workflow === "tts" &&
      row.instance.speechModel &&
      row.status.selectedSpeechModel !== row.instance.speechModel)
  )
    return;
  const target = runtimeWorkflowTargets[workflow];
  const modelID =
    workflow === "tts"
      ? row.instance.speechModel || row.instance.model
      : row.instance.model;
  const provider = session.runtime.providers.find(
    (entry) => entry.id === row.instance.provider,
  );
  if (
    !provider?.supported ||
    !provider.models?.some(
      (model) =>
        model.id === modelID &&
        model.contracts?.some((contract) => contract.role === target.role),
    )
  )
    return;
  return session.editor.applied?.savedConnections.entries?.find(
    (connection) =>
      connection.builtIn &&
      connection.details.managedInstanceID === instanceID &&
      connection.uses?.includes(target.purpose),
  );
}

export function runtimeUseBlockReason(
  session: Session,
  workBusy = false,
): string {
  const editor = session.editor;
  if (
    workBusy ||
    session.speech.submitting ||
    session.speech.previewing ||
    session.speech.listening !== null ||
    session.speech.status.phase === TTSPhase.Generating
  )
    return "Finish the current task before changing its connection.";
  if (
    !editor.applied ||
    !editor.draft ||
    editor.applied.configuration.recoveryRequired
  )
    return "Resolve settings recovery before choosing a runtime.";
  if (editor.dirty || editor.connectionDraft !== null)
    return "Save or discard your settings edits before choosing a runtime.";
  if (
    editor.saving ||
    editor.setupCompleting ||
    editor.configurationRetrying ||
    editor.configurationResetting ||
    editor.quickSettingsPending.length ||
    session.runtime.loading ||
    session.runtime.busy
  )
    return "Wait for the current change to finish.";
  return "";
}

/** Explicit selection through the same transaction used by the workflow picker. */
export async function useRuntimeConnection(
  session: Session,
  instanceID: string,
  workflow: WorkflowPane,
  workBusy = false,
): Promise<boolean> {
  if (runtimeUseBlockReason(session, workBusy)) return false;
  const connection = runtimeWorkflowConnection(session, instanceID, workflow);
  if (!connection) return false;
  const { purpose } = runtimeWorkflowTargets[workflow];
  if (
    session.editor.applied?.savedConnections.selected?.[purpose] ===
    connection.id
  )
    return true;
  return session.editor.changeConnection({
    action: Action.Select,
    purpose,
    id: connection.id,
    name: "",
  });
}
