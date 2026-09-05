import {
  Action,
  Purpose,
  type Change,
  type Connection,
  type Details,
} from "$bindings/savedconnection";
import { AuthenticationMode } from "$lib/state";
import { ID } from "$bindings/compatibility";
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
    | "TestSavedConnection"
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
  model?: string;
  postProcessing?: Partial<
    Pick<
      Settings["postProcessing"],
      "enabled" | "model" | "preset" | "styling" | "structure" | "context"
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
  | "stt-model"
  | "processing-enabled"
  | "processing-model"
  | "processing-profile"
  | "processing-controls";

/** Keeps the editable draft independent from the backend-confirmed snapshot. */
const copySettings = (settings: Settings): Settings => ({
  ...settings,
  savedConnections: {
    selected: { ...settings.savedConnections.selected },
    entries: (settings.savedConnections.entries ?? []).map((c) => ({
      ...c,
      details: { ...c.details, headers: { ...c.details.headers } },
    })),
  },
  transcriptionOptions: { ...settings.transcriptionOptions },
  headers:
    settings.headers == null ? settings.headers : { ...settings.headers },
  postProcessing: {
    ...settings.postProcessing,
    generationOptions: { ...settings.postProcessing.generationOptions },
  },
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
  connectionDraft = $state<
    | (Pick<
        Connection,
        "id" | "name" | "purpose" | "details" | "hasCredential"
      > & {
        creating: boolean;
        credentialDraft: string;
        clearCredential: boolean;
      })
    | null
  >(null);
  managedConnectionResult = $state<ConnectionResult | null>(null);
  managedConnectionTesting = $state(false);
  #managedConnectionRevision = 0;
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
  #processingConnectionRevision = 0;
  #ttsConnectionRevision = 0;
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
    this.#processingConnectionRevision++;
    this.processingConnectionStale =
      this.processingConnectionStale || this.processingConnection !== null;
    this.processingConnection = null;
  }

  #invalidateTTSConnection() {
    this.#ttsConnectionRevision++;
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
    if (settings.configuration.recoveryRequired) {
      this.#adopt(settings);
      this.clearCredentialDraft();
      return true;
    }
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
      (previous.savedConnections.selected?.stt !==
        settings.savedConnections.selected?.stt ||
        previous.baseURL !== settings.baseURL ||
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
      (previous.savedConnections.selected?.cleanup !==
        settings.savedConnections.selected?.cleanup ||
        previous.postProcessing.baseURL !== settings.postProcessing.baseURL ||
        previous.postProcessing.compatibilityProfile !==
          settings.postProcessing.compatibilityProfile ||
        previous.postProcessing.model !== settings.postProcessing.model)
    ) {
      this.#invalidateProcessingConnection();
    }
    if (
      previous &&
      (previous.savedConnections.selected?.speech !==
        settings.savedConnections.selected?.speech ||
        previous.textToSpeech.baseURL !== settings.textToSpeech.baseURL ||
        previous.textToSpeech.compatibilityProfile !==
          settings.textToSpeech.compatibilityProfile ||
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
    this.connectionDraft = null;
    this.#managedConnectionRevision++;
    this.managedConnectionResult = null;
    this.managedConnectionTesting = false;
    this.apiKey = "";
    this.clearKey = false;
    this.processingAPIKey = "";
    this.clearProcessingKey = false;
    this.ttsAPIKey = "";
    this.clearTTSKey = false;
  }

  get dirty(): boolean {
    return this.runtimeDirty || this.connectionDraft !== null;
  }

  get runtimeDirty(): boolean {
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

  #connectionExpectation() {
    return this.applied?.savedConnections.entries?.length
      ? { expectedConnections: { ...this.applied.savedConnections.selected } }
      : {};
  }

  beginConnection(connection?: Connection, purpose = Purpose.Transcription) {
    if (this.runtimeDirty || this.saving) {
      this.#messages.reportInfo(
        "Save or discard feature settings before editing a connection.",
      );
      return;
    }
    this.#messages.clear();
    this.managedConnectionResult = null;
    this.#managedConnectionRevision++;
    this.connectionDraft = {
      id: connection?.id ?? "",
      name: connection?.name ?? "",
      purpose: connection?.purpose ?? purpose,
      hasCredential: connection?.hasCredential ?? false,
      creating: !connection,
      details: connection
        ? { ...connection.details, headers: { ...connection.details.headers } }
        : {
            compatibilityProfile: ID.Generic,
            baseURL: "",
            allowInsecureHTTP: false,
            authenticationMode: AuthenticationMode.AuthenticationModeNone,
            healthPath: "",
            headers: {},
          },
      credentialDraft: "",
      clearCredential: false,
    };
  }
  cancelConnectionEdit() {
    this.clearCredentialDraft();
  }
  async saveConnection(): Promise<boolean> {
    const form = this.connectionDraft;
    if (!form) return false;
    return this.changeConnection(
      {
        action: form.creating ? Action.Create : Action.Update,
        purpose: form.purpose,
        id: form.id,
        name: form.name.trim(),
        replacementID: "",
        details: form.details,
      },
      form.credentialDraft,
      form.clearCredential,
    );
  }
  async testSavedConnection(id: string): Promise<void> {
    const revision = ++this.#managedConnectionRevision;
    this.managedConnectionTesting = true;
    this.managedConnectionResult = null;
    try {
      const result = await this.#service.connection.TestSavedConnection(id);
      if (revision === this.#managedConnectionRevision)
        this.managedConnectionResult = result;
    } catch (cause) {
      if (revision === this.#managedConnectionRevision)
        this.#messages.fail(cause);
    } finally {
      if (revision === this.#managedConnectionRevision)
        this.managedConnectionTesting = false;
    }
  }
  async changeConnection(
    change: Change,
    credentialDraft = "",
    clearCredential = false,
  ): Promise<boolean> {
    if (!this.draft || this.saving || this.quickSettingsPending.length)
      return false;
    if (
      this.runtimeDirty ||
      (this.connectionDraft &&
        change.action !== Action.Create &&
        change.action !== Action.Update)
    ) {
      this.#messages.reportInfo(
        "Save or discard your current edits before changing connections.",
      );
      return false;
    }
    this.saving = true;
    this.#messages.clear();
    try {
      const saved = await this.#service.settings.SaveSettings({
        expectedConnections: { ...this.applied?.savedConnections.selected },
        connectionChange: change,
        connectionCredentialDraft: credentialDraft,
        clearConnectionCredential: clearCredential,
        settings: this.applied!,
        clearSTTCredential: false,
        clearPostProcessingCredential: false,
        clearTextToSpeechCredential: false,
      });
      this.#adopt(saved);
      this.clearCredentialDraft();
      if (change.purpose === Purpose.Transcription)
        this.#invalidateSTTConnection();
      if (change.purpose === Purpose.Cleanup)
        this.#invalidateProcessingConnection();
      if (change.purpose === Purpose.Speech) this.#invalidateTTSConnection();
      this.#announceSettingsSaved(
        saved,
        change.action === Action.Select
          ? "Connection selected. Choose a model for this feature; saved connection details are unchanged."
          : change.action === Action.Delete
            ? "Connection deleted."
            : "Connection saved. Active feature selections are unchanged.",
      );
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    } finally {
      this.saving = false;
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
        ...this.#connectionExpectation(),
        clearConnectionCredential: false,
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
            previous.postProcessing.compatibilityProfile !==
              saved.postProcessing.compatibilityProfile ||
            previous.postProcessing.model !== saved.postProcessing.model))
      ) {
        this.#invalidateProcessingConnection();
      }
      if (
        ttsCredentialChanged ||
        (previous &&
          (previous.textToSpeech.baseURL !== saved.textToSpeech.baseURL ||
            previous.textToSpeech.compatibilityProfile !==
              saved.textToSpeech.compatibilityProfile ||
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
        ...this.#connectionExpectation(),
        clearConnectionCredential: false,
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
        ...this.#connectionExpectation(),
        clearConnectionCredential: false,
        settings: next,
        sttCredentialDraft: "",
        clearSTTCredential: false,
        postProcessingCredentialDraft: "",
        clearPostProcessingCredential: false,
        textToSpeechCredentialDraft: "",
        clearTextToSpeechCredential: false,
      });
      this.#adopt(saved);
      if (patch.model !== undefined) {
        this.#invalidateSTTConnection();
      }
      if (patch.postProcessing?.model !== undefined) {
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
    const revision = this.#processingConnectionRevision;
    this.processingConnectionTesting = true;
    this.processingConnection = null;
    this.#messages.clear();
    try {
      const result =
        await this.#service.connection.TestPostProcessingConnection({
          baseURL: settings.postProcessing.baseURL,
          compatibilityProfile: settings.postProcessing.compatibilityProfile,
          allowInsecureHTTP: settings.postProcessing.allowInsecureHTTP,
          model: settings.postProcessing.model,
          credentialDraft: apiKey,
        });
      if (revision === this.#processingConnectionRevision) {
        this.processingConnection = result;
        this.processingConnectionStale = false;
      }
    } catch (cause) {
      if (revision === this.#processingConnectionRevision)
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
    const revision = this.#ttsConnectionRevision;
    this.ttsConnectionTesting = true;
    this.ttsConnection = null;
    this.#messages.clear();
    try {
      const result = await this.#service.connection.TestTextToSpeechConnection({
        baseURL: settings.textToSpeech.baseURL,
        compatibilityProfile: settings.textToSpeech.compatibilityProfile,
        allowInsecureHTTP: settings.textToSpeech.allowInsecureHTTP,
        authenticationMode: settings.textToSpeech.authenticationMode,
        model: settings.textToSpeech.model,
        credentialDraft: apiKey,
      });
      if (revision === this.#ttsConnectionRevision) {
        this.ttsConnection = result;
        this.ttsConnectionStale = false;
      }
    } catch (cause) {
      if (revision === this.#ttsConnectionRevision) this.#messages.fail(cause);
    } finally {
      this.ttsConnectionTesting = false;
    }
  }

  busy = $derived(
    this.saving ||
      this.setupCompleting ||
      this.managedConnectionTesting ||
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
