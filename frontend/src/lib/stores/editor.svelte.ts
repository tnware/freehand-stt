import type {
  Settings,
  ConnectionResult,
  Device,
  ProfileDescriptor,
} from "$lib/state";
import { appearanceRestartRequired } from "$lib/appearance";
import {
  SYSTEM_DEFAULT_MICROPHONE,
  microphoneChoiceFor,
  microphoneIDFor,
  usableDevices,
} from "$lib/utils/microphone";
import type { SessionMessages } from "./messages.svelte";
import type * as SettingsBindings from "$bindings/settings/service";
import type * as InputBindings from "$bindings/input/service";
import type * as ConnectionBindings from "$bindings/connection/service";
export interface SettingsEditorServices {
  settings: Pick<
    typeof SettingsBindings,
    | "GetSettings"
    | "GetPostProcessingProfiles"
    | "RetryConfiguration"
    | "ResetConfiguration"
    | "SaveSettings"
  >;
  input: Pick<typeof InputBindings, "ListMicrophones">;
  connection: Pick<
    typeof ConnectionBindings,
    | "TestConnection"
    | "TestPostProcessingConnection"
    | "TestTextToSpeechConnection"
  >;
}

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

/** Owns one coherent settings/credential draft, probes, and serialized quick saves. */
export class SettingsEditor {
  readonly #service: SettingsEditorServices;
  readonly #messages: SessionMessages;
  readonly #refreshHistory: () => Promise<void>;
  constructor(
    service: SettingsEditorServices,
    messages: SessionMessages,
    refreshHistory: () => Promise<void>,
  ) {
    this.#service = service;
    this.#messages = messages;
    this.#refreshHistory = refreshHistory;
  }
  draft = $state<Settings | null>(null);
  applied = $state<Settings | null>(null);
  connection = $state<ConnectionResult | null>(null);
  sttConnectionChecked = $state(false);
  sttConnectionStale = $state(false);
  processingConnection = $state<ConnectionResult | null>(null);
  processingConnectionStale = $state(false);
  ttsConnection = $state<ConnectionResult | null>(null);
  ttsConnectionStale = $state(false);
  processingProfiles = $state<ProfileDescriptor[]>([]);
  devices = $state<Device[]>([]);
  microphoneChoice = $state(SYSTEM_DEFAULT_MICROPHONE);
  apiKey = $state("");
  clearKey = $state(false);
  processingAPIKey = $state("");
  clearProcessingKey = $state(false);
  ttsAPIKey = $state("");
  clearTTSKey = $state(false);
  saving = $state(false);
  setupCompleting = $state(false);
  sttConnectionTesting = $state(false);
  processingConnectionTesting = $state(false);
  ttsConnectionTesting = $state(false);
  configurationRetrying = $state(false);
  configurationResetting = $state(false);
  quickSettingsPending = $state<QuickSettingsField[]>([]);
  quickSettingsSaved = $state<QuickSettingsField | null>(null);
  devicesBusy = $state(false);
  #sttConnectionRevision = 0;
  #quickSettingsSavedTimer: ReturnType<typeof setTimeout> | undefined;
  #quickSettingsQueue: Promise<void> = Promise.resolve();

  #announceSettingsSaved(
    settings: Settings,
    message = "Settings saved and active.",
  ) {
    this.#messages.announce(
      appearanceRestartRequired(settings)
        ? `${message} Restart the app to apply the appearance change.`
        : message,
    );
  }

  /** Applies a settings payload confirmed by Go and starts a fresh draft. */
  #adopt(settings: Settings) {
    this.applied = copySettings(settings);
    this.draft = copySettings(this.applied);
    this.microphoneChoice = microphoneChoiceFor(this.applied.microphoneID);
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

  /**
   * Synchronizes a renderer with the backend snapshot committed by another
   * window. An active settings draft is never overwritten silently; the
   * settings window owns that draft until the user saves or discards it.
   */
  applySettingsSnapshot(settings: Settings): boolean {
    if (this.dirty && !this.saving) {
      this.#messages.reportInfo(
        "Settings changed in another window. Save or discard this draft, then reopen Settings to load the latest values.",
      );
      return false;
    }
    const previous = this.applied;
    this.#adopt(settings);
    if (
      previous &&
      (previous.baseURL !== settings.baseURL ||
        previous.compatibilityProfile !== settings.compatibilityProfile ||
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
        previous.postProcessing.compatibilityProfile !== settings.postProcessing.compatibilityProfile ||
        previous.postProcessing.model !== settings.postProcessing.model)
    ) {
      this.#invalidateProcessingConnection();
    }
    if (
      previous &&
      (previous.textToSpeech.baseURL !== settings.textToSpeech.baseURL ||
        previous.textToSpeech.compatibilityProfile !== settings.textToSpeech.compatibilityProfile ||
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

  get dirty(): boolean {
    return (
      !settingsMatch(this.draft, this.applied) ||
      this.apiKey !== "" ||
      this.clearKey ||
      this.processingAPIKey !== "" ||
      this.clearProcessingKey ||
      this.ttsAPIKey !== "" ||
      this.clearTTSKey
    );
  }

  discardSettingsDraft() {
    if (this.applied) {
      if (
        this.apiKey !== "" ||
        this.clearKey ||
        (this.draft &&
          (this.draft.baseURL !== this.applied.baseURL ||
            this.draft.model !== this.applied.model))
      ) {
        this.#invalidateSTTConnection();
      }
      if (
        this.processingAPIKey !== "" ||
        this.clearProcessingKey ||
        (this.draft &&
          (this.draft.postProcessing.baseURL !==
            this.applied.postProcessing.baseURL ||
            this.draft.postProcessing.model !==
              this.applied.postProcessing.model))
      ) {
        this.#invalidateProcessingConnection();
      }
      if (
        this.ttsAPIKey !== "" ||
        this.clearTTSKey ||
        (this.draft &&
          (this.draft.textToSpeech.baseURL !==
            this.applied.textToSpeech.baseURL ||
            this.draft.textToSpeech.model !== this.applied.textToSpeech.model))
      ) {
        this.#invalidateTTSConnection();
      }
      this.draft = copySettings(this.applied);
      this.microphoneChoice = microphoneChoiceFor(this.applied.microphoneID);
    }
    this.clearCredentialDraft();
    this.#messages.clear();
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
    if (this.draft) this.draft.microphoneID = microphoneIDFor(choice);
  }

  async refreshDevices() {
    if (this.devicesBusy) return;
    this.#messages.dismissError();
    this.devicesBusy = true;
    try {
      this.devices = usableDevices(await this.#service.input.ListMicrophones());
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.devicesBusy = false;
    }
  }

  async load(): Promise<boolean> {
    try {
      const [settings, processingProfiles] = await Promise.all([
        this.#service.settings.GetSettings(),
        this.#service.settings.GetPostProcessingProfiles(),
      ]);
      this.#adopt(settings);
      this.processingProfiles = processingProfiles ?? [];
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    }
  }

  async retryConfiguration(): Promise<boolean> {
    if (this.configurationRetrying || this.configurationResetting) return false;
    this.configurationRetrying = true;
    this.#messages.dismissError();
    try {
      const settings = await this.#service.settings.RetryConfiguration();
      this.#adopt(settings);
      if (settings.configuration.recoveryRequired) return false;
      this.#messages.announce("Saved settings loaded successfully.");
      await Promise.all([this.refreshDevices(), this.#refreshHistory()]);
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    } finally {
      this.configurationRetrying = false;
    }
  }

  async resetConfiguration(): Promise<boolean> {
    if (this.configurationRetrying || this.configurationResetting) return false;
    this.configurationResetting = true;
    this.#messages.dismissError();
    try {
      const settings = await this.#service.settings.ResetConfiguration();
      this.#adopt(settings);
      this.clearCredentialDraft();
      this.#invalidateSTTConnection();
      this.#invalidateProcessingConnection();
      this.#invalidateTTSConnection();
      this.#messages.announce(
        "Settings reset. Review the setup before starting transcription.",
      );
      await Promise.all([this.refreshDevices(), this.#refreshHistory()]);
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    } finally {
      this.configurationResetting = false;
    }
  }

  async save(): Promise<boolean> {
    if (!this.draft || this.saving) return false;
    this.saving = true;
    this.#messages.clear();
    try {
      const previous = this.applied;
      const sttCredentialChanged = this.apiKey !== "" || this.clearKey;
      const processingCredentialChanged =
        this.processingAPIKey !== "" || this.clearProcessingKey;
      const ttsCredentialChanged = this.ttsAPIKey !== "" || this.clearTTSKey;
      const saved = await this.#service.settings.SaveSettings({
        settings: this.draft,
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
            previous.compatibilityProfile !== saved.compatibilityProfile ||
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
            previous.postProcessing.compatibilityProfile !== saved.postProcessing.compatibilityProfile ||
            previous.postProcessing.model !== saved.postProcessing.model))
      ) {
        this.#invalidateProcessingConnection();
      }
      if (
        ttsCredentialChanged ||
        (previous &&
          (previous.textToSpeech.baseURL !== saved.textToSpeech.baseURL ||
            previous.textToSpeech.compatibilityProfile !== saved.textToSpeech.compatibilityProfile ||
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
      await this.#refreshHistory();
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    } finally {
      this.saving = false;
    }
  }

  async completeSetup(): Promise<boolean> {
    if (!this.applied || this.setupCompleting) return false;
    this.setupCompleting = true;
    this.#messages.dismissError();
    try {
      const next = copySettings(this.applied);
      next.setupCompleted = true;
      const saved = await this.#service.settings.SaveSettings({
        settings: next,
        sttCredentialDraft: "",
        clearSTTCredential: false,
        postProcessingCredentialDraft: "",
        clearPostProcessingCredential: false,
        textToSpeechCredentialDraft: "",
        clearTextToSpeechCredential: false,
      });
      this.#adopt(saved);
      this.#messages.announce("Freehand is ready to use.");
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
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
    if (!this.applied || this.isQuickSettingsPending(field)) return false;
    this.quickSettingsPending = [...this.quickSettingsPending, field];
    if (this.quickSettingsSaved === field) this.quickSettingsSaved = null;
    this.#messages.dismissError();

    let operationResult = false;
    const operation = this.#quickSettingsQueue.then(async () => {
      if (!this.applied) return;
      const next = copySettings(this.applied);
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

      const saved = await this.#service.settings.SaveSettings({
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
      this.#messages.fail(cause);
      return false;
    } finally {
      this.quickSettingsPending = this.quickSettingsPending.filter(
        (pending) => pending !== field,
      );
    }
  }

  async testConnection(
    settings = this.draft,
    apiKey = this.apiKey,
    clearExistingMessages = true,
  ) {
    if (this.sttConnectionTesting || !settings) return;
    const revision = this.#sttConnectionRevision;
    this.sttConnectionTesting = true;
    if (clearExistingMessages) this.#messages.clear();
    try {
      const result = await this.#service.connection.TestConnection({
        baseURL: settings.baseURL,
        compatibilityProfile: settings.compatibilityProfile,
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
      if (revision === this.#sttConnectionRevision) this.#messages.fail(cause);
    } finally {
      this.sttConnectionTesting = false;
      if (revision === this.#sttConnectionRevision)
        this.sttConnectionChecked = true;
    }
  }

  async testPostProcessingConnection(
    settings = this.draft,
    apiKey = this.processingAPIKey,
  ) {
    if (this.processingConnectionTesting || !settings) return;
    this.processingConnectionTesting = true;
    this.processingConnection = null;
    this.#messages.clear();
    try {
      this.processingConnection =
        await this.#service.connection.TestPostProcessingConnection({
          baseURL: settings.postProcessing.baseURL,
          compatibilityProfile: settings.postProcessing.compatibilityProfile,
          allowInsecureHTTP: settings.postProcessing.allowInsecureHTTP,
          model: settings.postProcessing.model,
          credentialDraft: apiKey,
        });
      this.processingConnectionStale = false;
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.processingConnectionTesting = false;
    }
  }

  async testTextToSpeechConnection(
    settings = this.draft,
    apiKey = this.ttsAPIKey,
  ) {
    if (this.ttsConnectionTesting || !settings) return;
    this.ttsConnectionTesting = true;
    this.ttsConnection = null;
    this.#messages.clear();
    try {
      this.ttsConnection =
        await this.#service.connection.TestTextToSpeechConnection({
          baseURL: settings.textToSpeech.baseURL,
          compatibilityProfile: settings.textToSpeech.compatibilityProfile,
          allowInsecureHTTP: settings.textToSpeech.allowInsecureHTTP,
          authenticationMode: settings.textToSpeech.authenticationMode,
          model: settings.textToSpeech.model,
          credentialDraft: apiKey,
        });
      this.ttsConnectionStale = false;
    } catch (cause) {
      this.#messages.fail(cause);
    } finally {
      this.ttsConnectionTesting = false;
    }
  }

  busy = $derived(
    this.saving ||
      this.setupCompleting ||
      this.sttConnectionTesting ||
      this.processingConnectionTesting ||
      this.ttsConnectionTesting ||
      this.configurationRetrying ||
      this.configurationResetting ||
      this.quickSettingsPending.length > 0,
  );
  dispose() {
    this.clearCredentialDraft();
    clearTimeout(this.#quickSettingsSavedTimer);
  }
}
