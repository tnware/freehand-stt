import { expect, it } from "vitest";
import { settings, connectionResult } from "$lib/stores/session-fixtures-data";
import { ConnectionErrorKind } from "$lib/state";
import { appReadiness } from "./readiness";
import { connectionDescription } from "./connection";

it("uses native credential and device wording for macOS readiness and recovery", () => {
  const mac = { ...settings, platform: "darwin", credentialConfigured: true };
  expect(
    appReadiness(mac, null, [], true).steps.find((s) => s.id === "microphone")
      ?.detail,
  ).toBe("Looking for macOS audio input devices…");
  expect(
    connectionDescription(
      {
        ...connectionResult,
        errorKind: ConnectionErrorKind.ConnectionErrorCredentialUnavailable,
      },
      mac.platform,
    ),
  ).toContain("macOS Keychain");
  expect(
    appReadiness(mac, null, [], false, "file").steps.map((s) => s.id),
  ).not.toContain("microphone");
});
