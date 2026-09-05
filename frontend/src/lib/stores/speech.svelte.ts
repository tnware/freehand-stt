import { TTSPhase, type TTSStatus, type HistoryTextVersion } from "$lib/state";
import type { SessionMessages } from "./messages.svelte";
import type * as TtsBindings from "$bindings/tts/service";
export type SpeechStateService = Pick<
  typeof TtsBindings,
  | "CurrentStatus"
  | "PlayHistoryEntry"
  | "PlayFileTranscript"
  | "PreviewVoice"
  | "SpeakText"
  | "Pause"
  | "Resume"
  | "Restart"
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
  #ttsStatusRevision = 0;

  applyStatus(status: TTSStatus) {
    if (status.generation < this.status.generation) return;
    this.#ttsStatusRevision++;
    this.status = status;
    if (status.phase === TTSPhase.Failed && status.message)
      this.#messages.reportFailure(status.message);
  }

  async listenHistoryEntry(id: number, version: HistoryTextVersion) {
    this.#messages.clear();
    try {
      await this.#service.PlayHistoryEntry(id, version);
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async listenFileTranscript() {
    this.#messages.clear();
    try {
      await this.#service.PlayFileTranscript();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async previewVoice() {
    if (this.previewing) return;
    this.previewing = true;
    this.#messages.clear();
    try {
      await this.#service.PreviewVoice();
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.previewing = false;
    }
  }

  async speakText(text: string) {
    this.#messages.clear();
    try {
      await this.#service.SpeakText(text);
    } catch (cause) {
      this.#messages.fail(cause);
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

  async stopTTS() {
    try {
      await this.#service.Stop();
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async saveTTSAudio() {
    this.#messages.clear();
    try {
      if (await this.#service.SaveAudio()) {
        this.#messages.announce("Generated speech saved as a WAV file.");
      }
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async clearTTSAudio() {
    this.#messages.clear();
    try {
      await this.#service.ClearAudio();
      this.#messages.announce("Generated speech cleared from memory.");
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }

  async load() {
    const revision = this.#ttsStatusRevision;
    const status = await this.#service.CurrentStatus();
    if (revision === this.#ttsStatusRevision) this.status = status;
  }
}
