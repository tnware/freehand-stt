import { describe, it, expect, vi } from "vitest";
import { CancellablePromise } from "@wailsio/runtime";
import { Purpose } from "$bindings/savedconnection";
import { taskConnectionDetails, taskConnectionStatus } from "$lib/utils/connection";
import {
  createEditor,
  settings,
  idle,
  serviceWithStatus,
  connectionResult,
} from "./session-fixtures";

describe("connection assessment snapshots", () => {
  it.each(["file", "tts"])(
    "does not attribute a draft-only check to the applied %s task",
    async (mode) => {
      const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
      services.connection.TestConnection = () => CancellablePromise.resolve(connectionResult);
      services.connection.TestTextToSpeechConnection = () =>
        CancellablePromise.resolve(connectionResult);
      const { editor } = createEditor(services);
      editor.applySettingsSnapshot({
        ...structuredClone(settings),
        textToSpeech: { ...settings.textToSpeech, enabled: true },
      });
      if (mode === "tts") {
        editor.draft!.textToSpeech.baseURL = "https://unsaved-speech.test/v1";
        await editor.testTextToSpeechConnection();
      } else {
        editor.draft!.baseURL = "https://unsaved-stt.test/v1";
        await editor.testConnection();
      }
      const footer = taskConnectionStatus(mode, editor, Date.now());
      expect(footer.label).toBe("Settings changed");
      expect(footer.detail).toContain("check again");
      expect(footer.dot).not.toBe("bg-success");
    },
  );
  it("captures model options and marks results stale only for relevant changes", async () => {
    const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
    services.connection.TestConnection = vi.fn(() => CancellablePromise.resolve(connectionResult));
    const { editor } = createEditor(services);
    editor.applySettingsSnapshot(structuredClone(settings));
    await editor.testConnection();
    expect(vi.mocked(services.connection.TestConnection).mock.calls[0][0].options?.profile).toBe(
      settings.modelProfile,
    );
    expect(editor.connectionResultStale(Purpose.Transcription)).toBe(false);
    editor.draft!.maxDurationSeconds += 1;
    expect(editor.connectionResultStale(Purpose.Transcription)).toBe(false);
    editor.draft!.language = "ja";
    expect(editor.connectionResultStale(Purpose.Transcription)).toBe(true);
    editor.discardSettingsDraft();
    expect(editor.connectionResultStale(Purpose.Transcription)).toBe(false);
  });
  it("does not present late results as a check of a newer draft", async () => {
    let resolve!: (result: typeof connectionResult) => void;
    const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
    services.connection.TestConnection = vi.fn(
      () =>
        new CancellablePromise<typeof connectionResult>((r) => {
          resolve = r;
        }),
    );
    const { editor } = createEditor(services);
    editor.applySettingsSnapshot(structuredClone(settings));
    const pending = editor.testConnection();
    editor.draft!.model = "new-model";
    resolve(connectionResult);
    await pending;
    expect(editor.connectionResultStale(Purpose.Transcription)).toBe(true);
  });
  it("keeps cleanup and speech assessment inputs independent", async () => {
    const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
    services.connection.TestPostProcessingConnection = vi.fn(() =>
      CancellablePromise.resolve(connectionResult),
    );
    services.connection.TestTextToSpeechConnection = vi.fn(() =>
      CancellablePromise.resolve(connectionResult),
    );
    const { editor } = createEditor(services);
    editor.applySettingsSnapshot(structuredClone(settings));
    await editor.testPostProcessingConnection();
    await editor.testTextToSpeechConnection();
    editor.draft!.postProcessing.systemPrompt = "Changed instruction.";
    expect(editor.connectionResultStale(Purpose.Cleanup)).toBe(true);
    expect(editor.connectionResultStale(Purpose.Speech)).toBe(false);
    editor.draft!.textToSpeech.voice = "another-voice";
    expect(editor.connectionResultStale(Purpose.Speech)).toBe(true);
  });
});

describe("explicit workspace connection checks", () => {
  it.each([Purpose.Voice, Purpose.Transcription, Purpose.Speech])(
    "checks the applied %s profile without credential drafts",
    async (purpose) => {
      const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
      services.connection.TestSavedConnection = vi.fn(() =>
        CancellablePromise.resolve(connectionResult),
      );
      services.connection.TestConnection = vi.fn(() =>
        CancellablePromise.resolve(connectionResult),
      );
      services.connection.TestTextToSpeechConnection = vi.fn(() =>
        CancellablePromise.resolve(connectionResult),
      );
      const { editor } = createEditor(services);
      const applied = structuredClone(settings);
      applied.savedConnections.selected = {
        voice: "voice-active",
        [Purpose.Transcription]: "file-active",
        speech: "speech-active",
      };
      applied.textToSpeech = {
        ...applied.textToSpeech,
        enabled: true,
        baseURL: "https://active-speech.test/v1",
      };
      editor.applySettingsSnapshot(applied);
      editor.draft!.baseURL = "https://draft-file.test/v1";
      editor.draft!.textToSpeech.baseURL = "https://draft-speech.test/v1";
      editor.apiKey = "fixture-draft-key";
      editor.ttsAPIKey = "fixture-draft-key";
      await editor.testAppliedConnection(purpose);
      if (purpose === Purpose.Voice) {
        expect(services.connection.TestSavedConnection).toHaveBeenCalledExactlyOnceWith(
          "voice-active",
        );
      } else if (purpose === Purpose.Transcription) {
        expect(services.connection.TestConnection).toHaveBeenCalledWith(
          expect.objectContaining({ baseURL: applied.baseURL, credentialDraft: "" }),
        );
      } else {
        expect(services.connection.TestTextToSpeechConnection).toHaveBeenCalledWith(
          expect.objectContaining({
            baseURL: "https://active-speech.test/v1",
            credentialDraft: "",
          }),
        );
      }
      expect(
        vi.mocked(services.connection.TestSavedConnection).mock.calls.length +
          vi.mocked(services.connection.TestConnection).mock.calls.length +
          vi.mocked(services.connection.TestTextToSpeechConnection).mock.calls.length,
      ).toBe(1);
      const mode =
        purpose === Purpose.Voice ? "voice" : purpose === Purpose.Speech ? "tts" : "file";
      expect(taskConnectionDetails(mode, editor).stale).toBe(false);
    },
  );

  it("does not check disabled speech, missing selections, or pending saves", async () => {
    const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
    services.connection.TestTextToSpeechConnection = vi.fn(() =>
      CancellablePromise.resolve(connectionResult),
    );
    services.connection.TestConnection = vi.fn(() => CancellablePromise.resolve(connectionResult));
    const { editor } = createEditor(services);
    const applied = structuredClone(settings);
    applied.savedConnections.selected = { speech: "speech-active" };
    applied.textToSpeech.enabled = false;
    editor.applySettingsSnapshot(applied);
    await editor.testAppliedConnection(Purpose.Speech);
    await editor.testAppliedConnection(Purpose.Transcription);
    editor.applied!.textToSpeech.enabled = true;
    editor.saving = true;
    await editor.testAppliedConnection(Purpose.Speech);
    expect(services.connection.TestTextToSpeechConnection).not.toHaveBeenCalled();
    expect(services.connection.TestConnection).not.toHaveBeenCalled();
  });
});
