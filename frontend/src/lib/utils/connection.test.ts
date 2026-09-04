import { describe, expect, it } from "vitest";
import {
  ConnectionErrorKind,
  ConnectionProbe,
  ModelPresence,
  type ConnectionResult,
} from "$lib/state";
import {
  connectionDescription,
  connectionProbeLabel,
  connectionStatusLabel,
  connectionSucceeded,
  modelPresenceLabel,
  shouldAutomaticallyTestConnection,
} from "$lib/utils/connection";

const result = (errorKind = ConnectionErrorKind.$zero): ConnectionResult => ({
  reachable: errorKind === ConnectionErrorKind.$zero,
  probe: ConnectionProbe.ConnectionProbeModels,
  requestedURL: "https://example.test/v1/models",
  httpStatus: errorKind === ConnectionErrorKind.ConnectionErrorHTTP ? 401 : 200,
  latencyMilliseconds: 12,
  errorKind,
  checkedAt: "2026-08-30T15:00:00Z",
  modelPresence: ModelPresence.ModelPresenceListed,
  modelIDs: ["speech/stt"],
});

describe("connection metadata presentation", () => {
  it("never probes an endpoint automatically before setup is complete", () => {
    expect(shouldAutomaticallyTestConnection({ setupCompleted: false }, false, false)).toBe(
      false,
    );
    expect(shouldAutomaticallyTestConnection({ setupCompleted: true }, false, false)).toBe(true);
    expect(shouldAutomaticallyTestConnection({ setupCompleted: true }, true, false)).toBe(false);
    expect(shouldAutomaticallyTestConnection({ setupCompleted: true }, false, true)).toBe(false);
  });

  it("recognizes a successful metadata-only model probe", () => {
    const value = result();
    expect(connectionSucceeded(value)).toBe(true);
    expect(connectionStatusLabel(value)).toBe("Connected");
    expect(connectionProbeLabel(value)).toBe("GET /models");
    expect(connectionDescription(value)).toContain("No model was invoked");
    expect(modelPresenceLabel(value)).toBe("Listed");
  });

  it.each([
    [ConnectionErrorKind.ConnectionErrorCredentialMissing, "Credential required"],
    [ConnectionErrorKind.ConnectionErrorCredentialUnavailable, "Credential unavailable"],
    [ConnectionErrorKind.ConnectionErrorDNS, "Name not found"],
    [ConnectionErrorKind.ConnectionErrorTLS, "TLS failed"],
    [ConnectionErrorKind.ConnectionErrorHTTP, "HTTP 401"],
    [ConnectionErrorKind.ConnectionErrorResponseTooLarge, "Response too large"],
    [ConnectionErrorKind.ConnectionErrorResponse, "Response unreadable"],
    [ConnectionErrorKind.ConnectionErrorInvalidURL, "Invalid endpoint"],
    [ConnectionErrorKind.ConnectionErrorInvalidSettings, "Check failed"],
    [ConnectionErrorKind.ConnectionErrorTimeout, "Timed out"],
    [ConnectionErrorKind.ConnectionErrorNetwork, "Network failed"],
  ])("maps %s to stable visible status", (kind, label) => {
    const value = result(kind);
    expect(connectionSucceeded(value)).toBe(false);
    expect(connectionStatusLabel(value)).toBe(label);
    expect(connectionDescription(value)).not.toBe("");
  });
});
