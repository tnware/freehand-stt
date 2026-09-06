import { describe, expect, it, vi } from "vitest";
import { CancellablePromise } from "@wailsio/runtime";
import { Purpose } from "$bindings/savedconnection";
import { ID } from "$bindings/modelprofile";
import { PostProcessingPreset } from "$lib/state";
import { applyModel, modelOptions } from "$lib/utils/modelSettings";
import { createEditor, settings, idle, serviceWithStatus } from "./session-fixtures";

function configured() {
  const v = structuredClone(settings);
  v.savedConnections.selected = { stt: "shared", cleanup: "shared", speech: "shared" };
  v.rememberedModels.entries = [
    {
      connectionID: "shared",
      purpose: Purpose.Cleanup,
      model: "s1",
      selected: false,
      options: { ...modelOptions(v, Purpose.Cleanup), profile: ID.S1Mini, styling: "formal" },
    },
    {
      connectionID: "shared",
      purpose: Purpose.Transcription,
      model: "whisper",
      selected: false,
      options: { ...modelOptions(v, Purpose.Transcription), language: "ja" },
    },
    {
      connectionID: "shared",
      purpose: Purpose.Speech,
      model: "voice-model",
      selected: false,
      options: { ...modelOptions(v, Purpose.Speech), voice: "voice-a", speed: 1.4 },
    },
    {
      connectionID: "another",
      purpose: Purpose.Cleanup,
      model: "s1",
      selected: false,
      options: { ...modelOptions(v, Purpose.Cleanup), profile: ID.Generic, styling: "casual" },
    },
  ];
  return v;
}
describe("remembered model settings", () => {
  it("restores engine options and preserves task intent without mutating connection or saved values", () => {
    const v = configured();
    const connections = structuredClone(v.savedConnections);
    const timeout = v.postProcessing.timeoutSeconds;
    expect(applyModel(v, Purpose.Cleanup, "s1")).toBe(true);
    expect(v.postProcessing.preset).toBe(PostProcessingPreset.PostProcessingPresetS1Mini);
    expect(v.postProcessing.styling).toBe(settings.postProcessing.styling);
    expect(applyModel(v, Purpose.Transcription, "whisper")).toBe(true);
    expect(v.language).toBe(settings.language);
    expect(applyModel(v, Purpose.Speech, "voice-model")).toBe(true);
    expect(v.textToSpeech.speed).toBe(settings.textToSpeech.speed);
    expect(v.textToSpeech.voice).toBe("voice-a");
    expect(v.savedConnections).toEqual(connections);
    expect(v.postProcessing.timeoutSeconds).toBe(timeout);
    v.transcriptionOptions.prompt = "draft";
    expect(v.rememberedModels.entries![1].options.transcription.prompt).not.toBe("draft");
  });
  it("new models use defaults and never infer S1 behavior from their names", () => {
    const v = configured();
    applyModel(v, Purpose.Cleanup, "s1");
    applyModel(v, Purpose.Cleanup, "superwhisper/s1-mini-new");
    expect(v.postProcessing.preset).toBe(PostProcessingPreset.PostProcessingPresetGeneric);
    expect(modelOptions(v, Purpose.Cleanup)).toEqual(v.rememberedModels.defaults![Purpose.Cleanup]);
  });
  it("preserves modified options while switching and discards all model drafts", () => {
    const { editor } = createEditor(serviceWithStatus(() => CancellablePromise.resolve(idle)));
    editor.applySettingsSnapshot(configured());
    expect(editor.chooseModel(Purpose.Cleanup, "s1")).toBe(true);
    editor.draft!.postProcessing.styling = "casual";
    expect(editor.chooseModel(Purpose.Cleanup, "new")).toBe(true);
    expect(editor.chooseModel(Purpose.Cleanup, "s1")).toBe(true);
    expect(editor.draft!.postProcessing.styling).toBe("casual");
    expect(editor.draft!.postProcessing.model).toBe("s1");
    editor.discardSettingsDraft();
    expect(editor.draft!.postProcessing.model).toBe(settings.postProcessing.model);
  });
  it("quick model selection saves restored options rather than previous model options", async () => {
    const bindings = serviceWithStatus(() => CancellablePromise.resolve(idle));
    bindings.settings.SaveSettings = vi.fn((request) =>
      CancellablePromise.resolve({ ...configured(), ...request.settings }),
    );
    const { editor } = createEditor(bindings);
    editor.applySettingsSnapshot(configured());
    expect(
      await editor.updateQuickSettings({ postProcessing: { model: "s1" } }, "processing-model"),
    ).toBe(true);
    const request = vi.mocked(bindings.settings.SaveSettings).mock.calls[0][0];
    expect(request.settings.postProcessing.styling).toBe(settings.postProcessing.styling);
    expect(request.settings.postProcessing.preset).toBe(
      PostProcessingPreset.PostProcessingPresetS1Mini,
    );
  });
  it("forgets using the current connection identity and refuses dirty drafts", async () => {
    const bindings = serviceWithStatus(() => CancellablePromise.resolve(idle));
    bindings.settings.SaveSettings = vi.fn(() => CancellablePromise.resolve(configured()));
    const { editor } = createEditor(bindings);
    editor.applySettingsSnapshot(configured());
    editor.draft!.language = "ja";
    expect(await editor.forgetModel(Purpose.Cleanup)).toBe(false);
    expect(bindings.settings.SaveSettings).not.toHaveBeenCalled();
    editor.discardSettingsDraft();
    expect(await editor.forgetModel(Purpose.Cleanup)).toBe(true);
    expect(
      vi.mocked(bindings.settings.SaveSettings).mock.calls[0][0].forgetModel?.connectionID,
    ).toBe("shared");
  });
  it("saves edits from multiple models together without changing their connection", async () => {
    const bindings = serviceWithStatus(() => CancellablePromise.resolve(idle));
    bindings.settings.SaveSettings = vi.fn((request) =>
      CancellablePromise.resolve({ ...configured(), ...request.settings }),
    );
    const { editor } = createEditor(bindings);
    editor.applySettingsSnapshot(configured());
    editor.chooseModel(Purpose.Cleanup, "s1");
    editor.draft!.postProcessing.styling = "casual";
    editor.chooseModel(Purpose.Cleanup, "another-model");
    editor.draft!.postProcessing.systemPrompt = "Preserve punctuation.";
    expect(await editor.save()).toBe(true);
    const edits = vi.mocked(bindings.settings.SaveSettings).mock.calls[0][0].modelEdits!;
    expect(edits).toHaveLength(2);
    expect(edits.find((e) => e.model === "s1")?.options.styling).toBe("casual");
    expect(edits.every((e) => e.connectionID === "shared")).toBe(true);
  });
});
