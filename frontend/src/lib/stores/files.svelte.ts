import {
  FileTranscriptionPhase,
  type FileTranscriptionStatus,
  type FileTranscriptionDelta,
} from "$lib/state";
import type { SessionMessages } from "./messages.svelte";
import type * as FiletranscriptionBindings from "$bindings/filetranscription/service";
export type FileTranscriptionStateService = Pick<
  typeof FiletranscriptionBindings,
  | "CurrentFileTranscription"
  | "ChooseAudioFile"
  | "ClearAudioFile"
  | "StartFileTranscription"
  | "TryFileStreamingAgain"
  | "CancelFileTranscription"
  | "CopyFileTranscript"
>;

const EMPTY_FILE: FileTranscriptionStatus = {
  generation: 0,
  phase: FileTranscriptionPhase.FileTranscriptionEmpty,
  streaming: false,
  buffered: false,
  streamingProfileUnavailable: false,
  streamingUnavailable: false,
  transcriptRevision: 0,
  canStart: false,
  canCancel: false,
  canCopy: false,
};

/** Owns stored-file selection state and snapshot/delta reconciliation. */
export class FileTranscriptionState {
  readonly #service: FileTranscriptionStateService;
  readonly #messages: SessionMessages;
  constructor(
    service: FileTranscriptionStateService,
    messages: SessionMessages,
  ) {
    this.#service = service;
    this.#messages = messages;
  }
  status = $state<FileTranscriptionStatus>(EMPTY_FILE);
  choosing = $state(false);
  starting = $state(false);
  cancelling = $state(false);
  clearing = $state(false);
  streamingPreferred = $state(true);
  resettingStreaming = $state(false);
  streamingEnabled = $derived(
    this.streamingPreferred && !this.status.streamingUnavailable,
  );
  selectionBusy = $derived(
    this.choosing || this.starting || this.clearing || this.resettingStreaming,
  );
  historyGeneration = $state(0);
  #fileStatusRevision = 0;
  #fileStatusRequest = 0;

  applyStatus(status: FileTranscriptionStatus): boolean {
    if (status.generation < this.status.generation) return false;
    if (
      status.generation === this.status.generation &&
      status.transcriptRevision < this.status.transcriptRevision
    )
      return false;
    this.#fileStatusRevision++;
    this.status = status;
    return true;
  }

  applyDelta(delta: FileTranscriptionDelta): "applied" | "ignored" | "gap" {
    if (delta.generation !== this.status.generation || !delta.text)
      return "ignored";
    if (
      this.status.phase !== FileTranscriptionPhase.FileTranscriptionStreaming &&
      !(
        this.status.phase ===
          FileTranscriptionPhase.FileTranscriptionProcessing &&
        this.status.streaming
      )
    )
      return "ignored";
    if (delta.revision <= this.status.transcriptRevision) return "ignored";
    if (delta.revision !== this.status.transcriptRevision + 1) return "gap";
    this.#fileStatusRevision++;
    this.status = {
      ...this.status,
      transcript: `${this.status.transcript ?? ""}${delta.text}`,
      transcriptRevision: delta.revision,
      message: "Receiving transcript",
    };
    return "applied";
  }

  async refresh() {
    const request = ++this.#fileStatusRequest;
    const statusRevision = this.#fileStatusRevision;
    try {
      const snapshot = await this.#service.CurrentFileTranscription();
      if (
        request === this.#fileStatusRequest &&
        statusRevision === this.#fileStatusRevision
      )
        this.applyStatus(snapshot);
    } catch (cause) {
      if (request === this.#fileStatusRequest) this.#messages.fail(cause);
    }
  }

  async chooseAudioFile() {
    if (this.selectionBusy || this.status.canCancel) return;
    this.#messages.clear();
    this.choosing = true;
    const revision = this.#fileStatusRevision;
    try {
      const status = await this.#service.ChooseAudioFile();
      // The picker response can arrive after its selection event or a newer
      // capability/progress update. Only a newer generation may supersede those.
      if (
        status.generation > this.status.generation ||
        revision === this.#fileStatusRevision
      )
        this.applyStatus(status);
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.choosing = false;
    }
  }

  async startFileTranscription() {
    if (this.selectionBusy || !this.status.canStart) return;
    this.starting = true;
    this.#messages.clear();
    try {
      await this.#service.StartFileTranscription(this.streamingEnabled);
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.starting = false;
    }
  }

  async cancelFileTranscription() {
    if (this.cancelling || !this.status.canCancel) return;
    this.cancelling = true;
    this.#messages.clear();
    try {
      await this.#service.CancelFileTranscription();
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.cancelling = false;
    }
  }

  async tryFileStreamingAgain() {
    if (this.selectionBusy || this.status.canCancel) return;
    this.resettingStreaming = true;
    this.#messages.clear();
    try {
      await this.#service.TryFileStreamingAgain();
      this.streamingPreferred = true;
      this.#messages.announce(
        "Text updates selected. Start transcription when you’re ready.",
      );
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.resettingStreaming = false;
    }
  }

  async copyFileTranscript(): Promise<boolean> {
    this.#messages.clear();
    try {
      await this.#service.CopyFileTranscript();
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    }
  }

  async clearAudioFile() {
    if (this.selectionBusy || this.status.canCancel) return;
    this.clearing = true;
    this.#messages.clear();
    try {
      await this.#service.ClearAudioFile();
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.clearing = false;
    }
  }

  acknowledgeHistory() {
    if (this.status.phase === FileTranscriptionPhase.FileTranscriptionCompleted)
      this.historyGeneration = this.status.generation;
  }
}
