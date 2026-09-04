import * as ConnectionService from "$bindings/connection/service";
import * as DictationService from "$bindings/dictation/service";
import * as FileTranscriptionService from "$bindings/filetranscription/service";
import * as HistoryService from "$bindings/history/service";
import * as InputService from "$bindings/input/service";
import * as SettingsService from "$bindings/settings/service";
import * as TTSService from "$bindings/tts/service";
import {
  FileTranscriptionPhase,
  RecordingMode,
  State,
  type ConnectionResult,
  type Device,
  type HistoryEntry,
  type HistoryTextVersion,
  type FileTranscriptionDelta,
  type FileTranscriptionStatus,
  type ProfileDescriptor,
  type Settings,
  type Status,
  TTSPhase,
  type TTSStatus,
} from "$lib/state";
import { appearanceRestartRequired } from "$lib/appearance";
import {
  SYSTEM_DEFAULT_MICROPHONE,
  microphoneChoiceFor,
  microphoneIDFor,
  usableDevices,
} from "$lib/utils/microphone";

// The generated modules remain the direct source of every method and wire
// shape. This object only supplies Session's default dependency as the
// composition root for the renderer; it does not translate any arguments or
// results.
const Service = {
  ...ConnectionService,
  ...DictationService,
  ...FileTranscriptionService,
  ...HistoryService,
  ...InputService,
  ...SettingsService,
  CurrentTTSStatus: TTSService.CurrentStatus,
  PlayHistoryEntry: TTSService.PlayHistoryEntry,
  PlayFileTranscript: TTSService.PlayFileTranscript,
  PreviewVoice: TTSService.PreviewVoice,
  SpeakText: TTSService.SpeakText,
  PauseTTS: TTSService.Pause,
  ResumeTTS: TTSService.Resume,
  RestartTTS: TTSService.Restart,
  StopTTS: TTSService.Stop,
  SaveTTSAudio: TTSService.SaveAudio,
  ClearTTSAudio: TTSService.ClearAudio,
};

const IDLE: Status = {
  state: State.Idle,
  generation: 0,
  canCancel: false,
  canCopy: false,
};

const EMPTY_FILE: FileTranscriptionStatus = {
  generation: 0,
  phase: FileTranscriptionPhase.FileTranscriptionEmpty,
  streaming: false,
  buffered: false,
  streamingUnavailable: false,
  transcriptRevision: 0,
  canStart: false,
  canCancel: false,
  canCopy: false,
};

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

export type SessionService = Pick<
  typeof Service,
  | "CurrentStatus"
  | "GetSettings"
  | "GetPostProcessingProfiles"
  | "RetryConfiguration"
  | "ResetConfiguration"
  | "SaveSettings"
  | "ListMicrophones"
  | "TestConnection"
  | "TestPostProcessingConnection"
  | "TestTextToSpeechConnection"
  | "StartRecording"
  | "StopRecording"
  | "Cancel"
  | "CopyPending"
  | "TranscriptHistory"
  | "CopyHistoryEntry"
  | "CopyHistoryEntryVersion"
  | "DeleteHistoryEntry"
  | "ClearHistory"
  | "CurrentFileTranscription"
  | "ChooseAudioFile"
  | "ClearAudioFile"
  | "StartFileTranscription"
  | "TryFileStreamingAgain"
  | "CancelFileTranscription"
  | "CopyFileTranscript"
  | "PlayHistoryEntry"
  | "PlayFileTranscript"
  | "PreviewVoice"
  | "SpeakText"
  | "CurrentTTSStatus"
  | "PauseTTS"
  | "ResumeTTS"
  | "RestartTTS"
  | "StopTTS"
  | "SaveTTSAudio"
  | "ClearTTSAudio"
>;

/**
 * How long a success notice stays up. Confirmations are transient and nothing
 * is lost when one goes away; errors stay until they are dismissed or the next
 * action replaces them.
 */
const NOTICE_MS = 6000;

export type QuickSettingsPatch = Partial<
  Pick<
    Settings,
    | "microphoneID"
    | "vadEnabled"
    | "silenceTrimming"
    | "autoStopEnabled"
    | "silenceSplitting"
    | "maxDurationSeconds"
    | "autoInsert"
    | "historyEnabled"
    | "overlayEnabled"
  >
> & {
  baseURL?: string;
  model?: string;
  postProcessing?: Partial<
    Pick<
      Settings["postProcessing"],
      | "enabled"
      | "baseURL"
      | "model"
      | "preset"
      | "styling"
      | "structure"
      | "context"
    >
  >;
};

export type QuickSettingsField =
  | "microphone"
  | "vad-enabled"
  | "silence-splitting"
  | "delivery"
  | "history-enabled"
  | "overlay-enabled"
  | "stt-endpoint"
  | "stt-model"
  | "processing-enabled"
  | "processing-endpoint"
  | "processing-model"
  | "processing-profile"
  | "processing-controls";

/** Keeps the editable draft independent from the backend-confirmed snapshot. */
const copySettings = (settings: Settings): Settings => ({
  ...settings,
  headers:
    settings.headers == null ? settings.headers : { ...settings.headers },
  postProcessing: { ...settings.postProcessing },
  textToSpeech: { ...settings.textToSpeech },
  microphoneID: settings.microphoneID ?? "",
});

const settingsMatch = (
  left: Settings | null,
  right: Settings | null,
): boolean =>
  left === null || right === null
    ? left === right
    : JSON.stringify(left) === JSON.stringify(right);

/**
 * Session owns every piece of mutable frontend state and the calls that change
 * it. Components stay presentational and this stays testable: pass a stub of
 * the generated service surface and drive it without a Wails runtime.
 */
export class Session {
  settings = $state<Settings | null>(null);
  appliedSettings = $state<Settings | null>(null);
  history = $state<HistoryEntry[]>([]);
  fileHistoryGeneration = $state(0);
  connection = $state<ConnectionResult | null>(null);
  sttConnectionChecked = $state(false);
  sttConnectionStale = $state(false);
  processingConnection = $state<ConnectionResult | null>(null);
  processingConnectionStale = $state(false);
  ttsConnection = $state<ConnectionResult | null>(null);
  ttsConnectionStale = $state(false);
  processingProfiles = $state<ProfileDescriptor[]>([]);
  devices = $state<Device[]>([]);
  status = $state<Status>(IDLE);
  fileStatus = $state<FileTranscriptionStatus>(EMPTY_FILE);
  ttsStatus = $state<TTSStatus>(IDLE_TTS);
  microphoneChoice = $state(SYSTEM_DEFAULT_MICROPHONE);
  apiKey = $state("");
  clearKey = $state(false);
  processingAPIKey = $state("");
  clearProcessingKey = $state(false);
  ttsAPIKey = $state("");
  clearTTSKey = $state(false);
  settingsSaving = $state(false);
  setupCompleting = $state(false);
  sttConnectionTesting = $state(false);
  processingConnectionTesting = $state(false);
  ttsConnectionTesting = $state(false);
  ttsPreviewing = $state(false);
  configurationRetrying = $state(false);
  configurationResetting = $state(false);
  quickSettingsPending = $state<QuickSettingsField[]>([]);
  quickSettingsSaved = $state<QuickSettingsField | null>(null);
  busy = $derived(
    this.settingsSaving ||
      this.setupCompleting ||
      this.sttConnectionTesting ||
      this.processingConnectionTesting ||
      this.ttsConnectionTesting ||
      this.ttsPreviewing ||
      this.configurationRetrying ||
      this.configurationResetting ||
      this.quickSettingsPending.length > 0,
  );
  devicesBusy = $state(false);
  fileChoosing = $state(false);
  info = $state("");
  notice = $state("");
  error = $state("");

  readonly #service: SessionService;
  #statusRevision = 0;
  #fileStatusRevision = 0;
  #ttsStatusRevision = 0;
  #fileStatusRequest = 0;
  #sttConnectionRevision = 0;
  #historyRequest = 0;
  #infoTimer: ReturnType<typeof setTimeout> | undefined;
  #noticeTimer: ReturnType<typeof setTimeout> | undefined;
  #quickSettingsSavedTimer: ReturnType<typeof setTimeout> | undefined;
  #quickSettingsQueue: Promise<void> = Promise.resolve();

  constructor(service: SessionService = Service) {
    this.#service = service;
  }

  clearMessages() {
    this.error = "";
    this.dismissInfo();
    this.dismissNotice();
  }

  dismissInfo() {
    clearTimeout(this.#infoTimer);
    this.#infoTimer = undefined;
    this.info = "";
  }

  dismissNotice() {
    clearTimeout(this.#noticeTimer);
    this.#noticeTimer = undefined;
    this.notice = "";
  }

  dismissError() {
    this.error = "";
  }

  /** Shows an actionable renderer failure in the existing visible channel. */
  reportFailure(message: string) {
    this.dismissNotice();
    this.error = message;
  }

  /** Shows a transient explanation of system behaviour, separate from success. */
  reportInfo(message: string) {
    clearTimeout(this.#infoTimer);
    this.info = message;
    this.#infoTimer = setTimeout(() => {
      this.info = "";
      this.#infoTimer = undefined;
    }, NOTICE_MS);
  }

  /** Shows a confirmation that takes itself down again. */
  #announce(message: string) {
    clearTimeout(this.#noticeTimer);
    this.notice = message;
    this.#noticeTimer = setTimeout(() => {
      this.notice = "";
      this.#noticeTimer = undefined;
    }, NOTICE_MS);
  }

  #announceSettingsSaved(
    settings: Settings,
    message = "Settings saved and active.",
  ) {
    this.#announce(
      appearanceRestartRequired(settings)
        ? `${message} Restart the app to apply the appearance change.`
        : message,
    );
  }

  /** Applies a settings payload confirmed by Go and starts a fresh draft. */
  #adopt(settings: Settings) {
    this.appliedSettings = copySettings(settings);
    this.settings = copySettings(this.appliedSettings);
    this.microphoneChoice = microphoneChoiceFor(
      this.appliedSettings.microphoneID,
    );
  }

  /** Records a caught value as the visible error. */
  #fail(cause: unknown) {
    this.error = String(cause);
  }

  #invalidateSTTConnection() {
    this.sttConnectionStale =
      this.sttConnectionStale ||
      this.connection !== null ||
      this.sttConnectionChecked;
    this.#sttConnectionRevision++;
    this.connection = null;
    this.sttConnectionChecked = false;
  }

  #invalidateProcessingConnection() {
    this.processingConnectionStale =
      this.processingConnectionStale || this.processingConnection !== null;
    this.processingConnection = null;
  }

  #invalidateTTSConnection() {
    this.ttsConnectionStale =
      this.ttsConnectionStale || this.ttsConnection !== null;
    this.ttsConnection = null;
  }

  applyStatus(status: Status) {
    this.#statusRevision++;
    this.status = status;
  }

  applyFileStatus(status: FileTranscriptionStatus) {
    if (status.generation < this.fileStatus.generation) return;
    if (
      status.generation === this.fileStatus.generation &&
      status.transcriptRevision < this.fileStatus.transcriptRevision
    )
      return;
    this.#fileStatusRevision++;
    this.fileStatus = status;
  }

  applyTTSStatus(status: TTSStatus) {
    if (status.generation < this.ttsStatus.generation) return;
    this.#ttsStatusRevision++;
    this.ttsStatus = status;
    if (status.phase === TTSPhase.Failed && status.message)
      this.reportFailure(status.message);
  }

  applyFileDelta(delta: FileTranscriptionDelta): "applied" | "ignored" | "gap" {
    if (delta.generation !== this.fileStatus.generation || !delta.text)
      return "ignored";
    if (
      this.fileStatus.phase !==
        FileTranscriptionPhase.FileTranscriptionStreaming &&
      !(
        this.fileStatus.phase ===
          FileTranscriptionPhase.FileTranscriptionProcessing &&
        this.fileStatus.streaming
      )
    )
      return "ignored";
    if (delta.revision <= this.fileStatus.transcriptRevision) return "ignored";
    if (delta.revision !== this.fileStatus.transcriptRevision + 1) return "gap";
    this.#fileStatusRevision++;
    this.fileStatus = {
      ...this.fileStatus,
      transcript: `${this.fileStatus.transcript ?? ""}${delta.text}`,
      transcriptRevision: delta.revision,
      message: "Receiving transcript",
    };
    return "applied";
  }

  async refreshFileStatus() {
    const request = ++this.#fileStatusRequest;
    const statusRevision = this.#fileStatusRevision;
    try {
      const snapshot = await this.#service.CurrentFileTranscription();
      if (
        request === this.#fileStatusRequest &&
        statusRevision === this.#fileStatusRevision
      )
        this.applyFileStatus(snapshot);
    } catch (cause) {
      if (request === this.#fileStatusRequest) this.#fail(cause);
    }
  }

  /**
   * Synchronizes a renderer with the backend snapshot committed by another
   * window. An active settings draft is never overwritten silently; the
   * settings window owns that draft until the user saves or discards it.
   */
  applySettingsSnapshot(settings: Settings): boolean {
    if (this.settingsDirty && !this.settingsSaving) {
      this.reportInfo(
        "Settings changed in another window. Save or discard this draft, then reopen Settings to load the latest values.",
      );
      return false;
    }
    const previous = this.appliedSettings;
    this.#adopt(settings);
    if (
      previous &&
      (previous.baseURL !== settings.baseURL ||
        previous.model !== settings.model ||
        previous.allowInsecureHTTP !== settings.allowInsecureHTTP ||
        previous.authenticationMode !== settings.authenticationMode ||
        previous.healthPath !== settings.healthPath ||
        JSON.stringify(previous.headers) !== JSON.stringify(settings.headers))
    ) {
      this.#invalidateSTTConnection();
    }
    if (
      previous &&
      (previous.postProcessing.baseURL !== settings.postProcessing.baseURL ||
        previous.postProcessing.model !== settings.postProcessing.model)
    ) {
      this.#invalidateProcessingConnection();
    }
    if (
      previous &&
      (previous.textToSpeech.baseURL !== settings.textToSpeech.baseURL ||
        previous.textToSpeech.model !== settings.textToSpeech.model ||
        previous.textToSpeech.allowInsecureHTTP !==
          settings.textToSpeech.allowInsecureHTTP ||
        previous.textToSpeech.authenticationMode !==
          settings.textToSpeech.authenticationMode)
    ) {
      this.#invalidateTTSConnection();
    }
    return true;
  }

  clearCredentialDraft() {
    this.apiKey = "";
    this.clearKey = false;
    this.processingAPIKey = "";
    this.clearProcessingKey = false;
    this.ttsAPIKey = "";
    this.clearTTSKey = false;
  }

  get settingsDirty(): boolean {
    return (
      !settingsMatch(this.settings, this.appliedSettings) ||
      this.apiKey !== "" ||
      this.clearKey ||
      this.processingAPIKey !== "" ||
      this.clearProcessingKey ||
      this.ttsAPIKey !== "" ||
      this.clearTTSKey
    );
  }

  discardSettingsDraft() {
    if (this.appliedSettings) {
      if (
        this.apiKey !== "" ||
        this.clearKey ||
        (this.settings &&
          (this.settings.baseURL !== this.appliedSettings.baseURL ||
            this.settings.model !== this.appliedSettings.model))
      ) {
        this.#invalidateSTTConnection();
      }
      if (
        this.processingAPIKey !== "" ||
        this.clearProcessingKey ||
        (this.settings &&
          (this.settings.postProcessing.baseURL !==
            this.appliedSettings.postProcessing.baseURL ||
            this.settings.postProcessing.model !==
              this.appliedSettings.postProcessing.model))
      ) {
        this.#invalidateProcessingConnection();
      }
      if (
        this.ttsAPIKey !== "" ||
        this.clearTTSKey ||
        (this.settings &&
          (this.settings.textToSpeech.baseURL !==
            this.appliedSettings.textToSpeech.baseURL ||
            this.settings.textToSpeech.model !==
              this.appliedSettings.textToSpeech.model))
      ) {
        this.#invalidateTTSConnection();
      }
      this.settings = copySettings(this.appliedSettings);
      this.microphoneChoice = microphoneChoiceFor(
        this.appliedSettings.microphoneID,
      );
    }
    this.clearCredentialDraft();
    this.clearMessages();
  }

  isQuickSettingsPending(field: QuickSettingsField): boolean {
    return this.quickSettingsPending.includes(field);
  }

  #markQuickSettingsSaved(field: QuickSettingsField) {
    clearTimeout(this.#quickSettingsSavedTimer);
    this.quickSettingsSaved = field;
    this.#quickSettingsSavedTimer = setTimeout(() => {
      this.quickSettingsSaved = null;
      this.#quickSettingsSavedTimer = undefined;
    }, 1800);
  }

  chooseMicrophone(choice: string) {
    this.microphoneChoice = choice;
    if (this.settings) this.settings.microphoneID = microphoneIDFor(choice);
  }

  async refreshDevices() {
    if (this.devicesBusy) return;
    this.dismissError();
    this.devicesBusy = true;
    try {
      this.devices = usableDevices(await this.#service.ListMicrophones());
    } catch (cause) {
      this.#fail(cause);
    } finally {
      this.devicesBusy = false;
    }
  }

  async loadSettings(): Promise<boolean> {
    try {
      const [settings, processingProfiles] = await Promise.all([
        this.#service.GetSettings(),
        this.#service.GetPostProcessingProfiles(),
      ]);
      this.#adopt(settings);
      this.processingProfiles = processingProfiles ?? [];
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    }
  }

  async load() {
    this.clearMessages();
    try {
      const statusRevision = this.#statusRevision;
      const initialStatus = await this.#service.CurrentStatus();
      if (this.#statusRevision === statusRevision) this.status = initialStatus;
      const ttsStatusRevision = this.#ttsStatusRevision;
      const initialTTSStatus = await this.#service.CurrentTTSStatus();
      if (this.#ttsStatusRevision === ttsStatusRevision)
        this.ttsStatus = initialTTSStatus;
      await this.refreshFileStatus();
      if (!(await this.loadSettings())) return;
      await this.refreshDevices();
      await this.refreshHistory();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async retryConfiguration(): Promise<boolean> {
    if (this.configurationRetrying || this.configurationResetting) return false;
    this.configurationRetrying = true;
    this.dismissError();
    try {
      const settings = await this.#service.RetryConfiguration();
      this.#adopt(settings);
      if (settings.configuration.recoveryRequired) return false;
      this.#announce("Saved settings loaded successfully.");
      await Promise.all([this.refreshDevices(), this.refreshHistory()]);
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    } finally {
      this.configurationRetrying = false;
    }
  }

  async resetConfiguration(): Promise<boolean> {
    if (this.configurationRetrying || this.configurationResetting) return false;
    this.configurationResetting = true;
    this.dismissError();
    try {
      const settings = await this.#service.ResetConfiguration();
      this.#adopt(settings);
      this.clearCredentialDraft();
      this.#invalidateSTTConnection();
      this.#invalidateProcessingConnection();
      this.#invalidateTTSConnection();
      this.#announce(
        "Settings reset. Review the setup before starting transcription.",
      );
      await Promise.all([this.refreshDevices(), this.refreshHistory()]);
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    } finally {
      this.configurationResetting = false;
    }
  }

  async save(): Promise<boolean> {
    if (!this.settings || this.settingsSaving) return false;
    this.settingsSaving = true;
    this.clearMessages();
    try {
      const previous = this.appliedSettings;
      const sttCredentialChanged = this.apiKey !== "" || this.clearKey;
      const processingCredentialChanged =
        this.processingAPIKey !== "" || this.clearProcessingKey;
      const ttsCredentialChanged = this.ttsAPIKey !== "" || this.clearTTSKey;
      const saved = await this.#service.SaveSettings({
        settings: this.settings,
        sttCredentialDraft: this.apiKey,
        clearSTTCredential: this.clearKey,
        postProcessingCredentialDraft: this.processingAPIKey,
        clearPostProcessingCredential: this.clearProcessingKey,
        textToSpeechCredentialDraft: this.ttsAPIKey,
        clearTextToSpeechCredential: this.clearTTSKey,
      });
      this.#adopt(saved);
      if (
        sttCredentialChanged ||
        (previous &&
          (previous.baseURL !== saved.baseURL ||
            previous.model !== saved.model ||
            previous.allowInsecureHTTP !== saved.allowInsecureHTTP ||
            previous.authenticationMode !== saved.authenticationMode ||
            previous.healthPath !== saved.healthPath ||
            JSON.stringify(previous.headers) !== JSON.stringify(saved.headers)))
      ) {
        this.#invalidateSTTConnection();
      }
      if (
        processingCredentialChanged ||
        (previous &&
          (previous.postProcessing.baseURL !== saved.postProcessing.baseURL ||
            previous.postProcessing.model !== saved.postProcessing.model))
      ) {
        this.#invalidateProcessingConnection();
      }
      if (
        ttsCredentialChanged ||
        (previous &&
          (previous.textToSpeech.baseURL !== saved.textToSpeech.baseURL ||
            previous.textToSpeech.model !== saved.textToSpeech.model ||
            previous.textToSpeech.allowInsecureHTTP !==
              saved.textToSpeech.allowInsecureHTTP ||
            previous.textToSpeech.authenticationMode !==
              saved.textToSpeech.authenticationMode))
      ) {
        this.#invalidateTTSConnection();
      }
      this.clearCredentialDraft();
      this.#announceSettingsSaved(saved);
      await this.refreshHistory();
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    } finally {
      this.settingsSaving = false;
    }
  }

  async completeSetup(): Promise<boolean> {
    if (!this.appliedSettings || this.setupCompleting) return false;
    this.setupCompleting = true;
    this.dismissError();
    try {
      const next = copySettings(this.appliedSettings);
      next.setupCompleted = true;
      const saved = await this.#service.SaveSettings({
        settings: next,
        sttCredentialDraft: "",
        clearSTTCredential: false,
        postProcessingCredentialDraft: "",
        clearPostProcessingCredential: false,
        textToSpeechCredentialDraft: "",
        clearTextToSpeechCredential: false,
      });
      this.#adopt(saved);
      this.#announce("Freehand is ready to use.");
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    } finally {
      this.setupCompleting = false;
    }
  }

  /**
   * Applies the narrow home-screen controls immediately. The update starts
   * from the backend-confirmed snapshot so a closed Settings window cannot
   * accidentally smuggle an unrelated unsaved draft into this save.
   */
  async updateQuickSettings(
    patch: QuickSettingsPatch,
    field: QuickSettingsField,
  ): Promise<boolean> {
    if (!this.appliedSettings || this.isQuickSettingsPending(field))
      return false;
    this.quickSettingsPending = [...this.quickSettingsPending, field];
    if (this.quickSettingsSaved === field) this.quickSettingsSaved = null;
    this.dismissError();

    let operationResult = false;
    const operation = this.#quickSettingsQueue.then(async () => {
      if (!this.appliedSettings) return;
      const next = copySettings(this.appliedSettings);
      if (patch.baseURL !== undefined) next.baseURL = patch.baseURL;
      if (patch.model !== undefined) next.model = patch.model;
      if (patch.microphoneID !== undefined)
        next.microphoneID = patch.microphoneID;
      if (patch.vadEnabled !== undefined) next.vadEnabled = patch.vadEnabled;
      if (patch.silenceTrimming !== undefined)
        next.silenceTrimming = patch.silenceTrimming;
      if (patch.autoStopEnabled !== undefined)
        next.autoStopEnabled = patch.autoStopEnabled;
      if (patch.silenceSplitting !== undefined) {
        next.silenceSplitting = patch.silenceSplitting;
      }
      if (patch.maxDurationSeconds !== undefined) {
        next.maxDurationSeconds = patch.maxDurationSeconds;
      }
      if (patch.autoInsert !== undefined) next.autoInsert = patch.autoInsert;
      if (patch.historyEnabled !== undefined)
        next.historyEnabled = patch.historyEnabled;
      if (patch.overlayEnabled !== undefined)
        next.overlayEnabled = patch.overlayEnabled;
      if (patch.postProcessing) {
        next.postProcessing = {
          ...next.postProcessing,
          ...patch.postProcessing,
        };
      }

      const saved = await this.#service.SaveSettings({
        settings: next,
        sttCredentialDraft: "",
        clearSTTCredential: false,
        postProcessingCredentialDraft: "",
        clearPostProcessingCredential: false,
        textToSpeechCredentialDraft: "",
        clearTextToSpeechCredential: false,
      });
      this.#adopt(saved);
      if (patch.baseURL !== undefined || patch.model !== undefined) {
        this.#invalidateSTTConnection();
      }
      if (
        patch.postProcessing?.baseURL !== undefined ||
        patch.postProcessing?.model !== undefined
      ) {
        this.#invalidateProcessingConnection();
      }
      this.#markQuickSettingsSaved(field);
      operationResult = true;
    });
    this.#quickSettingsQueue = operation.then(
      () => undefined,
      () => undefined,
    );
    try {
      await operation;
      return operationResult;
    } catch (cause) {
      this.#fail(cause);
      return false;
    } finally {
      this.quickSettingsPending = this.quickSettingsPending.filter(
        (pending) => pending !== field,
      );
    }
  }

  async testConnection(
    settings = this.settings,
    apiKey = this.apiKey,
    clearExistingMessages = true,
  ) {
    if (this.sttConnectionTesting || !settings) return;
    const revision = this.#sttConnectionRevision;
    this.sttConnectionTesting = true;
    if (clearExistingMessages) this.clearMessages();
    try {
      const result = await this.#service.TestConnection({
        baseURL: settings.baseURL,
        allowInsecureHTTP: settings.allowInsecureHTTP,
        authenticationMode: settings.authenticationMode,
        model: settings.model,
        healthPath: settings.healthPath ?? "",
        headers: settings.headers,
        credentialDraft: apiKey,
      });
      if (revision === this.#sttConnectionRevision) {
        this.connection = result;
        this.sttConnectionStale = false;
      }
    } catch (cause) {
      if (revision === this.#sttConnectionRevision) this.#fail(cause);
    } finally {
      this.sttConnectionTesting = false;
      if (revision === this.#sttConnectionRevision)
        this.sttConnectionChecked = true;
    }
  }

  async testPostProcessingConnection(
    settings = this.settings,
    apiKey = this.processingAPIKey,
  ) {
    if (this.processingConnectionTesting || !settings) return;
    this.processingConnectionTesting = true;
    this.processingConnection = null;
    this.clearMessages();
    try {
      this.processingConnection =
        await this.#service.TestPostProcessingConnection({
          baseURL: settings.postProcessing.baseURL,
          allowInsecureHTTP: settings.postProcessing.allowInsecureHTTP,
          model: settings.postProcessing.model,
          credentialDraft: apiKey,
        });
      this.processingConnectionStale = false;
    } catch (cause) {
      this.#fail(cause);
    } finally {
      this.processingConnectionTesting = false;
    }
  }

  async testTextToSpeechConnection(
    settings = this.settings,
    apiKey = this.ttsAPIKey,
  ) {
    if (this.ttsConnectionTesting || !settings) return;
    this.ttsConnectionTesting = true;
    this.ttsConnection = null;
    this.clearMessages();
    try {
      this.ttsConnection = await this.#service.TestTextToSpeechConnection({
        baseURL: settings.textToSpeech.baseURL,
        allowInsecureHTTP: settings.textToSpeech.allowInsecureHTTP,
        authenticationMode: settings.textToSpeech.authenticationMode,
        model: settings.textToSpeech.model,
        credentialDraft: apiKey,
      });
      this.ttsConnectionStale = false;
    } catch (cause) {
      this.#fail(cause);
    } finally {
      this.ttsConnectionTesting = false;
    }
  }

  async toggleRecording() {
    this.clearMessages();
    try {
      if (this.status.state === State.Recording)
        await this.#service.StopRecording();
      else await this.#service.StartRecording(RecordingMode.RecordingToggle);
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async cancel() {
    this.clearMessages();
    try {
      await this.#service.Cancel();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async copyPending(): Promise<boolean> {
    this.clearMessages();
    try {
      await this.#service.CopyPending();
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    }
  }

  async refreshHistory() {
    const request = ++this.#historyRequest;
    try {
      const history = (await this.#service.TranscriptHistory()) ?? [];
      if (request === this.#historyRequest) {
        this.history = history;
        if (
          this.fileStatus.phase ===
          FileTranscriptionPhase.FileTranscriptionCompleted
        ) {
          this.fileHistoryGeneration = this.fileStatus.generation;
        }
      }
    } catch (cause) {
      if (request === this.#historyRequest) this.#fail(cause);
    }
  }

  async copyHistoryEntry(id: number): Promise<boolean> {
    this.clearMessages();
    try {
      await this.#service.CopyHistoryEntry(id);
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    }
  }

  async copyHistoryEntryVersion(
    id: number,
    version: HistoryTextVersion,
  ): Promise<boolean> {
    this.clearMessages();
    try {
      await this.#service.CopyHistoryEntryVersion(id, version);
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    }
  }

  async deleteHistoryEntry(id: number): Promise<boolean> {
    this.clearMessages();
    try {
      await this.#service.DeleteHistoryEntry(id);
      this.#historyRequest++;
      this.history = this.history.filter((entry) => entry.id !== id);
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    }
  }

  async clearHistory() {
    this.clearMessages();
    try {
      await this.#service.ClearHistory();
      this.#historyRequest++;
      this.history = [];
      this.#announce("Transcript history cleared.");
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async chooseAudioFile() {
    if (this.fileChoosing || this.fileStatus.canCancel) return;
    this.clearMessages();
    this.fileChoosing = true;
    try {
      this.fileStatus = await this.#service.ChooseAudioFile();
    } catch (cause) {
      this.#fail(cause);
    } finally {
      this.fileChoosing = false;
    }
  }

  async startFileTranscription(stream: boolean) {
    this.clearMessages();
    try {
      await this.#service.StartFileTranscription(stream);
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async cancelFileTranscription() {
    this.clearMessages();
    try {
      await this.#service.CancelFileTranscription();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async tryFileStreamingAgain() {
    this.clearMessages();
    try {
      await this.#service.TryFileStreamingAgain();
      this.#announce(
        "Streaming can be tried again for this endpoint and model.",
      );
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async copyFileTranscript(): Promise<boolean> {
    this.clearMessages();
    try {
      await this.#service.CopyFileTranscript();
      return true;
    } catch (cause) {
      this.#fail(cause);
      return false;
    }
  }

  async clearAudioFile() {
    this.clearMessages();
    try {
      await this.#service.ClearAudioFile();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async listenHistoryEntry(id: number, version: HistoryTextVersion) {
    this.clearMessages();
    try {
      await this.#service.PlayHistoryEntry(id, version);
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async listenFileTranscript() {
    this.clearMessages();
    try {
      await this.#service.PlayFileTranscript();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async previewVoice() {
    if (this.ttsPreviewing) return;
    this.ttsPreviewing = true;
    this.clearMessages();
    try {
      await this.#service.PreviewVoice();
    } catch (cause) {
      this.#fail(cause);
    } finally {
      this.ttsPreviewing = false;
    }
  }

  async speakText(text: string) {
    this.clearMessages();
    try {
      await this.#service.SpeakText(text);
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async pauseTTS() {
    try {
      await this.#service.PauseTTS();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async resumeTTS() {
    try {
      await this.#service.ResumeTTS();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async restartTTS() {
    try {
      await this.#service.RestartTTS();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async stopTTS() {
    try {
      await this.#service.StopTTS();
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async saveTTSAudio() {
    this.clearMessages();
    try {
      if (await this.#service.SaveTTSAudio()) {
        this.#announce("Generated speech saved as a WAV file.");
      }
    } catch (cause) {
      this.#fail(cause);
    }
  }

  async clearTTSAudio() {
    this.clearMessages();
    try {
      await this.#service.ClearTTSAudio();
      this.#announce("Generated speech cleared from memory.");
    } catch (cause) {
      this.#fail(cause);
    }
  }
}

export const session = new Session();
