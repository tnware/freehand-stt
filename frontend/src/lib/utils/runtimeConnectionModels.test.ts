import { describe, expect, it, vi } from "vitest";
import { render } from "svelte/server";
import {
  ProviderID,
  type Instance,
  type Model,
  type Status,
} from "$bindings/managedruntime";
import { ID as CompatibilityID } from "$bindings/compatibility";
import { AuthenticationMode } from "$bindings/config";
import { Purpose, type Connection } from "$bindings/savedconnection";
import BuiltInConnectionDetails from "$lib/components/settings/BuiltInConnectionDetails.svelte";
import ConnectionList from "$lib/components/settings/ConnectionList.svelte";
import { runtimeConnectionModels } from "./runtimeConnectionModels";

const instance: Instance = {
  id: "nemo",
  name: "Local speech",
  provider: ProviderID.NeMoSpeechCPP,
  model: "asr",
  speechModel: "tts",
  autoStart: false,
};
const model = (id: string, installed: boolean): Model => ({
  id,
  name: id === "asr" ? "Primary recognition" : "Optional speech",
  installed,
  description: "",
  sizeBytes: 1,
  recommended: true,
  realtime: false,
  profile: "generic",
  contracts: [],
});
const status = (primary = false, speech = false): Status => ({
  supported: true,
  state: "installed",
  enabled: false,
  realtime: false,
  selectedModel: "asr",
  selectedSpeechModel: "tts",
  backend: "cpu",
  version: "fixture",
  progress: -1,
  phase: "",
  error: "",
  acquisition: { phase: "", bytes: 0, totalBytes: 0 },
  operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
  models: [model("asr", primary), model("tts", speech)],
});
const connection: Connection = {
  id: "builtin-nemo",
  name: "Local speech",
  builtIn: true,
  uses: [Purpose.Voice, Purpose.Speech],
  hasCredential: false,
  details: {
    managedInstanceID: "nemo",
    compatibilityProfile: CompatibilityID.NeMoSpeechV1,
    baseURL: "http://127.0.0.1:8178",
    allowInsecureHTTP: false,
    authenticationMode: AuthenticationMode.AuthenticationModeNone,
    healthPath: "/health",
    headers: {},
  },
};

describe("runtime connection model readiness", () => {
  it.each([
    [false, true, "Model not downloaded"],
    [true, false, "Speech model not downloaded"],
    [false, false, "Models not downloaded"],
    [true, true, "Models downloaded"],
  ] as const)(
    "keeps runtime installation separate with primary=%s speech=%s",
    (primary, speech, summary) => {
      const live = status(primary, speech);
      const view = runtimeConnectionModels(instance, live);
      expect(view.summary).toBe(summary);
      expect(view.downloaded).toBe(primary && speech);
      expect(view.primary?.label).toBe(
        primary ? "Downloaded" : "Not downloaded",
      );
      expect(view.speech?.label).toBe(speech ? "Downloaded" : "Not downloaded");
      expect(live.state).toBe("installed");
    },
  );

  it("does not require an unselected speech model or treat recommendations as selections", () => {
    expect(
      runtimeConnectionModels({ ...instance, speechModel: "" }, status(true))
        .summary,
    ).toBe("Model downloaded");
    const empty = runtimeConnectionModels(
      { ...instance, model: "", speechModel: "" },
      status(true, true),
    );
    expect(empty).toMatchObject({
      primary: null,
      speech: null,
      downloaded: false,
      summary: "No model selected",
    });
  });

  it("uses saved IDs despite a stale selectedModel and never treats catalog metadata as download proof", () => {
    const saved = { ...instance, model: "other-asr", speechModel: "" };
    const view = runtimeConnectionModels(saved, status(true, true), [
      model("other-asr", true),
    ]);
    expect(view).toMatchObject({
      downloaded: false,
      summary: "Selected model unavailable",
    });
    expect(view.primary).toMatchObject({
      id: "other-asr",
      label: "Not in runtime catalog",
    });
    expect(
      runtimeConnectionModels(instance, undefined, status(true, true).models!)
        .summary,
    ).toBe("Model status unavailable");
  });

  it("shows an explicit download, then cancellation and completion without changing selections", () => {
    const live = status(true, false);
    live.operation = {
      id: 3,
      kind: "download",
      model: "tts",
      outcome: "running",
      error: "",
    };
    expect(runtimeConnectionModels(instance, live)).toMatchObject({
      summary: "Downloading speech model",
      speech: { label: "Downloading" },
    });
    live.operation.outcome = "cancelled";
    expect(runtimeConnectionModels(instance, live).summary).toBe(
      "Speech model not downloaded",
    );
    live.operation.outcome = "succeeded";
    live.models![1].installed = true;
    expect(runtimeConnectionModels(instance, live).summary).toBe(
      "Models downloaded",
    );
    expect(instance).toMatchObject({ model: "asr", speechModel: "tts" });
  });

  it("does not confuse an unrelated model download with the saved selections", () => {
    const live = status(true, true);
    live.operation = {
      id: 3,
      kind: "download",
      model: "different-model",
      outcome: "running",
      error: "",
    };
    expect(runtimeConnectionModels(instance, live).summary).toBe(
      "Models downloaded",
    );
    expect(
      runtimeConnectionModels(instance, {
        ...live,
        models: [model("asr", true)],
      }).summary,
    ).toBe("Speech model unavailable");
  });

  it.each([false, true])(
    "renders Installed and missing speech readiness together in the connection list (sidebar=%s)",
    (sidebar) => {
      const select = vi.fn();
      const live = status(true, false);
      const before = structuredClone(live);
      const html = render(ConnectionList, {
        props: {
          catalog: { entries: [connection], selected: {} },
          instances: [{ instance, status: live, activeModel: "" }],
          pendingFor: () => "",
          errorFor: () => "",
          onSelect: select,
          onAdd: vi.fn(),
          sidebar,
        },
      }).body;
      expect(html).toContain("Installed");
      expect(html).toContain("Speech model not downloaded");
      expect(live).toEqual(before);
      expect(select).not.toHaveBeenCalled();
    },
  );

  it("renders both selected models and their separate download status without claiming running readiness", () => {
    const manage = vi.fn();
    const html = render(BuiltInConnectionDetails, {
      props: {
        connection,
        instance,
        status: { instance, status: status(true, false), activeModel: "" },
        providers: [
          {
            id: ProviderID.NeMoSpeechCPP,
            name: "NeMo",
            version: "fixture",
            supported: true,
            backends: ["cpu"],
            models: status(true, true).models,
          },
        ],
        onManageRuntime: manage,
        onWorkflow: vi.fn(),
      },
    }).body;
    for (const text of [
      "Installed",
      "Selected model",
      "Primary recognition",
      "Downloaded",
      "Selected speech model",
      "Optional speech",
      "Not downloaded",
      "Download missing models",
      "Speech model ID",
    ])
      expect(html).toContain(text);
    expect(html).not.toContain("Running");
    expect(manage).not.toHaveBeenCalled();
  });
});
