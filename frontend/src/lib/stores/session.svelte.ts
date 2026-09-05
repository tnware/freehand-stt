import * as ConnectionService from "$bindings/connection/service";
import * as DictationService from "$bindings/dictation/service";
import * as FileTranscriptionService from "$bindings/filetranscription/service";
import * as HistoryService from "$bindings/history/service";
import * as InputService from "$bindings/input/service";
import * as SettingsService from "$bindings/settings/service";
import * as TTSService from "$bindings/tts/service";
import { SessionMessages } from "./messages.svelte";
import { SettingsEditor, type SettingsEditorServices } from "./editor.svelte";
import { DictationState, type DictationStateService } from "./dictation.svelte";
import {
  FileTranscriptionState,
  type FileTranscriptionStateService,
} from "./files.svelte";
import { SpeechState, type SpeechStateService } from "./speech.svelte";
import { HistoryState, type HistoryStateService } from "./history.svelte";

export interface SessionServices extends SettingsEditorServices {
  dictation: DictationStateService;
  files: FileTranscriptionStateService;
  speech: SpeechStateService;
  history: HistoryStateService;
}

const services: SessionServices = {
  settings: SettingsService,
  input: InputService,
  connection: ConnectionService,
  dictation: DictationService,
  files: FileTranscriptionService,
  speech: TTSService,
  history: HistoryService,
};

/** Per-WebView composition. Feature state and commands live in their owners. */
export class Session {
  readonly messages = new SessionMessages();
  readonly editor: SettingsEditor;
  readonly dictation: DictationState;
  readonly files: FileTranscriptionState;
  readonly speech: SpeechState;
  readonly history: HistoryState;

  constructor(bindings: SessionServices = services) {
    this.dictation = new DictationState(bindings.dictation, this.messages);
    this.files = new FileTranscriptionState(bindings.files, this.messages);
    this.speech = new SpeechState(bindings.speech, this.messages);
    this.history = new HistoryState(bindings.history, this.messages, () =>
      this.files.acknowledgeHistory(),
    );
    this.editor = new SettingsEditor(bindings, this.messages, () =>
      this.history.refresh(),
    );
  }

  get busy() {
    return this.editor.busy || this.speech.previewing;
  }

  async load() {
    this.messages.clear();
    try {
      await this.dictation.load();
      await this.speech.load();
      await this.files.refresh();
      if (!(await this.editor.load())) return;
      await this.editor.refreshDevices();
      await this.history.refresh();
    } catch (cause) {
      this.messages.fail(cause);
    }
  }

  dispose() {
    this.editor.dispose();
    this.messages.dispose();
  }
}

export const session = new Session();
