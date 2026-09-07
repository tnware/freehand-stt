import { describe, expect, it } from "vitest";
import { settings } from "$lib/stores/session-fixtures";
import { readFileSync } from "node:fs";
import { parse } from "svelte/compiler";
import { render } from "svelte/server";
import { ID, type Profile } from "$bindings/compatibility";
import CompatibilityProfilePicker from "$lib/components/settings/CompatibilityProfilePicker.svelte";

const pickerSource = readFileSync(
  new URL(
    "../components/settings/CompatibilityProfilePicker.svelte",
    import.meta.url,
  ),
  "utf8",
);

describe("compatibility profile picker options", () => {
  it("keeps an unavailable selection visible with an explanation, without changing it", () => {
    const props = {
      id: "test-compatibility",
      value: ID.LocalAI,
      profiles: [
        {
          id: ID.LocalAI,
          label: "LocalAI",
          available: false,
          description: "Dedicated support is not implemented.",
          capabilities: {} as Profile["capabilities"],
        },
      ],
    };
    const { body } = render(CompatibilityProfilePicker, { props });
    expect(body).toContain("LocalAI");
    expect(body).toContain("Dedicated support is not implemented.");
    expect(body).toContain("This saved profile is unavailable");
    expect(props.value).toBe(ID.LocalAI);
  });

  it("renders options only from the available profiles, not the planned catalog", () => {
    const ast = parse(pickerSource, { modern: true });
    const optionLists: string[] = [];
    function visit(node: unknown) {
      if (!node || typeof node !== "object") return;
      const value = node as Record<string, unknown>;
      if (value.type === "EachBlock") {
        const block = node as {
          body: { start: number; end: number };
          expression: { start: number; end: number };
        };
        if (
          pickerSource
            .slice(block.body.start, block.body.end)
            .includes("Select.Item")
        ) {
          optionLists.push(
            pickerSource.slice(block.expression.start, block.expression.end),
          );
        }
      }
      for (const child of Object.values(value)) {
        if (Array.isArray(child)) child.forEach(visit);
        else if (child && typeof child === "object") visit(child);
      }
    }
    visit(ast.fragment);
    expect(optionLists).toEqual(["available"]);
  });
});
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
  taskConnectionStatus,
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
  it.each(["voice", "file", "tts"])(
    "scopes %s metadata to its own endpoint",
    (mode) => {
      const status = taskConnectionStatus(
        mode,
        {
          applied: {
            ...settings,
            baseURL: "https://stt.test/v1",
            textToSpeech: {
              ...settings.textToSpeech,
              enabled: true,
              baseURL: "https://tts.test/v1",
            },
          },
          connectionResultStale: () => false,
          connection: result(),
          ttsConnection: result(ConnectionErrorKind.ConnectionErrorTimeout),
          sttConnectionStale: false,
          ttsConnectionStale: false,
          sttConnectionTesting: false,
          ttsConnectionTesting: false,
        },
        Date.parse("2026-08-30T15:01:00Z"),
      );
      expect(status.scope).toBe(
        mode === "tts" ? "Text to speech" : "Transcription",
      );
      expect(status.label).toBe(mode === "tts" ? "Unavailable" : "Reachable");
      expect(status.detail).toBe(
        `${mode === "tts" ? "tts.test" : "stt.test"} · checked 1m ago`,
      );
      expect(status.title).toContain("No model was invoked");
    },
  );
  it.each([
    {
      mode: "tts",
      stt: result(ConnectionErrorKind.ConnectionErrorTimeout),
      tts: result(),
      stale: false,
      testing: false,
      enabled: true,
      label: "Reachable",
    },
    {
      mode: "tts",
      stt: result(),
      tts: null,
      stale: false,
      testing: false,
      enabled: true,
      label: "Not checked",
    },
    {
      mode: "voice",
      stt: null,
      tts: result(),
      stale: false,
      testing: false,
      enabled: true,
      label: "Not checked",
    },
    {
      mode: "tts",
      stt: result(),
      tts: result(),
      stale: true,
      testing: false,
      enabled: true,
      label: "Settings changed",
    },
    {
      mode: "file",
      stt: result(),
      tts: result(),
      stale: true,
      testing: false,
      enabled: true,
      label: "Settings changed",
    },
    {
      mode: "voice",
      stt: null,
      tts: result(),
      stale: true,
      testing: false,
      enabled: true,
      label: "Settings changed",
    },
    {
      mode: "tts",
      stt: result(),
      tts: null,
      stale: true,
      testing: false,
      enabled: true,
      label: "Settings changed",
    },
    {
      mode: "tts",
      stt: result(),
      tts: result(),
      stale: true,
      testing: true,
      enabled: true,
      label: "Checking",
    },
    {
      mode: "voice",
      stt: result(),
      tts: result(),
      stale: false,
      testing: true,
      enabled: true,
      label: "Checking",
    },
    {
      mode: "tts",
      stt: result(),
      tts: result(),
      stale: false,
      testing: false,
      enabled: false,
      label: "Off",
    },
  ])(
    "keeps $mode metadata truthful: $label",
    ({ mode, stt, tts, stale, testing, enabled, label }) => {
      const status = taskConnectionStatus(
        mode,
        {
          applied: {
            ...settings,
            baseURL: "https://stt.test/v1",
            textToSpeech: {
              ...settings.textToSpeech,
              enabled,
              baseURL: "https://tts.test/v1",
            },
          },
          connectionResultStale: () => false,
          connection: stt,
          ttsConnection: tts,
          sttConnectionStale: mode !== "tts" && stale,
          ttsConnectionStale: mode === "tts" && stale,
          sttConnectionTesting: mode !== "tts" && testing,
          ttsConnectionTesting: mode === "tts" && testing,
        },
        Date.parse("2026-08-30T15:01:00Z"),
      );
      expect(status.label).toBe(label);
      if (label === "Settings changed") {
        expect(status.detail).toContain("check again");
        expect(status.dot).not.toBe("bg-success");
        expect(status.title).not.toContain("Model list received");
      }
    },
  );

  it("never probes an endpoint automatically before setup is complete", () => {
    expect(
      shouldAutomaticallyTestConnection(
        { setupCompleted: false },
        false,
        false,
      ),
    ).toBe(false);
    expect(
      shouldAutomaticallyTestConnection({ setupCompleted: true }, false, false),
    ).toBe(true);
    expect(
      shouldAutomaticallyTestConnection({ setupCompleted: true }, true, false),
    ).toBe(false);
    expect(
      shouldAutomaticallyTestConnection({ setupCompleted: true }, false, true),
    ).toBe(false);
  });

  it("recognizes a successful metadata-only model probe", () => {
    const value = result();
    expect(connectionSucceeded(value)).toBe(true);
    expect(connectionStatusLabel(value)).toBe("Model list received");
    expect(connectionProbeLabel(value)).toBe("GET /models");
    expect(connectionDescription(value)).toContain("No model was invoked");
    expect(modelPresenceLabel(value)).toBe("Listed");
  });

  it("distinguishes health reachability from model inventory validation", () => {
    const value = {
      ...result(),
      probe: ConnectionProbe.ConnectionProbeHealth,
      modelPresence: ModelPresence.ModelPresenceUnavailable,
      modelIDs: [],
    };
    expect(connectionSucceeded(value)).toBe(true);
    expect(connectionStatusLabel(value)).toBe("Server reachable");
    expect(connectionDescription(value)).toContain(
      "inference compatibility is unverified",
    );
  });

  it("does not label an HTTP 200 malformed model list as a successful check", () => {
    const value = {
      ...result(ConnectionErrorKind.ConnectionErrorResponse),
      reachable: true,
      httpStatus: 200,
      modelPresence: ModelPresence.ModelPresenceUnavailable,
      modelIDs: [],
    };
    expect(connectionSucceeded(value)).toBe(false);
    expect(connectionStatusLabel(value)).toBe("Invalid response");
    expect(connectionDescription(value)).toContain("valid model list");
  });

  it("accepts an empty valid inventory without claiming the model is listed", () => {
    const value = {
      ...result(),
      modelPresence: ModelPresence.ModelPresenceNotListed,
      modelIDs: [],
    };
    expect(connectionStatusLabel(value)).toBe("Model list received");
    expect(modelPresenceLabel(value)).toBe("Not listed");
    expect(connectionDescription(value)).toContain(
      "inference compatibility is unverified",
    );
  });

  it.each([
    [
      ConnectionErrorKind.ConnectionErrorCredentialMissing,
      "Credential required",
    ],
    [
      ConnectionErrorKind.ConnectionErrorCredentialUnavailable,
      "Credential unavailable",
    ],
    [ConnectionErrorKind.ConnectionErrorDNS, "Name not found"],
    [ConnectionErrorKind.ConnectionErrorTLS, "TLS failed"],
    [ConnectionErrorKind.ConnectionErrorHTTP, "HTTP 401"],
    [ConnectionErrorKind.ConnectionErrorResponseTooLarge, "Response too large"],
    [ConnectionErrorKind.ConnectionErrorResponse, "Invalid response"],
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
