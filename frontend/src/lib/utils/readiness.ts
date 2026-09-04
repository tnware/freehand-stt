import type { SettingsSectionID } from "$lib/navigation";
import {
  AuthenticationMode,
  type ConnectionResult,
  type Device,
  type Settings,
} from "$lib/state";
import { connectionStatusLabel, connectionSucceeded } from "$lib/utils/connection";
import { endpointLabel } from "$lib/utils/endpoint";
import { microphoneLabel, microphoneMissing, SYSTEM_DEFAULT_MICROPHONE } from "$lib/utils/microphone";

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

const compactModel = (model: string): string => model.split("/").at(-1) ?? model;

export function appReadiness(
  settings: Settings,
  connection: ConnectionResult | null,
  devices: Device[],
  devicesLoading: boolean,
): Readiness {
  const initialSetup = !settings.setupCompleted;
  const serverConfigured = Boolean(settings.baseURL.trim() && settings.model.trim());
  const credentialConfigured =
    settings.authenticationMode === AuthenticationMode.AuthenticationModeNone ||
    settings.credentialConfigured;
  const microphoneChoice = settings.microphoneID || SYSTEM_DEFAULT_MICROPHONE;
  const selectedMicrophoneMissing = microphoneMissing(microphoneChoice, devices);
  const microphoneConfigured = devices.length > 0 && !selectedMicrophoneMissing;
  const shortcutConfigured = Boolean(settings.toggleShortcut.trim());
  const connectionVerified = connection !== null && connectionSucceeded(connection);

  const steps: ReadinessStep[] = [
    {
      id: "server",
      label: "Speech-to-text server",
      detail: serverConfigured
        ? `${endpointLabel(settings.baseURL)} · ${compactModel(settings.model)}`
        : "Add an endpoint and model.",
      status: serverConfigured ? "complete" : "attention",
      blocking: !serverConfigured,
      settingsSection: "server",
    },
    {
      id: "credential",
      label: "Authentication",
      detail:
        settings.authenticationMode === AuthenticationMode.AuthenticationModeNone
          ? "This endpoint does not require a credential."
          : settings.credentialConfigured
            ? "API key stored in Windows Credential Manager."
            : "Add the API key required by this endpoint.",
      status: credentialConfigured ? "complete" : "attention",
      blocking: !credentialConfigured,
      settingsSection: "server",
    },
    {
      id: "microphone",
      label: "Microphone",
      detail: devicesLoading
        ? "Looking for Windows audio input devices…"
        : devices.length === 0
          ? "No usable microphone was found."
          : selectedMicrophoneMissing
            ? "The selected microphone is not currently available."
            : microphoneLabel(microphoneChoice, devices),
      status: devicesLoading ? "pending" : microphoneConfigured ? "complete" : "attention",
      blocking: initialSetup ? !microphoneConfigured : !devicesLoading && !microphoneConfigured,
      settingsSection: "audio",
    },
    {
      id: "shortcut",
      label: "Recording shortcut",
      detail: shortcutConfigured ? settings.toggleShortcut : "Choose a global recording shortcut.",
      status: shortcutConfigured ? "complete" : "attention",
      blocking: !shortcutConfigured,
      settingsSection: "shortcuts",
    },
    {
      id: "connection",
      label: "Connection check",
      detail: connectionVerified
        ? connection!.latencyMilliseconds > 0
          ? `Connected in ${connection!.latencyMilliseconds.toLocaleString()} ms. No model was invoked.`
          : "Connected. No model was invoked."
        : connection
          ? `${connectionStatusLabel(connection)}. Review the server settings and try again.`
          : initialSetup
            ? "Run one metadata-only check before finishing setup."
            : "Not checked during this session.",
      status: connectionVerified ? "complete" : connection ? "attention" : "pending",
      blocking: connection ? !connectionVerified : initialSetup,
      settingsSection: "server",
    },
  ];

  const blockers = steps.filter((step) => step.blocking && step.status !== "complete");
  const recoveryNeeded = steps.some(
    (step) => step.status === "attention" && step.blocking,
  );
  const recoveryKey = JSON.stringify({
    attention: steps
      .filter((step) => step.status === "attention" && step.blocking)
      .map((step) => step.id),
    server: settings.baseURL,
    model: settings.model,
    authentication: settings.authenticationMode,
    credentialConfigured: settings.credentialConfigured,
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
    canTestConnection: serverConfigured && credentialConfigured,
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
