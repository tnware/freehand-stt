import {
  ConnectionErrorKind,
  ConnectionProbe,
  ModelPresence,
  type ConnectionResult,
  type Settings,
} from "$lib/state";

import { endpointHost } from "$lib/utils/endpoint";
import { Purpose } from "$bindings/savedconnection";
import type { SettingsEditor } from "$lib/stores/editor.svelte";

export interface TaskConnectionStatus {
  scope: string;
  label: string;
  detail: string;
  title: string;
  dot: string;
}

type TaskConnectionSource = Pick<
  SettingsEditor,
  | "applied"
  | "connection"
  | "ttsConnection"
  | "sttConnectionStale"
  | "ttsConnectionStale"
  | "sttConnectionTesting"
  | "ttsConnectionTesting"
  | "connectionResultStale"
>;

export function taskConnectionStatus(
  mode: string,
  source: TaskConnectionSource,
  now: number,
): TaskConnectionStatus {
  const speech = mode === "tts";
  const scope = speech ? "Text to speech" : "Transcription";
  const settings = source.applied;
  const connection = speech ? source.ttsConnection : source.connection;
  const host = settings
    ? endpointHost(speech ? settings.textToSpeech.baseURL : settings.baseURL)
    : "";
  const summary = (
    label: string,
    detail: string,
    dot = "bg-muted-foreground",
    title = detail,
  ): TaskConnectionStatus => ({
    scope,
    label,
    detail,
    dot,
    title: `${scope}: ${title}`,
  });
  if (!settings) return summary("Loading", "reading settings");
  if (speech && !settings.textToSpeech.enabled)
    return summary("Off", "generation disabled");
  // A settings window may be checking an unsaved draft. Never attach that
  // pending check or its result to the applied endpoint used by the home task.
  if (speech ? source.ttsConnectionTesting : source.sttConnectionTesting)
    return summary("Checking", "metadata only");
  if (
    (speech ? source.ttsConnectionStale : source.sttConnectionStale) ||
    source.connectionResultStale(
      speech ? Purpose.Speech : Purpose.Transcription,
      settings,
    )
  )
    return summary(
      "Settings changed",
      [host, "check again"].filter(Boolean).join(" · "),
      "bg-warning",
    );
  if (!connection) return summary("Not checked", host);
  const elapsed = Math.max(0, now - Date.parse(connection.checkedAt));
  const minutes = Math.floor(elapsed / 60_000);
  const hours = Math.floor(minutes / 60);
  const age =
    minutes < 1
      ? "checked just now"
      : minutes < 60
        ? `checked ${minutes}m ago`
        : hours < 24
          ? `checked ${hours}h ago`
          : `checked ${Math.floor(hours / 24)}d ago`;
  return summary(
    connectionSucceeded(connection) ? "Reachable" : "Unavailable",
    [host, age].filter(Boolean).join(" · "),
    connectionSucceeded(connection) ? "bg-success" : "bg-destructive",
    `${connectionStatusLabel(connection)}${host ? ` · ${host}` : ""}. No model was invoked; inference compatibility is unverified.`,
  );
}

export const shouldAutomaticallyTestConnection = (
  settings: Pick<Settings, "setupCompleted"> | null,
  checked: boolean,
  testing: boolean,
): boolean => Boolean(settings?.setupCompleted && !checked && !testing);

export const connectionSucceeded = (result: ConnectionResult): boolean =>
  result.errorKind === ConnectionErrorKind.$zero;

export const connectionStatusLabel = (
  result: ConnectionResult | null,
): string => {
  if (!result) return "Not checked";
  if (connectionSucceeded(result)) {
    return result.probe === ConnectionProbe.ConnectionProbeHealth
      ? "Server reachable"
      : "Model list received";
  }
  switch (result.errorKind) {
    case ConnectionErrorKind.ConnectionErrorCredentialMissing:
      return "Credential required";
    case ConnectionErrorKind.ConnectionErrorCredentialUnavailable:
      return "Credential unavailable";
    case ConnectionErrorKind.ConnectionErrorDNS:
      return "Name not found";
    case ConnectionErrorKind.ConnectionErrorTLS:
      return "TLS failed";
    case ConnectionErrorKind.ConnectionErrorHTTP:
      return result.httpStatus ? `HTTP ${result.httpStatus}` : "HTTP failed";
    case ConnectionErrorKind.ConnectionErrorResponseTooLarge:
      return "Response too large";
    case ConnectionErrorKind.ConnectionErrorResponse:
      return "Invalid response";
    case ConnectionErrorKind.ConnectionErrorInvalidURL:
      return "Invalid endpoint";
    case ConnectionErrorKind.ConnectionErrorInvalidSettings:
      return "Check failed";
    case ConnectionErrorKind.ConnectionErrorTimeout:
      return "Timed out";
    default:
      return "Network failed";
  }
};

export const connectionDescription = (result: ConnectionResult): string => {
  if (connectionSucceeded(result)) {
    return result.probe === ConnectionProbe.ConnectionProbeHealth
      ? "The configured health endpoint responded successfully. No model was invoked; inference compatibility is unverified."
      : "A valid model list was received. No model was invoked; inference compatibility is unverified.";
  }
  switch (result.errorKind) {
    case ConnectionErrorKind.ConnectionErrorCredentialMissing:
      return "Enter an API key before checking this endpoint.";
    case ConnectionErrorKind.ConnectionErrorCredentialUnavailable:
      return "The stored API credential could not be read from Windows Credential Manager.";
    case ConnectionErrorKind.ConnectionErrorDNS:
      return "The endpoint name could not be resolved.";
    case ConnectionErrorKind.ConnectionErrorTLS:
      return "The endpoint could not establish a trusted TLS connection.";
    case ConnectionErrorKind.ConnectionErrorHTTP:
      return result.httpStatus
        ? `The endpoint responded with HTTP ${result.httpStatus}.`
        : "The endpoint returned an unsuccessful HTTP response.";
    case ConnectionErrorKind.ConnectionErrorResponseTooLarge:
      return "The metadata response exceeded the 1 MiB safety limit.";
    case ConnectionErrorKind.ConnectionErrorResponse:
      return result.probe === ConnectionProbe.ConnectionProbeHealth
        ? "The health response could not be read."
        : "The server responded, but its response could not be read as a valid model list.";
    case ConnectionErrorKind.ConnectionErrorInvalidURL:
      return "The configured endpoint could not be turned into a metadata URL.";
    case ConnectionErrorKind.ConnectionErrorInvalidSettings:
      return "Review the displayed endpoint settings and credential draft before checking again.";
    case ConnectionErrorKind.ConnectionErrorTimeout:
      return "The metadata check did not respond within 15 seconds.";
    default:
      return "The endpoint could not be reached over the network.";
  }
};

export const connectionProbeLabel = (result: ConnectionResult): string =>
  result.probe === ConnectionProbe.ConnectionProbeHealth
    ? "GET /health"
    : "GET /models";

export const modelPresenceLabel = (result: ConnectionResult): string => {
  if (result.modelPresence === ModelPresence.ModelPresenceListed)
    return "Listed";
  if (result.modelPresence === ModelPresence.ModelPresenceNotListed)
    return "Not listed";
  return "Unavailable";
};
