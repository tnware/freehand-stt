import {
  ConnectionErrorKind,
  ConnectionProbe,
  ModelPresence,
  type ConnectionResult,
  type Settings,
} from "$lib/state";

export const shouldAutomaticallyTestConnection = (
  settings: Pick<Settings, "setupCompleted"> | null,
  checked: boolean,
  testing: boolean,
): boolean => Boolean(settings?.setupCompleted && !checked && !testing);

export const connectionSucceeded = (result: ConnectionResult): boolean =>
  result.errorKind === ConnectionErrorKind.$zero;

export const connectionStatusLabel = (result: ConnectionResult | null): string => {
  if (!result) return "Not checked";
  if (connectionSucceeded(result)) return "Connected";
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
      return "Response unreadable";
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
      ? "The configured health endpoint responded successfully."
      : "The model inventory responded successfully. No model was invoked.";
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
      return "The metadata response could not be read.";
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
  result.probe === ConnectionProbe.ConnectionProbeHealth ? "GET /health" : "GET /models";

export const modelPresenceLabel = (result: ConnectionResult): string => {
  if (result.modelPresence === ModelPresence.ModelPresenceListed) return "Listed";
  if (result.modelPresence === ModelPresence.ModelPresenceNotListed) return "Not listed";
  return "Unavailable";
};
