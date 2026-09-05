import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import {
  AuthenticationMode,
  PostProcessingPreset,
  type ConnectionResult,
  type Settings,
} from "$lib/state";
import type { SessionServices } from "$lib/stores/session.svelte";
import {
  settings,
  idle,
  connectionResult,
  serviceWithStatus,
  createEditor,
} from "./session-fixtures";

describe("SettingsEditor configuration recovery", () => {
  const invalidSettings: Settings = {
    ...settings,
    configuration: {
      recoveryRequired: true,
      errorKind: "invalid_json",
      message: "The settings file contains invalid JSON.",
    },
  };

  it("keeps the recovery state visible when retry still cannot load the file", async () => {
    const RetryConfiguration: SessionServices["settings"]["RetryConfiguration"] =
      vi.fn(() =>
        CancellablePromise.resolve({
          ...invalidSettings,
          configuration: {
            ...invalidSettings.configuration,
            message: "The saved value for appearanceMode has the wrong type.",
          },
        }),
      );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          GetSettings: () => CancellablePromise.resolve(invalidSettings),
          RetryConfiguration,
        },
      }),
    );

    await session.editor.load();
    expect(await session.editor.retryConfiguration()).toBe(false);
    expect(session.editor.applied?.configuration.recoveryRequired).toBe(true);
    expect(session.editor.applied?.configuration.message).toContain(
      "appearanceMode",
    );
    expect(session.editor.configurationRetrying).toBe(false);
  });

  it("adopts a recovered profile and refreshes dependent snapshots", async () => {
    const RetryConfiguration: SessionServices["settings"]["RetryConfiguration"] =
      vi.fn(() =>
        CancellablePromise.resolve({
          ...settings,
          model: "restored-model",
          configuration: {
            recoveryRequired: false,
            preservedFields: ["realtime"],
          },
        }),
      );
    const ListMicrophones: SessionServices["input"]["ListMicrophones"] = vi.fn(
      () => CancellablePromise.resolve([]),
    );
    const TranscriptHistory: SessionServices["history"]["TranscriptHistory"] =
      vi.fn(() => CancellablePromise.resolve([]));
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          GetSettings: () => CancellablePromise.resolve(invalidSettings),
          RetryConfiguration,
        },
        input: { ListMicrophones },
        history: { TranscriptHistory },
      }),
    );

    await session.editor.load();
    expect(await session.editor.retryConfiguration()).toBe(true);
    expect(session.editor.applied?.model).toBe("restored-model");
    expect(session.editor.applied?.configuration.preservedFields).toEqual([
      "realtime",
    ]);
    expect(ListMicrophones).toHaveBeenCalledOnce();
    expect(TranscriptHistory).toHaveBeenCalledOnce();
  });

  it("adopts explicit defaults and clears credential drafts after reset", async () => {
    const ResetConfiguration: SessionServices["settings"]["ResetConfiguration"] =
      vi.fn(() => CancellablePromise.resolve(settings));
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          GetSettings: () => CancellablePromise.resolve(invalidSettings),
          ResetConfiguration,
        },
      }),
    );
    await session.editor.load();
    session.editor.apiKey = "unsaved-secret";
    session.editor.processingAPIKey = "unsaved-processing-secret";

    expect(await session.editor.resetConfiguration()).toBe(true);
    expect(session.editor.applied?.configuration.recoveryRequired).toBe(false);
    expect(session.editor.apiKey).toBe("");
    expect(session.editor.processingAPIKey).toBe("");
    expect(session.messages.notice).toContain("Settings reset");
  });
});

describe("SettingsEditor credential draft", () => {
  it("clears the key and pending deletion choice together", () => {
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    session.editor.apiKey = "temporary-secret";
    session.editor.clearKey = true;

    session.editor.clearCredentialDraft();

    expect(session.editor.apiKey).toBe("");
    expect(session.editor.clearKey).toBe(false);
  });
});

describe("SettingsEditor settings snapshots", () => {
  it("adopts a backend settings event when this renderer has no draft", async () => {
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    await session.editor.load();
    await session.editor.refreshDevices();

    const changed = {
      ...settings,
      model: "speech/updated",
      postProcessing: { ...settings.postProcessing },
    };
    expect(session.editor.applySettingsSnapshot(changed)).toBe(true);
    expect(session.editor.applied?.model).toBe("speech/updated");
    expect(session.editor.draft?.model).toBe("speech/updated");
    expect(session.editor.connection).toBeNull();
  });

  it("does not overwrite an active settings-window draft", async () => {
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft) throw new Error("expected settings");
    session.editor.draft.model = "speech/draft";

    const changed = {
      ...settings,
      model: "speech/elsewhere",
      postProcessing: { ...settings.postProcessing },
    };
    expect(session.editor.applySettingsSnapshot(changed)).toBe(false);
    expect(session.editor.draft.model).toBe("speech/draft");
    expect(session.editor.applied?.model).toBe("speech/stt");
    expect(session.messages.info).toContain("another window");
  });

  it("tracks and discards unsaved settings and credential drafts", async () => {
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );

    expect(session.editor.dirty).toBe(false);
    await session.editor.load();
    await session.editor.refreshDevices();
    expect(session.editor.dirty).toBe(false);
    if (!session.editor.draft) throw new Error("expected settings");

    session.editor.draft.baseURL = "https://draft.example/v1";
    session.editor.apiKey = "temporary-secret";
    expect(session.editor.dirty).toBe(true);

    session.editor.discardSettingsDraft();
    expect(session.editor.dirty).toBe(false);
    expect(session.editor.draft.baseURL).toBe("https://example.test/v1");
    expect(session.editor.apiKey).toBe("");
  });

  it("keeps draft fields and nested headers independent from applied settings", async () => {
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );

    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft?.headers)
      throw new Error("expected settings headers");
    session.editor.draft.baseURL = "https://draft.example/v1";
    session.editor.draft.holdShortcut = "Ctrl+Alt+Space";
    session.editor.draft.headers["X-Test"] = "draft";

    expect(session.editor.applied).toMatchObject({
      baseURL: "https://example.test/v1",
      headers: { "X-Test": "applied" },
    });
    expect(session.editor.applied?.holdShortcut).toBeUndefined();
  });

  it("replaces both snapshots after Go confirms a successful save", async () => {
    const SaveSettings: SessionServices["settings"]["SaveSettings"] = ({
      settings: draft,
    }) =>
      CancellablePromise.resolve({
        ...settings,
        ...draft,
        baseURL: "https://confirmed.example/v1",
        headers: draft.headers == null ? draft.headers : { ...draft.headers },
      });
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );

    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft?.headers)
      throw new Error("expected settings headers");
    session.editor.draft.baseURL = "https://draft.example/v1";
    session.editor.draft.overlayEnabled = false;
    session.editor.draft.overlaySizePercent = 125;
    session.editor.draft.overlayOpacityPercent = 80;
    session.editor.draft.overlayTopOffset = 42;
    session.editor.draft.overlayGlowPercent = 50;
    session.editor.draft.headers["X-Test"] = "saved";
    await session.editor.save();

    expect(session.editor.draft?.baseURL).toBe("https://confirmed.example/v1");
    expect(session.editor.applied?.baseURL).toBe(
      "https://confirmed.example/v1",
    );
    expect(session.editor.applied?.headers).toEqual({ "X-Test": "saved" });
    expect(session.editor.applied?.overlayEnabled).toBe(false);
    expect(session.editor.applied?.overlaySizePercent).toBe(125);
    expect(session.editor.applied?.overlayOpacityPercent).toBe(80);
    expect(session.editor.applied?.overlayTopOffset).toBe(42);
    expect(session.editor.applied?.overlayGlowPercent).toBe(50);
    expect(session.editor.draft?.headers).not.toBe(
      session.editor.applied?.headers,
    );
  });

  it("persists completion of the one-time setup without sending credential drafts", async () => {
    const SaveSettings: SessionServices["settings"]["SaveSettings"] = vi.fn(
      ({ settings: draft }) =>
        CancellablePromise.resolve({ ...draft, setupCompleted: true }),
    );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();

    expect(await session.editor.completeSetup()).toBe(true);
    expect(SaveSettings).toHaveBeenCalledWith({
      settings: expect.objectContaining({ setupCompleted: true }),
      sttCredentialDraft: "",
      clearSTTCredential: false,
      postProcessingCredentialDraft: "",
      clearPostProcessingCredential: false,
      textToSpeechCredentialDraft: "",
      clearTextToSpeechCredential: false,
    });
    expect(session.editor.applied?.setupCompleted).toBe(true);
    expect(session.editor.draft?.setupCompleted).toBe(true);
    expect(session.messages.notice).toBe("Freehand is ready to use.");
  });

  it("keeps the launch material active and requests a restart after changing Mica", async () => {
    const SaveSettings: SessionServices["settings"]["SaveSettings"] = ({
      settings: draft,
    }) =>
      CancellablePromise.resolve({
        ...settings,
        ...draft,
        useMica: true,
        micaActive: false,
      });
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );

    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft) throw new Error("expected settings");
    session.editor.draft.useMica = true;
    await session.editor.save();

    expect(session.editor.applied?.useMica).toBe(true);
    expect(session.editor.applied?.micaActive).toBe(false);
    expect(session.messages.notice).toContain("Restart the app");
  });

  it("keeps the applied snapshot unchanged when saving fails", async () => {
    const response = CancellablePromise.withResolvers<Settings>();
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings: () => response.promise },
      }),
    );

    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft) throw new Error("expected settings");
    session.editor.draft.baseURL = "https://rejected.example/v1";
    const saving = session.editor.save();
    response.reject(new Error("save failed"));
    await saving;

    expect(session.editor.draft.baseURL).toBe("https://rejected.example/v1");
    expect(session.editor.applied?.baseURL).toBe("https://example.test/v1");
  });

  it("applies quick settings from the confirmed snapshot instead of an unrelated draft", async () => {
    let received:
      | Parameters<SessionServices["settings"]["SaveSettings"]>[0]
      | undefined;
    const SaveSettings: SessionServices["settings"]["SaveSettings"] = vi.fn(
      (request) => {
        received = request;
        return CancellablePromise.resolve({ ...settings, ...request.settings });
      },
    );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );

    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft) throw new Error("expected settings");
    session.editor.draft.autoInsert = false;
    session.editor.connection = connectionResult;
    session.editor.processingConnection = connectionResult;

    const saved = await session.editor.updateQuickSettings(
      {
        model: "speech/faster",
        postProcessing: { model: "processor/faster", styling: "formal" },
      },
      "stt-model",
    );

    expect(saved).toBe(true);
    expect(received?.settings.autoInsert).toBe(true);
    expect(received?.settings.model).toBe("speech/faster");
    expect(received?.settings.postProcessing.model).toBe("processor/faster");
    expect(received?.settings.postProcessing.styling).toBe("formal");
    expect(SaveSettings).toHaveBeenCalledWith({
      settings: expect.any(Object),
      sttCredentialDraft: "",
      clearSTTCredential: false,
      postProcessingCredentialDraft: "",
      clearPostProcessingCredential: false,
      textToSpeechCredentialDraft: "",
      clearTextToSpeechCredential: false,
    });
    expect(session.editor.draft.autoInsert).toBe(true);
    expect(session.editor.connection).toBeNull();
    expect(session.editor.processingConnection).toBeNull();
    expect(session.editor.sttConnectionStale).toBe(true);
    expect(session.editor.processingConnectionStale).toBe(true);
    expect(session.editor.quickSettingsSaved).toBe("stt-model");

    await session.editor.testConnection(session.editor.applied);
    await session.editor.testPostProcessingConnection(session.editor.applied);
    expect(session.editor.sttConnectionStale).toBe(false);
    expect(session.editor.processingConnectionStale).toBe(false);
  });

  it("keeps applied settings unchanged when a quick update fails", async () => {
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: {
          SaveSettings: () =>
            CancellablePromise.reject(new Error("quick save failed")),
        },
      }),
    );

    await session.editor.load();
    await session.editor.refreshDevices();
    const saved = await session.editor.updateQuickSettings(
      { model: "rejected-model" },
      "stt-model",
    );

    expect(saved).toBe(false);
    expect(session.editor.applied?.model).toBe("speech/stt");
    expect(session.messages.error).toContain("quick save failed");
  });

  it("persists the compact quick controls through the confirmed settings snapshot", async () => {
    let received:
      | Parameters<SessionServices["settings"]["SaveSettings"]>[0]
      | undefined;
    const SaveSettings: SessionServices["settings"]["SaveSettings"] = vi.fn(
      (request) => {
        received = request;
        return CancellablePromise.resolve(request.settings);
      },
    );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();

    expect(
      await session.editor.updateQuickSettings(
        {
          microphoneID: "usb-mic",
          vadEnabled: false,
          silenceTrimming: false,
          autoStopEnabled: false,
          silenceSplitting: false,
          maxDurationSeconds: 262,
          autoInsert: false,
          historyEnabled: true,
          overlayEnabled: false,
          postProcessing: { enabled: true },
        },
        "vad-enabled",
      ),
    ).toBe(true);

    expect(received?.settings).toMatchObject({
      microphoneID: "usb-mic",
      vadEnabled: false,
      silenceTrimming: false,
      autoStopEnabled: false,
      silenceSplitting: false,
      maxDurationSeconds: 262,
      autoInsert: false,
      historyEnabled: true,
      overlayEnabled: false,
      postProcessing: { enabled: true },
    });
    expect(session.editor.microphoneChoice).toBe("usb-mic");
    expect(session.editor.quickSettingsSaved).toBe("vad-enabled");
  });

  it("persists a processing behavior without changing its stored profile-specific values", async () => {
    let received:
      | Parameters<SessionServices["settings"]["SaveSettings"]>[0]
      | undefined;
    const SaveSettings: SessionServices["settings"]["SaveSettings"] = vi.fn(
      (request) => {
        received = request;
        return CancellablePromise.resolve({ ...settings, ...request.settings });
      },
    );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();

    expect(
      await session.editor.updateQuickSettings(
        {
          postProcessing: {
            preset: PostProcessingPreset.PostProcessingPresetS1Mini,
          },
        },
        "processing-profile",
      ),
    ).toBe(true);

    expect(received?.settings.postProcessing).toMatchObject({
      preset: PostProcessingPreset.PostProcessingPresetS1Mini,
      systemPrompt: "Clean the transcript.",
      styling: "semi-casual",
      structure: "prose",
      context: "general",
    });
    expect(session.editor.quickSettingsSaved).toBe("processing-profile");
  });

  it("serializes quick changes so each save starts from the latest confirmed snapshot", async () => {
    const requests: Settings[] = [];
    const first = CancellablePromise.withResolvers<Settings>();
    const SaveSettings: SessionServices["settings"]["SaveSettings"] = vi.fn(
      (request) => {
        requests.push(request.settings);
        if (requests.length === 1) return first.promise;
        return CancellablePromise.resolve(request.settings);
      },
    );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        settings: { SaveSettings },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();

    const endpoint = session.editor.updateQuickSettings(
      { baseURL: "https://new.example/v1" },
      "stt-endpoint",
    );
    const model = session.editor.updateQuickSettings(
      { model: "speech/new" },
      "stt-model",
    );
    await Promise.resolve();
    first.resolve({ ...settings, baseURL: "https://new.example/v1" });

    expect(await endpoint).toBe(true);
    expect(await model).toBe(true);
    expect(requests).toHaveLength(2);
    expect(requests[1]).toMatchObject({
      baseURL: "https://new.example/v1",
      model: "speech/new",
    });
  });
});

describe("SettingsEditor microphone inventory", () => {
  it("refreshes devices without rewriting the selected microphone", async () => {
    let request = 0;
    const ListMicrophones: SessionServices["input"]["ListMicrophones"] = vi.fn(
      () =>
        CancellablePromise.resolve(
          request++ === 0
            ? [
                { id: "", name: "System default microphone", default: true },
                { id: "usb-mic", name: "USB microphone", default: false },
              ]
            : [{ id: "webcam-mic", name: "Webcam microphone", default: true }],
        ),
    );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        input: { ListMicrophones },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();
    session.editor.chooseMicrophone("usb-mic");

    await session.editor.refreshDevices();

    expect(session.editor.devices).toEqual([
      { id: "webcam-mic", name: "Webcam microphone", default: true },
    ]);
    expect(session.editor.microphoneChoice).toBe("usb-mic");
    expect(session.editor.draft?.microphoneID).toBe("usb-mic");
  });

  it("debounces repeated refreshes while enumeration is pending", async () => {
    const response = CancellablePromise.withResolvers<
      { id: string; name: string; default: boolean }[] | null
    >();
    const ListMicrophones: SessionServices["input"]["ListMicrophones"] = vi.fn(
      () => response.promise,
    );
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        input: { ListMicrophones },
      }),
    );

    const first = session.editor.refreshDevices();
    await session.editor.refreshDevices();
    response.resolve([
      { id: "usb-mic", name: "USB microphone", default: true },
    ]);
    await first;

    expect(ListMicrophones).toHaveBeenCalledOnce();
    expect(session.editor.devices).toHaveLength(1);
    expect(session.editor.devicesBusy).toBe(false);
  });
});

describe("SettingsEditor connection metadata", () => {
  it("keeps the structured result for the settings-window lifetime", async () => {
    const TestConnection: SessionServices["connection"]["TestConnection"] =
      vi.fn(() => CancellablePromise.resolve(connectionResult));
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        connection: { TestConnection },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();

    await session.editor.testConnection();

    expect(TestConnection).toHaveBeenCalledWith({
      baseURL: "https://example.test/v1",
      allowInsecureHTTP: false,
      authenticationMode: AuthenticationMode.AuthenticationModeAPIKey,
      model: "speech/stt",
      healthPath: "",
      headers: { "X-Test": "applied" },
      credentialDraft: "",
    });
    expect(session.editor.connection).toEqual(connectionResult);
    expect(session.messages.notice).toBe("");
  });

  it("debounces a repeated check while one is in flight", async () => {
    const response = CancellablePromise.withResolvers<ConnectionResult>();
    const TestConnection: SessionServices["connection"]["TestConnection"] =
      vi.fn(() => response.promise);
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        connection: { TestConnection },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();

    const first = session.editor.testConnection();
    await session.editor.testConnection();
    response.resolve(connectionResult);
    await first;

    expect(TestConnection).toHaveBeenCalledOnce();
    expect(session.editor.connection).toEqual(connectionResult);
  });

  it("does not clear an existing confirmation during an automatic check", async () => {
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle)),
    );
    await session.editor.load();
    await session.editor.refreshDevices();
    session.messages.notice = "Settings saved and active.";

    await session.editor.testConnection(session.editor.applied, "", false);

    expect(session.editor.connection).toEqual(connectionResult);
    expect(session.messages.notice).toBe("Settings saved and active.");
  });

  it("sends only focused post-processing probe values", async () => {
    const TestPostProcessingConnection: SessionServices["connection"]["TestPostProcessingConnection"] =
      vi.fn(() => CancellablePromise.resolve(connectionResult));
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        connection: { TestPostProcessingConnection },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft) throw new Error("expected settings");
    session.editor.draft.postProcessing.model = "processor/s1-mini";
    session.editor.draft.postProcessing.systemPrompt =
      "unrelated unsaved prompt";

    await session.editor.testPostProcessingConnection();

    expect(TestPostProcessingConnection).toHaveBeenCalledWith({
      baseURL: "http://127.0.0.1:8080/v1",
      allowInsecureHTTP: false,
      model: "processor/s1-mini",
      credentialDraft: "",
    });
  });

  it("discovers speech playback models with its dedicated connection values", async () => {
    const TestTextToSpeechConnection: SessionServices["connection"]["TestTextToSpeechConnection"] =
      vi.fn(() => CancellablePromise.resolve(connectionResult));
    const session = createEditor(
      serviceWithStatus(() => CancellablePromise.resolve(idle), {
        connection: { TestTextToSpeechConnection },
      }),
    );
    await session.editor.load();
    await session.editor.refreshDevices();
    if (!session.editor.draft) throw new Error("expected settings");
    session.editor.draft.textToSpeech.baseURL = "http://127.0.0.1:8000/v1";
    session.editor.draft.textToSpeech.allowInsecureHTTP = true;
    session.editor.draft.textToSpeech.authenticationMode =
      AuthenticationMode.AuthenticationModeNone;
    session.editor.draft.textToSpeech.model = "local/kokoro";
    session.editor.draft.textToSpeech.voice = "unrelated-voice";

    await session.editor.testTextToSpeechConnection();

    expect(TestTextToSpeechConnection).toHaveBeenCalledWith({
      baseURL: "http://127.0.0.1:8000/v1",
      allowInsecureHTTP: true,
      authenticationMode: AuthenticationMode.AuthenticationModeNone,
      model: "local/kokoro",
      credentialDraft: "",
    });
    expect(session.editor.ttsConnection).toEqual(connectionResult);
  });
});
