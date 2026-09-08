import { describe, it, expect, vi } from "vitest";
import { CancellablePromise } from "@wailsio/runtime";
import { Purpose } from "$bindings/savedconnection";
import { taskConnectionStatus } from "$lib/utils/connection";
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
