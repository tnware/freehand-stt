import type { Status as ManagedStatus } from "$bindings/managedruntime";
import { runtimePresentation } from "$lib/utils/managedRuntime";
import { platformPresentation } from "$lib/platform";
import type { SettingsSectionID } from "$lib/navigation";
import {
  AuthenticationMode,
  type ConnectionResult,
  type Device,
  type Settings,
} from "$lib/state";
import {
  connectionStatusLabel,
  connectionSucceeded,
} from "$lib/utils/connection";
import { endpointLabel } from "$lib/utils/endpoint";
import {
  microphoneLabel,
  microphoneMissing,
  SYSTEM_DEFAULT_MICROPHONE,
} from "$lib/utils/microphone";

export type ReadinessStepStatus = "complete" | "pending" | "attention";

export type ReadinessStep = {
  id: "server" | "credential" | "microphone" | "shortcut" | "connection";
  label: string;
  detail: string;
  status: ReadinessStepStatus;
  blocking: boolean;
  settingsSection?: SettingsSectionID;
};

export type Readiness = {
  initialSetup: boolean;
  recoveryNeeded: boolean;
  recoveryKey: string;
  show: boolean;
  canComplete: boolean;
  canTestConnection: boolean;
  completedCount: number;
  steps: ReadinessStep[];
};

const compactModel = (model: string): string =>
  model.split("/").at(-1) ?? model;

export function appReadiness(
  settings: Settings,
  connection: ConnectionResult | null,
  devices: Device[],
  devicesLoading: boolean,
  task: "voice" | "file" = "voice",
  runtime?: ManagedStatus | null,
): Readiness {
  const native = platformPresentation(settings.platform);
  const voice = task === "voice";
  const endpoint = voice ? settings.voiceTranscription : settings;
  const hasCredential = voice
    ? !!settings.savedConnections.entries?.find(
        (c) => c.id === settings.savedConnections.selected?.voice,
      )?.hasCredential
    : settings.credentialConfigured;
  const initialSetup = voice && !settings.setupCompleted;
  const serverLoadedModel =
    !(voice && settings.voiceTranscription.realtime) &&
    !!settings.compatibilityProfiles.transcription?.find(
      (p) => p.id === endpoint.compatibilityProfile,
    )?.capabilities.serverLoadedModel;
  const serverConfigured = Boolean(
    endpoint.baseURL.trim() && (serverLoadedModel || endpoint.model.trim()),
  );
  const credentialConfigured =
    endpoint.authenticationMode === AuthenticationMode.AuthenticationModeNone ||
    hasCredential;
  const microphoneChoice = settings.microphoneID || SYSTEM_DEFAULT_MICROPHONE;
  const selectedMicrophoneMissing = microphoneMissing(
    microphoneChoice,
    devices,
  );
  const microphoneConfigured = devices.length > 0 && !selectedMicrophoneMissing;
  const shortcutConfigured = Boolean(settings.toggleShortcut.trim());
  const connectionVerified =
    connection !== null && connectionSucceeded(connection);

  const allSteps: ReadinessStep[] = [
    {
      id: "server",
      label: "Speech-to-text server",
      detail: serverConfigured
        ? `${endpointLabel(endpoint.baseURL)} · ${serverLoadedModel ? "Server-loaded model" : compactModel(endpoint.model)}`
        : serverLoadedModel
          ? "Add a server endpoint."
          : "Add an endpoint and model.",
      status: serverConfigured ? "complete" : "attention",
      blocking: !serverConfigured,
      settingsSection: voice ? "voice-transcription" : "server",
    },
    {
      id: "credential",
      label: "Authentication",
      detail:
        endpoint.authenticationMode ===
        AuthenticationMode.AuthenticationModeNone
          ? "This endpoint does not require a credential."
          : hasCredential
            ? `API key stored in ${native.credentialStore}.`
            : "Add the API key required by this endpoint.",
      status: credentialConfigured ? "complete" : "attention",
      blocking: !credentialConfigured,
      settingsSection: voice ? "voice-transcription" : "server",
    },
    {
      id: "microphone",
      label: "Microphone",
      detail: devicesLoading
        ? `Looking for ${native.name} audio input devices…`
        : devices.length === 0
          ? "No usable microphone was found."
          : selectedMicrophoneMissing
            ? "The selected microphone is not currently available."
            : microphoneLabel(microphoneChoice, devices),
      status: devicesLoading
        ? "pending"
        : microphoneConfigured
          ? "complete"
          : "attention",
      blocking: initialSetup
        ? !microphoneConfigured
        : !devicesLoading && !microphoneConfigured,
      settingsSection: "audio",
    },
    {
      id: "shortcut",
      label: "Recording shortcut",
      detail: shortcutConfigured
        ? settings.toggleShortcut
        : "Not assigned. Use Start recording in Freehand, or record a shortcut in Settings.",
      status: "complete",
      blocking: false,
      settingsSection: "shortcuts",
    },
    {
      id: "connection",
      label: "Connection check",
      detail: connectionVerified
        ? connection!.latencyMilliseconds > 0
          ? `${connectionStatusLabel(connection!)} in ${connection!.latencyMilliseconds.toLocaleString()} ms. No model was invoked.`
          : `${connectionStatusLabel(connection!)}. No model was invoked.`
        : connection
          ? `${connectionStatusLabel(connection)}. Review the server settings and try again.`
          : initialSetup
            ? "Run one metadata-only check before finishing setup."
            : "Not checked during this session.",
      status: connectionVerified
        ? "complete"
        : connection
          ? "attention"
          : "pending",
      blocking: connection ? !connectionVerified : initialSetup,
      settingsSection: voice ? "voice-transcription" : "server",
    },
  ];

  const managed = runtime?.enabled ?? settings.managedRuntime?.enabled ?? false;
  if (managed) {
    const view = runtimePresentation(runtime);
    allSteps[0] = {
      id: "server",
      label: "Local speech runtime",
      detail: view.selected?.name ?? runtime?.selectedModel ?? "Local runtime",
      status: view.ready ? "complete" : "attention",
      blocking: !view.ready,
      settingsSection: "local-runtime",
    };
    allSteps[1] = {
      id: "credential",
      label: "Local transcription",
      detail:
        "No speech-provider API key is needed. No automatic fallback to a saved server.",
      status: "complete",
      blocking: false,
    };
    allSteps[4] = {
      id: "connection",
      label: "Runtime readiness",
      detail: runtime?.error || runtime?.phase || view.label,
      status: view.ready ? "complete" : "attention",
      blocking: !view.ready,
      settingsSection: "local-runtime",
    };
  }

  const steps =
    task === "file"
      ? allSteps.filter(
          (step) => step.id !== "microphone" && step.id !== "shortcut",
        )
      : allSteps;
  const blockers = steps.filter(
    (step) => step.blocking && step.status !== "complete",
  );
  const recoveryNeeded = steps.some(
    (step) => step.status === "attention" && step.blocking,
  );
  const recoveryKey = JSON.stringify({
    managed: managed
      ? {
          state: runtime?.state,
          supported: runtime?.supported,
          model: runtime?.selectedModel,
        }
      : null,
    attention: steps
      .filter((step) => step.status === "attention" && step.blocking)
      .map((step) => step.id),
    server: endpoint.baseURL,
    model: serverLoadedModel ? "" : endpoint.model,
    profile: endpoint.compatibilityProfile,
    authentication: endpoint.authenticationMode,
    credentialConfigured: hasCredential,
    microphone: microphoneChoice,
    connection: connection
      ? {
          reachable: connection.reachable,
          probe: connection.probe,
          httpStatus: connection.httpStatus,
          errorKind: connection.errorKind,
          modelPresence: connection.modelPresence,
        }
      : null,
  });

  return {
    initialSetup,
    recoveryNeeded,
    recoveryKey,
    show: initialSetup || recoveryNeeded,
    canComplete: initialSetup && blockers.length === 0,
    canTestConnection: !managed && serverConfigured && credentialConfigured,
    completedCount: steps.filter((step) => step.status === "complete").length,
    steps,
  };
}

/** First-run setup cannot be bypassed; an established user's repeated recovery state can. */
export const readinessVisible = (
  readiness: Readiness,
  dismissedRecoveryKey: string,
): boolean =>
  readiness.initialSetup ||
  (readiness.recoveryNeeded && readiness.recoveryKey !== dismissedRecoveryKey);
