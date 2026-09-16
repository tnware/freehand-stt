import * as ManagedRuntimeService from "$bindings/managedruntime/manager";
import {
  ManagedRuntimeState,
  type ManagedRuntimeService as RuntimeService,
} from "./managed-runtime.svelte";
import * as ConnectionService from "$bindings/connection/service";
import * as DictationService from "$bindings/dictation/service";
import * as FileTranscriptionService from "$bindings/filetranscription/service";
import * as HistoryService from "$bindings/history/service";
import * as InputService from "$bindings/input/service";
import * as SettingsService from "$bindings/settings/service";
import * as TTSService from "$bindings/tts/service";
import * as ResourcesService from "$bindings/resources/service";
import { ResourceState, type ResourceService } from "./resources.svelte";
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
  resources?: ResourceService;
  runtime?: RuntimeService;
  dictation: DictationStateService;
  files: FileTranscriptionStateService;
  speech: SpeechStateService;
  history: HistoryStateService;
}

const services: SessionServices = {
  resources: ResourcesService,
  runtime: ManagedRuntimeService,
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
  readonly resources: ResourceState;
  readonly messages = new SessionMessages();
  readonly runtime: ManagedRuntimeState;
  readonly editor: SettingsEditor;
  readonly dictation: DictationState;
  readonly files: FileTranscriptionState;
  readonly speech: SpeechState;
  readonly history: HistoryState;
  #disposed = false;

  constructor(bindings: SessionServices = services) {
    this.resources = new ResourceState(bindings.resources);
    this.dictation = new DictationState(bindings.dictation, this.messages);
    this.files = new FileTranscriptionState(bindings.files, this.messages);
    this.speech = new SpeechState(bindings.speech, this.messages);
    this.history = new HistoryState(bindings.history, this.messages, () =>
      this.files.acknowledgeHistory(),
    );
    this.editor = new SettingsEditor(
      bindings,
      this.messages,
      () => this.history.refresh(),
      {
        statusFor: (id) => this.runtime?.statusFor(id),
        pendingFor: (id) => this.runtime?.pendingFor(id) ?? "",
      },
    );
    this.runtime = new ManagedRuntimeState(
      bindings.runtime,
      // Metadata probes do not change settings or own a runtime process. Keep
      // mutation locks explicit so a slow probe cannot reject Start or Stop.
      () =>
        !this.editor.dirty &&
        !this.editor.saving &&
        !this.editor.setupCompleting &&
        !this.editor.configurationRetrying &&
        !this.editor.configurationResetting &&
        !this.editor.quickSettingsPending.length,
      () => this.editor.refresh(),
    );
  }

  get busy() {
    return this.editor.busy || this.speech.previewing;
  }

  async load() {
    if (this.#disposed) return;
    this.messages.clear();
    try {
      await Promise.all([this.dictation.load(), this.runtime.load()]);
      if (this.#disposed) return;
      await this.speech.load();
      if (this.#disposed) return;
      await this.files.refresh();
      if (this.#disposed) return;
      if (!(await this.editor.load()) || this.#disposed) return;
      await this.editor.refreshDevices();
      if (this.#disposed) return;
      await this.history.refresh();
    } catch (cause) {
      if (!this.#disposed) this.messages.fail(cause);
    }
  }

  dispose() {
    if (this.#disposed) return;
    this.#disposed = true;
    this.resources.dispose();
    this.runtime.dispose();
    this.speech.draft = "";
    this.editor.dispose();
    this.messages.dispose();
  }
}

export const session = new Session();

// Component HMR may replace App without replacing this shared module. Keep the
// default session usable until the WebView leaves or this module is replaced.
if (typeof window !== "undefined") {
  const dispose = () => {
    window.removeEventListener("pagehide", pageHide);
    session.dispose();
  };
  const pageHide = (event: PageTransitionEvent) => {
    // A cached page resumes with the same renderer state on Back/Forward.
    if (!event.persisted) dispose();
  };
  window.addEventListener("pagehide", pageHide);
  import.meta.hot?.dispose(dispose);
}
