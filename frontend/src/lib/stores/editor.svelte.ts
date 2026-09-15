import {
  copySettings,
  quickSettingsDraft,
  type QuickSettingsPatch,
} from "$lib/utils/settingsDraft";
import { VoiceScope, type VoicesResult } from "$bindings/inference";
import {
  managedConnectionMetadata,
  type RuntimeMetadataSource,
} from "$lib/utils/managedConnectionMetadata";
import { connectionInputKey } from "$lib/utils/connectionInputs";
import type { Edit } from "$bindings/modelsettings";
import {
  Action,
  Purpose,
  type Change,
  type Connection,
  type Details,
} from "$bindings/savedconnection";
import {
  applyModelOptions,
  applyModel,
  modelFor,
  modelOptions,
  savedModelOptions,
} from "$lib/utils/modelSettings";
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
  settingsValidationIssue,
  type SettingsValidationIssue,
} from "$lib/utils/settingsValidation";
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
    | "ListSpeechVoices"
    | "TestSavedConnection"
    | "TestConnection"
    | "TestPostProcessingConnection"
    | "TestTextToSpeechConnection"
  >;
}

export type { QuickSettingsPatch } from "$lib/utils/settingsDraft";

export type QuickSettingsField =
  | "speech-controls"
  | "voice-transcription"
  | "microphone"
  | "vad-enabled"
  | "silence-splitting"
  | "delivery"
  | "history-enabled"
  | "overlay-enabled"
  | "stt-model"
  | "stt-language"
  | "processing-enabled"
  | "processing-model"
  | "processing-profile"
  | "processing-controls";

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
    | (Pick<Connection, "id" | "name" | "details" | "hasCredential"> & {
        uses: Purpose[];
        creating: boolean;
        credentialDraft: string;
        clearCredential: boolean;
      })
    | null
  >(null);
  #connectionBaseline = "";
  managedConnectionResult = $state<ConnectionResult | null>(null);
  managedConnectionTesting = $state(false);
  savedConnectionCheckingID = $state("");
  savedConnectionChecks = $state<Record<string, ConnectionResult>>({});
  savedConnectionCheckErrors = $state<Record<string, string>>({});
  #managedConnectionRevision = 0;
  readonly #service: SettingsEditorServices;
  readonly #messages: SessionMessages;
  readonly #refreshHistory: () => Promise<void>;
  constructor(
    service: SettingsEditorServices,
    messages: SessionMessages,
    refreshHistory: () => Promise<void>,
    private readonly runtimeMetadata?: RuntimeMetadataSource,
  ) {
    this.#service = service;
    this.#messages = messages;
    this.#refreshHistory = refreshHistory;
  }
  managedMetadata(purpose: Purpose, settings = this.applied) {
    return managedConnectionMetadata(settings, purpose, this.runtimeMetadata);
  }
  #metadataInputKey(settings: Settings, purpose: Purpose) {
    return (
      connectionInputKey(settings, purpose) +
      (this.managedMetadata(purpose, settings)?.key ?? "")
    );
  }
  #metadataBlocked(purpose: Purpose, settings = this.applied) {
    return this.managedMetadata(purpose, settings)?.ready === false;
  }
  #modelDrafts = $state<Edit[]>([]);
  draft = $state<Settings | null>(null);
  validationIssue = $state<SettingsValidationIssue | null>(null);
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
  quickSettingsFailed = $state<QuickSettingsField | null>(null);
  devicesBusy = $state(false);
  voicesBusy = $state(false);
  #voices = $state<VoicesResult | null>(null);
  #voicesInput = $state("");
  #voicesRevision = $state(-1);

  #voiceInput(settings = this.draft): string {
    if (!settings) return "";
    return JSON.stringify([
      settings.savedConnections.selected?.speech,
      settings.textToSpeech.model,
      settings.textToSpeech.compatibilityProfile,
      settings.textToSpeech.baseURL,
    ]);
  }

  get voices(): VoicesResult | null {
    return this.voicesFor(this.draft);
  }

  voicesFor(settings: Settings | null): VoicesResult | null {
    return this.#voicesInput === this.#voiceInput(settings) &&
      this.#voicesRevision === this.#ttsConnectionRevision
      ? this.#voices
      : null;
  }

  async discoverVoices(applied = false) {
    const current = () => (applied ? this.applied : this.draft);
    const settings = current();
    const connectionID = settings?.savedConnections.selected?.speech;
    if (this.voicesBusy || !connectionID || !settings) return;
    const input = this.#voiceInput(settings);
    const revision = this.#ttsConnectionRevision;
    this.voicesBusy = true;
    try {
      const result = await this.#service.connection.ListSpeechVoices({
        connectionID,
        model: settings.textToSpeech.model,
      });
      if (
        revision === this.#ttsConnectionRevision &&
        input === this.#voiceInput(current())
      ) {
        this.#voices = result;
        this.#voicesInput = input;
        this.#voicesRevision = revision;
      }
    } catch {
      if (
        revision === this.#ttsConnectionRevision &&
        input === this.#voiceInput(current())
      ) {
        this.#voices = {
          voices: [],
          scope: VoiceScope.$zero,
          errorKind: "network",
          httpStatus: 0,
          latencyMilliseconds: 0,
          truncated: false,
        };
        this.#voicesInput = input;
        this.#voicesRevision = revision;
      }
    } finally {
      this.voicesBusy = false;
    }
  }

  #pendingExternalSettings: Settings | null = null;

  #testedInputs = $state<Partial<Record<Purpose, string>>>({});
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

  /** Invalidate confirmed metadata independently of whether local drafts can be replaced. */
  #invalidateConfirmedMetadata() {
    // A confirmed snapshot may include a credential rotation invisible to the renderer.
    // Keep check results only until that boundary; late responses cannot repopulate them.
    this.#clearSavedConnectionChecks();
    this.#invalidateSTTConnection();
    this.#invalidateProcessingConnection();
    this.#invalidateTTSConnection();
    this.#voiceConnectionRevision++;
    this.voiceConnection = null;
    this.voiceConnectionTesting = false;
    this.#voiceTestID = "";
    this.#testedInputs = {};
    this.#automaticMetadataInputs = {};
  }

  /** Applies a settings payload confirmed by Go and starts a fresh draft. */
  #adopt(settings: Settings) {
    this.#invalidateConfirmedMetadata();
    this.quickSettingsFailed = null;
    this.validationIssue = null;
    this.#pendingExternalSettings = null;
    this.#modelDrafts = [];
    this.applied = copySettings(settings);
    this.draft = copySettings(this.applied);
    this.microphoneChoice = microphoneChoiceFor(this.applied.microphoneID);
  }

  #invalidateSTTConnection() {
    delete this.#testedInputs[Purpose.Transcription];
    delete this.#automaticMetadataInputs[Purpose.Transcription];
    this.sttConnectionTesting = false;
    this.sttConnectionStale =
      this.sttConnectionStale ||
      this.connection !== null ||
      this.sttConnectionChecked;
    this.#sttConnectionRevision++;
    this.connection = null;
    this.sttConnectionChecked = false;
  }

  #invalidateProcessingConnection() {
    delete this.#testedInputs[Purpose.Cleanup];
    delete this.#automaticMetadataInputs[Purpose.Cleanup];
    this.processingConnectionTesting = false;
    this.#processingConnectionRevision++;
    this.processingConnectionStale =
      this.processingConnectionStale || this.processingConnection !== null;
    this.processingConnection = null;
  }

  #invalidateTTSConnection() {
    delete this.#testedInputs[Purpose.Speech];
    delete this.#automaticMetadataInputs[Purpose.Speech];
    this.ttsConnectionTesting = false;
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
    if ((this.dirty || this.connectionDraft !== null) && !this.saving) {
      this.#invalidateConfirmedMetadata();
      this.#pendingExternalSettings = copySettings(settings);
      this.#messages.reportInfo(
        "Settings changed in another window. Your edits are preserved; discard them to load the latest settings.",
      );
      return false;
    }
    this.#adopt(settings);
    return true;
  }

  #clearSavedConnectionChecks() {
    this.#managedConnectionRevision++;
    this.managedConnectionTesting = false;
    this.managedConnectionResult = null;
    this.savedConnectionCheckingID = "";
    this.savedConnectionChecks = {};
    this.savedConnectionCheckErrors = {};
  }

  clearCredentialDraft() {
    this.#clearSavedConnectionChecks();
    this.connectionDraft = null;
    this.#connectionBaseline = "";
    this.apiKey = "";
    this.clearKey = false;
    this.processingAPIKey = "";
    this.clearProcessingKey = false;
    this.ttsAPIKey = "";
    this.clearTTSKey = false;
    // A discarded connection draft must not strand Home on the older snapshot.
    // Keep runtime drafts protected until those edits are explicitly resolved too.
    if (this.#pendingExternalSettings && !this.runtimeDirty && !this.saving) {
      this.applySettingsSnapshot(this.#pendingExternalSettings);
    }
  }

  get dirty(): boolean {
    return this.runtimeDirty || this.connectionDirty;
  }

  // Only non-credential fields enter the comparison snapshot. A replacement
  // key stays solely in the transient form and is checked for presence.
  #connectionFields(): string {
    const form = this.connectionDraft;
    return form
      ? JSON.stringify({
          name: form.name,
          uses: [...form.uses].sort(),
          details: form.details,
        })
      : "";
  }

  get connectionDirty(): boolean {
    const form = this.connectionDraft;
    return (
      form !== null &&
      (form.credentialDraft !== "" ||
        form.clearCredential ||
        this.#connectionFields() !== this.#connectionBaseline)
    );
  }

  get runtimeDirty(): boolean {
    return (
      this.#modelDrafts.length > 0 ||
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
    this.validationIssue = null;
    this.#modelDrafts = [];
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

  beginConnection(connection?: Connection, purpose = Purpose.Voice) {
    if (connection?.builtIn) return;
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
      uses: connection ? [...(connection.uses ?? [])] : [purpose],
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
    this.#connectionBaseline = this.#connectionFields();
  }
  setConnectionTarget(managedInstanceID: string | undefined) {
    const form = this.connectionDraft;
    if (!form || this.saving) return;
    form.credentialDraft = "";
    form.clearCredential = false;
    this.apiKey = "";
    this.processingAPIKey = "";
    this.ttsAPIKey = "";
    this.clearKey = this.clearProcessingKey = this.clearTTSKey = false;
    form.details = {
      ...(managedInstanceID !== undefined ? { managedInstanceID } : {}),
      compatibilityProfile:
        managedInstanceID !== undefined ? ID.$zero : ID.Generic,
      baseURL: "",
      allowInsecureHTTP: false,
      authenticationMode: AuthenticationMode.AuthenticationModeNone,
      healthPath: "",
      headers: {},
    };
  }
  cancelConnectionEdit() {
    this.clearCredentialDraft();
  }
  async saveConnection(activateFor?: Purpose): Promise<boolean> {
    const form = this.connectionDraft;
    if (!form) return false;
    return this.changeConnection(
      {
        action: form.creating ? Action.Create : Action.Update,
        ...(activateFor ? { activateFor } : {}),
        uses: [...form.uses],
        id: form.id,
        name: form.name.trim(),
        details: form.details,
      },
      form.credentialDraft,
      form.clearCredential,
    );
  }
  async testSavedConnection(id: string): Promise<void> {
    if (
      this.managedConnectionTesting ||
      !this.applied?.savedConnections.entries?.some((c) => c.id === id)
    )
      return;
    const revision = ++this.#managedConnectionRevision;
    this.managedConnectionTesting = true;
    this.savedConnectionCheckingID = id;
    this.managedConnectionResult = null;
    delete this.savedConnectionCheckErrors[id];
    delete this.savedConnectionChecks[id];
    try {
      const result = await this.#service.connection.TestSavedConnection(id);
      if (revision === this.#managedConnectionRevision) {
        this.managedConnectionResult = result;
        this.savedConnectionChecks[id] = result;
      }
    } catch (cause) {
      if (revision === this.#managedConnectionRevision) {
        this.savedConnectionCheckErrors[id] =
          "Could not check this connection. Try again.";
        this.#messages.fail(cause);
      }
    } finally {
      if (revision === this.#managedConnectionRevision) {
        this.managedConnectionTesting = false;
        this.savedConnectionCheckingID = "";
      }
    }
  }
  async changeConnection(
    change: Change,
    credentialDraft = "",
    clearCredential = false,
  ): Promise<boolean> {
    if (!this.draft || this.saving || this.quickSettingsPending.length)
      return false;
    const target = this.applied?.savedConnections.entries?.find(
      (c) => c.id === change.id,
    );
    if (target?.builtIn && change.action !== Action.Select) {
      this.#messages.reportInfo(
        "This connection is owned by its local runtime. Manage the runtime to change it.",
      );
      return false;
    }
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
    if (
      change.details?.managedInstanceID !== undefined &&
      change.details.managedInstanceID !== null
    ) {
      const details = change.details;
      if (
        !details.managedInstanceID ||
        credentialDraft ||
        clearCredential ||
        details.baseURL ||
        details.healthPath ||
        Object.keys(details.headers ?? {}).length ||
        details.allowInsecureHTTP ||
        details.authenticationMode !== AuthenticationMode.AuthenticationModeNone
      ) {
        if (this.connectionDraft) this.connectionDraft.credentialDraft = "";
        this.#messages.reportInfo(
          "Managed connections require an instance and cannot contain a URL, headers, or credentials.",
        );
        return false;
      }
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
      this.#invalidateSTTConnection();
      this.#invalidateProcessingConnection();
      this.#invalidateTTSConnection();
      this.#announceSettingsSaved(
        saved,
        change.action === Action.Select
          ? "Connection selected."
          : change.action === Action.Delete
            ? "Connection deleted."
            : change.activateFor
              ? "Connection saved. Choose a model to finish setup."
              : "Connection saved.",
      );
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    } finally {
      this.saving = false;
    }
  }

  modelDraftIDs(purpose: Purpose): string[] {
    return this.#modelDrafts
      .filter((e) => e.purpose === purpose)
      .map((e) => e.model);
  }
  #retainModelDraft(purpose: Purpose) {
    if (!this.draft || !this.applied) return;
    const model = modelFor(this.draft, purpose),
      connectionID = this.draft.savedConnections.selected?.[purpose];
    if (!model || !connectionID) return;
    const options = modelOptions(this.draft, purpose);
    const baseline =
      model === modelFor(this.applied, purpose)
        ? modelOptions(this.applied, purpose)
        : savedModelOptions(this.draft, purpose, model);
    const rest = this.#modelDrafts.filter(
      (e) =>
        !(
          e.purpose === purpose &&
          e.connectionID === connectionID &&
          e.model === model
        ),
    );
    this.#modelDrafts =
      baseline && JSON.stringify(options) === JSON.stringify(baseline)
        ? rest
        : [...rest, { connectionID, purpose, model, options }];
  }
  chooseModel(purpose: Purpose, model: string): boolean {
    // Metadata reads do not lock local model drafts; every mutation guard remains.
    if (
      !this.draft ||
      !this.applied ||
      this.saving ||
      this.setupCompleting ||
      this.managedConnectionTesting ||
      this.configurationRetrying ||
      this.configurationResetting ||
      this.quickSettingsPending.length > 0
    )
      return false;
    model = model.trim();
    if (model === modelFor(this.draft, purpose)) return true;
    this.#retainModelDraft(purpose);
    const cached = this.#modelDrafts.find(
      (e) =>
        e.purpose === purpose &&
        e.connectionID === this.draft?.savedConnections.selected?.[purpose] &&
        e.model === model,
    );
    if (cached) applyModelOptions(this.draft, purpose, model, cached.options);
    else if (!applyModel(this.draft, purpose, model)) return false;
    return true;
  }

  async forgetModel(purpose: Purpose): Promise<boolean> {
    if (!this.applied || this.busy || this.runtimeDirty) {
      this.#messages.reportInfo(
        "Save or discard your edits before forgetting a model.",
      );
      return false;
    }
    this.saving = true;
    try {
      const saved = await this.#service.settings.SaveSettings({
        ...this.#connectionExpectation(),
        settings: this.applied,
        forgetModel: {
          connectionID: this.applied.savedConnections.selected?.[purpose] ?? "",
          purpose,
          model: modelFor(this.applied, purpose),
        },
        clearConnectionCredential: false,
        clearSTTCredential: false,
        clearPostProcessingCredential: false,
        clearTextToSpeechCredential: false,
      });
      this.#adopt(saved);
      this.#messages.announce(
        "Model preferences forgotten. Choose a model to continue.",
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
    this.validationIssue = null;
    for (const purpose of [
      Purpose.Transcription,
      Purpose.Cleanup,
      Purpose.Speech,
      Purpose.Voice,
    ])
      this.#retainModelDraft(purpose);
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
        modelEdits: this.#modelDrafts,
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
      this.validationIssue = settingsValidationIssue(cause);
      if (!this.validationIssue) this.#messages.fail(cause);
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
    if (this.quickSettingsFailed === field) this.quickSettingsFailed = null;
    this.#messages.dismissError();

    let operationResult = false;
    const operation = this.#quickSettingsQueue.then(async () => {
      if (!this.applied) return;
      const next = quickSettingsDraft(this.applied, patch);

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
      this.quickSettingsFailed = field;
      this.#messages.fail(cause);
      return false;
    } finally {
      this.quickSettingsPending = this.quickSettingsPending.filter(
        (pending) => pending !== field,
      );
    }
  }

  connectionResultStale(purpose: Purpose, settings = this.draft): boolean {
    const tested = this.#testedInputs[purpose];
    return !!(
      tested &&
      settings &&
      tested !== this.#metadataInputKey(settings, purpose)
    );
  }

  voiceConnection = $state<ConnectionResult | null>(null);
  voiceConnectionTesting = $state(false);
  #voiceTestID = $state("");
  #voiceConnectionRevision = 0;
  async testVoiceConnection() {
    const settings = this.applied;
    const id = settings?.savedConnections.selected?.voice;
    if (
      !settings ||
      !id ||
      this.voiceConnectionTesting ||
      this.#metadataBlocked(Purpose.Voice, settings)
    )
      return;
    const key = this.#metadataInputKey(settings, Purpose.Voice);
    const revision = this.#voiceConnectionRevision;
    this.voiceConnectionTesting = true;
    this.voiceConnection = null;
    delete this.#testedInputs[Purpose.Voice];
    this.#automaticMetadataInputs[Purpose.Voice] = key;
    this.#voiceTestID = key;
    try {
      const result = await this.#service.connection.TestSavedConnection(id);
      if (
        revision === this.#voiceConnectionRevision &&
        this.applied &&
        key === this.#metadataInputKey(this.applied, Purpose.Voice)
      ) {
        this.voiceConnection = result;
        this.#testedInputs[Purpose.Voice] = key;
        this.#voiceTestID = key;
        return result;
      }
    } catch (cause) {
      if (revision === this.#voiceConnectionRevision)
        this.#messages.fail(cause);
    } finally {
      if (revision === this.#voiceConnectionRevision)
        this.voiceConnectionTesting = false;
    }
  }
  get voiceConnectionChecked(): boolean {
    return (
      !!this.applied &&
      this.#voiceTestID === this.#metadataInputKey(this.applied, Purpose.Voice)
    );
  }
  get currentVoiceConnection(): ConnectionResult | null {
    return this.applied &&
      this.#voiceTestID === this.#metadataInputKey(this.applied, Purpose.Voice)
      ? this.voiceConnection
      : null;
  }

  #automaticMetadataInputs = $state<Partial<Record<Purpose, string>>>({});
  connectionMetadataResult(purpose: Purpose): ConnectionResult | null {
    if (!this.applied || this.connectionResultStale(purpose, this.applied))
      return null;
    return purpose === Purpose.Voice
      ? this.currentVoiceConnection
      : purpose === Purpose.Transcription
        ? this.connection
        : purpose === Purpose.Cleanup
          ? this.processingConnection
          : this.ttsConnection;
  }
  connectionMetadataStatus(
    purpose: Purpose,
  ): "idle" | "loading" | "ready" | "empty" | "failed" {
    if (this.connectionMetadataBusy(purpose)) return "loading";
    const result = this.connectionMetadataResult(purpose);
    if (result)
      return !result.reachable || result.errorKind
        ? "failed"
        : result.modelIDs?.length
          ? "ready"
          : "empty";
    return this.applied &&
      this.#automaticMetadataInputs[purpose] ===
        this.#metadataInputKey(this.applied, purpose)
      ? "failed"
      : "idle";
  }
  connectionMetadataBusy(purpose: Purpose): boolean {
    return purpose === Purpose.Voice
      ? this.voiceConnectionTesting
      : purpose === Purpose.Transcription
        ? this.sttConnectionTesting
        : purpose === Purpose.Cleanup
          ? this.processingConnectionTesting
          : this.ttsConnectionTesting;
  }
  /** An explicit workspace check always uses the applied profile and no renderer credential draft. */
  async testAppliedConnection(purpose: Purpose): Promise<void> {
    const settings = this.applied;
    if (
      !settings ||
      settings.configuration.recoveryRequired ||
      this.saving ||
      this.quickSettingsPending.length ||
      !settings.savedConnections.selected?.[purpose] ||
      this.connectionMetadataBusy(purpose)
    )
      return;
    switch (purpose) {
      case Purpose.Voice:
        await this.testVoiceConnection();
        break;
      case Purpose.Transcription:
        await this.testConnection(settings, "", false);
        break;
      case Purpose.Cleanup:
        await this.testPostProcessingConnection(settings, "");
        break;
      case Purpose.Speech:
        if (settings.textToSpeech.enabled)
          await this.testTextToSpeechConnection(settings, "");
        break;
    }
  }

  /** Effects use the default; a deliberate open may retry one failed attempt. */
  async ensureConnectionMetadata(
    purpose: Purpose,
    deliberateEntry = false,
  ): Promise<void> {
    const settings = this.applied;
    if (
      !settings?.savedConnections.selected?.[purpose] ||
      settings.configuration.recoveryRequired ||
      this.saving ||
      this.quickSettingsPending.length
    )
      return;
    if (
      this.connectionMetadataBusy(purpose) ||
      this.#metadataBlocked(purpose, settings)
    )
      return;
    const key = this.#metadataInputKey(settings, purpose);
    const retry =
      deliberateEntry && this.connectionMetadataStatus(purpose) === "failed";
    if (
      !retry &&
      (this.#automaticMetadataInputs[purpose] === key ||
        this.#testedInputs[purpose] === key)
    )
      return;
    this.#automaticMetadataInputs[purpose] = key;
    switch (purpose) {
      case Purpose.Voice:
        await this.testVoiceConnection();
        break;
      case Purpose.Transcription:
        await this.testConnection(settings, "", false);
        break;
      case Purpose.Cleanup:
        await this.testPostProcessingConnection(settings, "");
        break;
      case Purpose.Speech:
        await this.testTextToSpeechConnection(settings, "");
        break;
    }
  }

  async testConnection(
    settings = this.draft,
    apiKey = this.apiKey,
    clearExistingMessages = true,
  ) {
    if (
      this.sttConnectionTesting ||
      !settings ||
      this.#metadataBlocked(Purpose.Transcription, settings)
    )
      return;
    const inputKey = this.#metadataInputKey(settings, Purpose.Transcription);
    const revision = this.#sttConnectionRevision;
    this.connection = null;
    delete this.#testedInputs[Purpose.Transcription];
    this.#automaticMetadataInputs[Purpose.Transcription] = inputKey;
    this.sttConnectionTesting = true;
    if (clearExistingMessages) this.#messages.clear();
    try {
      const result = settings.managedInstanceID
        ? await this.#service.connection.TestSavedConnection(
            settings.savedConnections.selected?.stt ?? "",
          )
        : await this.#service.connection.TestConnection({
            baseURL: settings.baseURL,
            compatibilityProfile: settings.compatibilityProfile,
            allowInsecureHTTP: settings.allowInsecureHTTP,
            authenticationMode: settings.authenticationMode,
            model: settings.model,
            healthPath: settings.healthPath ?? "",
            headers: settings.headers,
            options: modelOptions(settings, Purpose.Transcription),
            credentialDraft: apiKey,
          });
      if (revision === this.#sttConnectionRevision) {
        this.#testedInputs[Purpose.Transcription] = inputKey;
        this.connection = result;
        this.sttConnectionStale = false;
      }
    } catch (cause) {
      if (revision === this.#sttConnectionRevision) this.#messages.fail(cause);
    } finally {
      if (revision === this.#sttConnectionRevision)
        this.sttConnectionTesting = false;
      if (revision === this.#sttConnectionRevision)
        this.sttConnectionChecked = true;
    }
  }

  async testPostProcessingConnection(
    settings = this.draft,
    apiKey = this.processingAPIKey,
  ) {
    if (
      this.processingConnectionTesting ||
      !settings ||
      this.#metadataBlocked(Purpose.Cleanup, settings)
    )
      return;
    const inputKey = this.#metadataInputKey(settings, Purpose.Cleanup);
    const revision = this.#processingConnectionRevision;
    this.processingConnection = null;
    delete this.#testedInputs[Purpose.Cleanup];
    this.#automaticMetadataInputs[Purpose.Cleanup] = inputKey;
    this.processingConnectionTesting = true;
    this.#messages.clear();
    try {
      const result = settings.postProcessing.managedInstanceID
        ? await this.#service.connection.TestSavedConnection(
            settings.savedConnections.selected?.cleanup ?? "",
          )
        : await this.#service.connection.TestPostProcessingConnection({
            baseURL: settings.postProcessing.baseURL,
            compatibilityProfile: settings.postProcessing.compatibilityProfile,
            allowInsecureHTTP: settings.postProcessing.allowInsecureHTTP,
            model: settings.postProcessing.model,
            options: modelOptions(settings, Purpose.Cleanup),
            credentialDraft: apiKey,
          });
      if (revision === this.#processingConnectionRevision) {
        this.#testedInputs[Purpose.Cleanup] = inputKey;
        this.processingConnection = result;
        this.processingConnectionStale = false;
      }
    } catch (cause) {
      if (revision === this.#processingConnectionRevision)
        this.#messages.fail(cause);
    } finally {
      if (revision === this.#processingConnectionRevision)
        this.processingConnectionTesting = false;
    }
  }

  async testTextToSpeechConnection(
    settings = this.draft,
    apiKey = this.ttsAPIKey,
  ) {
    if (
      this.ttsConnectionTesting ||
      !settings ||
      this.#metadataBlocked(Purpose.Speech, settings)
    )
      return;
    const inputKey = this.#metadataInputKey(settings, Purpose.Speech);
    const revision = this.#ttsConnectionRevision;
    this.ttsConnection = null;
    delete this.#testedInputs[Purpose.Speech];
    this.#automaticMetadataInputs[Purpose.Speech] = inputKey;
    this.ttsConnectionTesting = true;
    this.#messages.clear();
    try {
      const result = settings.textToSpeech.managedInstanceID
        ? await this.#service.connection.TestSavedConnection(
            settings.savedConnections.selected?.speech ?? "",
          )
        : await this.#service.connection.TestTextToSpeechConnection({
            baseURL: settings.textToSpeech.baseURL,
            compatibilityProfile: settings.textToSpeech.compatibilityProfile,
            allowInsecureHTTP: settings.textToSpeech.allowInsecureHTTP,
            authenticationMode: settings.textToSpeech.authenticationMode,
            model: settings.textToSpeech.model,
            options: modelOptions(settings, Purpose.Speech),
            credentialDraft: apiKey,
          });
      if (revision === this.#ttsConnectionRevision) {
        this.#testedInputs[Purpose.Speech] = inputKey;
        this.ttsConnection = result;
        this.ttsConnectionStale = false;
      }
    } catch (cause) {
      if (revision === this.#ttsConnectionRevision) this.#messages.fail(cause);
    } finally {
      if (revision === this.#ttsConnectionRevision)
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
