import { describe, it, expect, vi } from "vitest";
import { CancellablePromise } from "@wailsio/runtime";
import { VoiceScope, type VoicesResult } from "$bindings/inference";
import { createEditor, settings, idle, serviceWithStatus } from "./session-fixtures";

const result: VoicesResult = {
  voices: [{ id: "af_heart", name: "Heart", language: "en-us" }],
  scope: VoiceScope.VoiceScopeModel,
  errorKind: "", httpStatus: 200, latencyMilliseconds: 1, truncated: false,
};
function setup() {
  const services = serviceWithStatus(() => CancellablePromise.resolve(idle));
  services.connection.ListSpeechVoices = vi.fn(() => CancellablePromise.resolve(result));
  const { editor } = createEditor(services);
  const config = structuredClone(settings);
  config.savedConnections.selected = { ...config.savedConnections.selected, speech: "speech-server" };
  config.textToSpeech.model = "kokoro";
  config.textToSpeech.voice = "custom-alias";
  editor.applySettingsSnapshot(config);
  return { services, editor };
}
describe("speech voice discovery", () => {
  it("sends only the saved connection/model and preserves the current voice", async () => {
    const { services, editor } = setup();
    await editor.discoverVoices();
    expect(services.connection.ListSpeechVoices).toHaveBeenCalledWith({connectionID: "speech-server", model: "kokoro"});
    expect(editor.voices).toEqual(result);
    expect(editor.draft!.textToSpeech.voice).toBe("custom-alias");
    editor.draft!.textToSpeech.speed = 1.2;
    expect(editor.voices).toEqual(result);
    editor.draft!.textToSpeech.model = "other-model";
    expect(editor.voices).toBeNull();
  });
  it.each(["model", "connection"])("rejects late results after a %s switch", async (field) => {
    const { services, editor } = setup();
    const response = CancellablePromise.withResolvers<VoicesResult>();
    services.connection.ListSpeechVoices = vi.fn(() => response.promise);
    const pending = editor.discoverVoices();
    await editor.discoverVoices();
    if (field === "model") editor.draft!.textToSpeech.model = "other";
    else editor.draft!.savedConnections.selected = { ...editor.draft!.savedConnections.selected, speech: "other" };
    response.resolve(result);
    await pending;
    expect(editor.voices).toBeNull();
    expect(services.connection.ListSpeechVoices).toHaveBeenCalledOnce();
    expect(editor.voicesBusy).toBe(false);
  });
  it("preserves a custom voice after discovery fails", async () => {
    const { services, editor } = setup();
    await editor.discoverVoices();
    services.connection.ListSpeechVoices = vi.fn(() => CancellablePromise.resolve({...result, voices: [], errorKind: "http", httpStatus: 404}));
    await editor.discoverVoices();
    expect(editor.voices?.errorKind).toBe("http");
    expect(editor.draft!.textToSpeech.voice).toBe("custom-alias");
  });
});
