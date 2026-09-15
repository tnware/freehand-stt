import { describe, expect, it } from "vitest";
import type { Status } from "$bindings/managedruntime";
import { Purpose } from "$bindings/savedconnection";
import {
  ConnectionErrorKind,
  ConnectionProbe,
  ModelPresence,
  type ConnectionResult,
} from "$lib/state";
import { settings } from "$lib/stores/session-fixtures";
import type { TaskConnectionDetails } from "$lib/utils/connection";
import {
  managedVoiceAvailability,
  manualVoiceAvailability,
} from "./voiceCaptureAvailability";

const reachable: ConnectionResult = {
  reachable: true,
  probe: ConnectionProbe.ConnectionProbeModels,
  requestedURL: "https://voice.test/v1/models",
  httpStatus: 200,
  latencyMilliseconds: 12,
  errorKind: ConnectionErrorKind.$zero,
  checkedAt: "2026-09-15T12:00:00Z",
  modelPresence: ModelPresence.ModelPresenceListed,
  modelIDs: ["selected-speech-model"],
};
const connection: TaskConnectionDetails = {
  purpose: Purpose.Voice,
  platform: "windows",
  task: "Voice transcription",
  loading: false,
  enabled: true,
  selected: {
    id: "voice",
    name: "Voice server",
    uses: [Purpose.Voice],
    details: settings.voiceTranscription,
    builtIn: false,
    hasCredential: false,
  },
  host: "voice.test",
  model: "selected-speech-model",
  result: null,
  busy: false,
  stale: false,
};
const runtime: Status = {
  supported: true,
  state: "installed",
  enabled: true,
  selectedModel: "selected-speech-model",
  realtime: false,
  backend: "cpu",
  version: "",
  progress: -1,
  phase: "",
  error: "",
  models: [],
  acquisition: { phase: "", bytes: 0, totalBytes: 0 },
  operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
};

describe("Voice connection availability", () => {
  it("blocks the first-render gap and the active metadata check without reusing a prior result", () => {
    expect(manualVoiceAvailability(connection, false)).toMatchObject({
      label: "Checking connection",
      blocked: true,
      busy: true,
    });
    expect(
      manualVoiceAvailability(
        { ...connection, busy: true, result: reachable },
        true,
      ),
    ).toMatchObject({
      label: "Checking connection",
      blocked: true,
      busy: true,
    });
    expect(
      manualVoiceAvailability(
        { ...connection, loading: true, selected: undefined },
        false,
      ).blocked,
    ).toBe(true);
  });

  it("allows a recording after an unsupported metadata endpoint, with a visible retry action", () => {
    const view = manualVoiceAvailability(
      {
        ...connection,
        result: {
          ...reachable,
          reachable: false,
          httpStatus: 404,
          errorKind: ConnectionErrorKind.ConnectionErrorHTTP,
        },
      },
      true,
    );
    expect(view).toMatchObject({
      label: "Connection check failed",
      blocked: false,
      busy: false,
      attention: true,
      action: "check",
    });
    expect(view.detail).toContain("HTTP 404");
    expect(view.detail).toContain("Check again or try recording");
  });

  it("does not claim readiness from missing, stale, or unreachable metadata", () => {
    expect(manualVoiceAvailability(connection, true)).toMatchObject({
      label: "Connection availability unknown",
      blocked: false,
      action: "check",
    });
    expect(
      manualVoiceAvailability(
        { ...connection, stale: true, result: reachable },
        true,
      ),
    ).toMatchObject({
      label: "Connection availability unknown",
      blocked: false,
      action: "check",
    });
    const unavailable = manualVoiceAvailability(
      { ...connection, result: { ...reachable, reachable: false } },
      true,
    );
    expect(unavailable.label).toBe("Connection check failed");
    expect(unavailable.detail).not.toContain("reachable");
  });

  it("keeps coherent successful metadata informational, and does not invent an automatic check without a connection", () => {
    expect(
      manualVoiceAvailability({ ...connection, result: reachable }, true),
    ).toMatchObject({
      detail: "Server reachable",
      blocked: false,
      busy: false,
      attention: false,
    });
    expect(
      manualVoiceAvailability({ ...connection, selected: undefined }, false),
    ).toMatchObject({
      label: "Connection availability unknown",
      blocked: false,
      busy: false,
      action: "",
    });
  });
});

describe("Voice local runtime availability", () => {
  it("keeps unknown, unavailable, and failed startup states blocked", () => {
    expect(
      managedVoiceAvailability(undefined, 10_000, { checking: true }),
    ).toMatchObject({
      label: "Checking runtime",
      blocked: true,
      busy: true,
    });
    expect(
      managedVoiceAvailability(undefined, 10_000, {
        checking: true,
        error: "Could not read runtime inventory.",
      }),
    ).toMatchObject({
      label: "Runtime status unavailable",
      blocked: true,
      busy: false,
      action: "runtime",
    });
    expect(
      managedVoiceAvailability(
        {
          ...runtime,
          state: "error",
          error: "The selected model could not load.",
        },
        10_000,
      ),
    ).toMatchObject({
      label: "Runtime needs attention",
      detail: "The selected model could not load.",
      blocked: true,
      attention: true,
    });
    expect(
      managedVoiceAvailability(
        { ...runtime, supported: false, state: "running" },
        10_000,
      ).blocked,
    ).toBe(true);
  });

  it("shows pending Start immediately and then the measured native warm-up stage", () => {
    expect(
      managedVoiceAvailability(runtime, 10_000, { pending: "Start" }),
    ).toMatchObject({
      label: "Starting runtime",
      blocked: true,
      busy: true,
    });
    const warming = {
      ...runtime,
      state: "starting",
      phase: "start",
      startupProgress: { phase: "warming_up", startedAt: 8_000 },
    };
    expect(managedVoiceAvailability(warming, 10_000)).toMatchObject({
      label: "Starting runtime",
      detail: "Warming up selected model · 2s in this stage",
      blocked: true,
      busy: true,
    });
    expect(managedVoiceAvailability(warming, 11_000).detail).toContain(
      "3s in this stage",
    );
    expect(
      managedVoiceAvailability(warming, 11_000, { pending: "Cancelling" }),
    ).toMatchObject({
      label: "Cancelling operation",
      detail: "",
      blocked: true,
    });
  });

  it("requires running with no pending work, and preserves authoritative Stop admission feedback", () => {
    expect(managedVoiceAvailability(runtime, 10_000)).toMatchObject({
      label: "Runtime stopped",
      blocked: true,
    });
    const running = { ...runtime, state: "running" };
    expect(managedVoiceAvailability(running, 10_000)).toMatchObject({
      detail: "Local runtime running",
      blocked: false,
      busy: false,
    });
    expect(
      managedVoiceAvailability(running, 10_000, { pending: "Stop" }),
    ).toMatchObject({ label: "Stopping runtime", blocked: true, busy: true });
    expect(
      managedVoiceAvailability({ ...running, state: "stopping" }, 10_000),
    ).toMatchObject({ label: "Stopping runtime", blocked: true });
    expect(
      managedVoiceAvailability({ ...running, state: "stopped" }, 10_000),
    ).toMatchObject({ label: "Runtime stopped", blocked: true, busy: false });
    expect(
      managedVoiceAvailability(running, 10_000, { busy: true }).blocked,
    ).toBe(true);
  });
});
