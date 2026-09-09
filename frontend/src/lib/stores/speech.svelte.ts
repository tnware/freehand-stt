import { TTSPhase, TTSSource, type TTSStatus, type HistoryTextVersion } from "$lib/state";
import type { SessionMessages } from "./messages.svelte";
import type * as TtsBindings from "$bindings/tts/service";
import type { Settings } from "$lib/state";
export type SpeechStateService = Pick<
  typeof TtsBindings,
  | "CurrentStatus"
  | "PlayHistoryEntry"
  | "PlayFileTranscript"
  | "PlayVoiceTranscript"
  | "PreviewVoice"
  | "SpeakText"
  | "Pause"
  | "Resume"
  | "Restart"
  | "Seek"
  | "Stop"
  | "SaveAudio"
  | "ClearAudio"
>;

const IDLE_TTS: TTSStatus = {
  generation: 0,
  phase: TTSPhase.Idle,
  positionMilliseconds: 0,
  durationMilliseconds: 0,
  canPause: false,
  canResume: false,
  canRestart: false,
  canSeek: false,
  canStop: false,
  canSave: false,
  canClear: false,
};

/** Projects backend-owned speech playback and issues playback commands. */
export class SpeechState {
  readonly #service: SpeechStateService;
  readonly #messages: SessionMessages;
  constructor(service: SpeechStateService, messages: SessionMessages) {
    this.#service = service;
    this.#messages = messages;
  }
  status = $state<TTSStatus>(IDLE_TTS);
  previewing = $state(false);
  submitting = $state(false);
  seeking = $state(false);
  saving = $state(false);
  listening = $state<Pick<TTSStatus, "source" | "historyID"> | null>(null);
  get canListen() {
    return (
      !this.listening &&
      !this.submitting &&
      !this.previewing &&
      this.status.phase !== TTSPhase.Generating
    );
  }
  // Unsent work belongs to the WebView session, never browser or disk storage.
  draft = $state("");
  #ttsStatusRevision = 0;

  applyStatus(status: TTSStatus) {
    if (status.generation < this.status.generation) return;
    const previous = this.status;
    this.#ttsStatusRevision++;
    this.status = status;
    if (status.phase === TTSPhase.Failed && status.message) {
      if (
        previous.generation !== status.generation ||
        previous.phase !== status.phase ||
        previous.message !== status.message
      )
        this.#messages.reportSpeechFailure(status.message, status.generation);
    } else if (this.#messages.isSpeechFailure(previous.generation)) {
      this.#messages.dismissError();
    }
  }

  async listenHistoryEntry(id: number, version: HistoryTextVersion) {
    await this.#listen({ source: TTSSource.SourceHistory, historyID: id }, () =>
      this.#service.PlayHistoryEntry(id, version),
    );
  }

  async listenFileTranscript() {
    await this.#listen({ source: TTSSource.SourceFile }, () => this.#service.PlayFileTranscript());
  }

  async listenVoiceTranscript(generation: number) {
    await this.#listen({ source: TTSSource.SourceVoice }, () =>
      this.#service.PlayVoiceTranscript(generation),
    );
  }

  async #listen(target: Pick<TTSStatus, "source" | "historyID">, start: () => Promise<void>) {
    if (!this.canListen) return;
    this.listening = target;
    this.#messages.clear();
    try {
      await start();
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.listening = null;
    }
  }

  async previewVoice(settings: Settings) {
    if (this.previewing || this.listening) return;
    this.previewing = true;
    this.#messages.clear();
    try {
      const draft = settings.textToSpeech;
      await this.#service.PreviewVoice({
        connectionID: settings.savedConnections.selected?.speech ?? "",
        enabled: draft.enabled,
        modelProfile: draft.modelProfile,
        model: draft.model,
        voice: draft.voice,
        speed: draft.speed,
        timeoutSeconds: draft.timeoutSeconds,
        options: { ...draft.options },
      });
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.previewing = false;
    }
  }

  async speakText(text: string) {
    if (this.submitting || this.listening) return;
    this.submitting = true;
    this.#messages.clear();
    try {
      await this.#service.SpeakText(text);
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.submitting = false;
    }
  }

  async pauseTTS() {
    try {
      await this.#service.Pause();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async resumeTTS() {
    try {
      await this.#service.Resume();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async restartTTS() {
    try {
      await this.#service.Restart();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async seekTTS(request: Parameters<SpeechStateService["Seek"]>[0]) {
    if (this.seeking || request.generation !== this.status.generation || !this.status.canSeek)
      return;
    this.seeking = true;
    const revision = this.#ttsStatusRevision;
    try {
      const status = await this.#service.Seek(request);
      if (revision === this.#ttsStatusRevision) this.applyStatus(status);
    } catch (cause) {
      if (request.generation === this.status.generation) this.#messages.fail(cause);
    } finally {
      this.seeking = false;
    }
  }

  async stopTTS() {
    try {
      await this.#service.Stop();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async saveTTSAudio() {
    if (this.saving) return;
    this.saving = true;
    this.#messages.clear();
    try {
      if (await this.#service.SaveAudio()) {
        this.#messages.announce("Generated speech saved as a WAV file.");
      }
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.saving = false;
    }
  }

  async clearTTSAudio() {
    this.#messages.clear();
    try {
      await this.#service.ClearAudio();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async load() {
    const revision = this.#ttsStatusRevision;
    const status = await this.#service.CurrentStatus();
    if (revision === this.#ttsStatusRevision) this.applyStatus(status);
  }
}
