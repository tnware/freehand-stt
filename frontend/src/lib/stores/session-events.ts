import { Events } from "@wailsio/runtime";
import {
  FileTranscriptionPhase,
  State,
  type FileTranscriptionDelta,
  type FileTranscriptionStatus,
  type Settings,
  type Status,
  type TTSStatus,
} from "$lib/state";
import type { Session } from "./session.svelte";

export interface SessionEventMap {
  "dictation:status": Status;
  "file-transcription:status": FileTranscriptionStatus;
  "file-transcription:delta": FileTranscriptionDelta;
  "tts:status": TTSStatus;
  "settings:changed": Settings;
}

export type SubscribeSessionEvent = <K extends keyof SessionEventMap>(
  name: K,
  callback: (event: { data: SessionEventMap[K] }) => void,
) => () => void;

const fileActive = (phase: FileTranscriptionPhase) =>
  phase === FileTranscriptionPhase.FileTranscriptionUploading ||
  phase === FileTranscriptionPhase.FileTranscriptionProcessing ||
  phase === FileTranscriptionPhase.FileTranscriptionStreaming ||
  phase === FileTranscriptionPhase.FileTranscriptionCancelling;

/**
 * Shared per-window event wiring. Subscribe before loading snapshots; feature
 * owners reconcile values, while this composition owns cross-feature reactions.
 * The returned disposer releases only renderer subscriptions, never Go work.
 */
export function subscribeSessionEvents(
  session: Session,
  on: SubscribeSessionEvent = Events.On,
  dictationChanged: (status: Status, previous: Status) => void = () => {},
): () => void {
  const subscriptions: (() => void)[] = [];
  let active = true;
  const dispose = () => {
    if (!active) return;
    active = false;
    for (const off of subscriptions.reverse()) off();
  };
  const subscribe: SubscribeSessionEvent = (name, callback) => {
    const off = on(name, (event) => {
      if (active) callback(event);
    });
    subscriptions.push(off);
    return off;
  };
  try {
    subscribe("dictation:status", ({ data }) => {
      const previous = session.dictation.status;
      session.dictation.applyStatus(data);
      if (data.state === State.Idle || data.state === State.Failed)
        void session.history.refresh();
      dictationChanged(data, previous);
    });
    subscribe("file-transcription:status", ({ data }) => {
      const previous = session.files.status;
      const accepted = session.files.applyStatus(data);
      // Rejected generations/revisions cannot trigger history side effects.
      if (accepted && fileActive(previous.phase) && !fileActive(data.phase))
        void session.history.refresh();
    });
    subscribe("file-transcription:delta", ({ data }) => {
      if (session.files.applyDelta(data) === "gap")
        void session.files.refresh();
    });
    subscribe("tts:status", ({ data }) => session.speech.applyStatus(data));
    subscribe("settings:changed", ({ data }) =>
      session.editor.applySettingsSnapshot(data),
    );
  } catch (cause) {
    dispose();
    throw cause;
  }
  return dispose;
}
