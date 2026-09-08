import { describe, expect, it } from "vitest";
import { AuthenticationMode } from "$lib/state";
import { settings, connectionResult } from "$lib/stores/session-fixtures-data";
import { appReadiness } from "./readiness";
import { nextReadinessAction } from "./readinessPresentation";

const configured = () => ({
  ...structuredClone(settings),
  voiceTranscription: {
    ...settings.voiceTranscription,
    baseURL: "https://speech.test/v1",
    model: "speech/stt",
    authenticationMode: AuthenticationMode.AuthenticationModeNone,
  },
});
const devices = [{ id: "mic", name: "Microphone", default: true }];

describe("readiness next action", () => {
  it("keeps prerequisites ahead of checking and completion", () => {
    const current = configured();
    current.voiceTranscription.model = "";
    expect(nextReadinessAction(appReadiness(current, null, [], false))).toMatchObject({
      kind: "settings",
      section: "voice-transcription",
    });
    current.voiceTranscription.model = "speech/stt";
    expect(nextReadinessAction(appReadiness(current, null, [], false))).toMatchObject({
      kind: "settings",
      section: "audio",
    });
    current.toggleShortcut = "";
    expect(nextReadinessAction(appReadiness(current, null, devices, false))).toMatchObject({
      kind: "settings",
      section: "shortcuts",
    });
  });
  it("waits for device discovery and offers finish only after metadata success", () => {
    const current = configured();
    expect(nextReadinessAction(appReadiness(current, null, [], true)).kind).toBe("wait");
    expect(nextReadinessAction(appReadiness(current, null, devices, false)).kind).toBe("check");
    expect(nextReadinessAction(appReadiness(current, connectionResult, devices, false)).kind).toBe(
      "complete",
    );
  });
  it("routes missing authentication before issuing another check", () => {
    const current = configured();
    current.voiceTranscription.authenticationMode = AuthenticationMode.AuthenticationModeAPIKey;
    expect(nextReadinessAction(appReadiness(current, null, devices, false))).toMatchObject({
      kind: "settings",
      step: "credential",
    });
  });
  it("does not offer voice prerequisites or setup completion for files", () => {
    const current = configured();
    current.authenticationMode = AuthenticationMode.AuthenticationModeNone;
    current.toggleShortcut = "";
    expect(nextReadinessAction(appReadiness(current, null, [], false, "file")).kind).toBe("check");
    expect(
      nextReadinessAction(appReadiness(current, connectionResult, [], false, "file")).kind,
    ).not.toBe("complete");
  });
});
