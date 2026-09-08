import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import { TTSPhase, HistoryTextVersion, type TTSStatus } from "$lib/state";
import { settings, idle, serviceWithStatus } from "./session-fixtures";
import { SessionMessages } from "./messages.svelte";
import { SpeechState } from "./speech.svelte";

const bindings = () => serviceWithStatus(() => CancellablePromise.resolve(idle)).speech;

describe("SpeechState", () => {
  it("keeps live playback status over a pending snapshot and ignores older generations", async () => {
    const snapshot = CancellablePromise.withResolvers<TTSStatus>();
    const speech = new SpeechState(
      { ...bindings(), CurrentStatus: () => snapshot.promise },
      new SessionMessages(),
    );
    const initial = speech.status;
    const loading = speech.load();
    const live = {
      ...initial,
      generation: 3,
      phase: TTSPhase.Failed,
      message: "speaker unavailable",
    };
    speech.applyStatus(live);
    snapshot.resolve(initial);
    await loading;
    speech.applyStatus({ ...initial, generation: 2 });
    expect(speech.status).toEqual(live);
  });

  it("forwards commands through the generated speech namespace without status/name collisions", async () => {
    const service = bindings();
    const PlayHistoryEntry = vi.fn(service.PlayHistoryEntry);
    const PlayFileTranscript = vi.fn(service.PlayFileTranscript);
    const PlayVoiceTranscript = vi.fn(service.PlayVoiceTranscript);
    const SpeakText = vi.fn(service.SpeakText);
    const Pause = vi.fn(service.Pause);
    const Resume = vi.fn(service.Resume);
    const Restart = vi.fn(service.Restart);
    const Stop = vi.fn(service.Stop);
    const SaveAudio = vi.fn(() => CancellablePromise.resolve(true));
    const ClearAudio = vi.fn(service.ClearAudio);
    const messages = new SessionMessages();
    const speech = new SpeechState(
      {
        ...service,
        PlayHistoryEntry,
        PlayFileTranscript,
        PlayVoiceTranscript,
        SpeakText,
        Pause,
        Resume,
        Restart,
        Stop,
        SaveAudio,
        ClearAudio,
      },
      messages,
    );
    try {
      await speech.listenHistoryEntry(7, HistoryTextVersion.HistoryTextRaw);
      await speech.listenFileTranscript();
      await speech.listenVoiceTranscript(17);
      await speech.speakText("user-authored text");
      await speech.pauseTTS();
      await speech.resumeTTS();
      await speech.restartTTS();
      await speech.stopTTS();
      await speech.saveTTSAudio();
      expect(messages.notice).toBe("Generated speech saved as a WAV file.");
      await speech.clearTTSAudio();
      expect(messages.notice).toBe("");
      expect(PlayHistoryEntry).toHaveBeenCalledExactlyOnceWith(
        7,
        HistoryTextVersion.HistoryTextRaw,
      );
      expect(PlayVoiceTranscript).toHaveBeenCalledExactlyOnceWith(17);
      expect(SpeakText).toHaveBeenCalledExactlyOnceWith("user-authored text");
      for (const call of [PlayFileTranscript, Pause, Resume, Restart, Stop, SaveAudio, ClearAudio])
        expect(call).toHaveBeenCalledExactlyOnceWith();
    } finally {
      messages.dispose();
    }
  });

  it("owns preview admission and resets it after rejection", async () => {
    const preview = CancellablePromise.withResolvers<void>();
    const PreviewVoice = vi.fn(() => preview.promise);
    const messages = new SessionMessages();
    const speech = new SpeechState({ ...bindings(), PreviewVoice }, messages);
    const pending = speech.previewVoice(settings);
    await speech.previewVoice(settings);
    expect(speech.previewing).toBe(true);
    expect(PreviewVoice).toHaveBeenCalledTimes(1);
    preview.reject(new Error("preview failed"));
    await pending;
    expect(speech.previewing).toBe(false);
    expect(messages.error).toContain("preview failed");
  });
});

it("keeps unsent text through status refresh and playback commands", async () => {
  const messages = new SessionMessages();
  const speech = new SpeechState(bindings(), messages);
  speech.draft = "An unfinished thought.";
  await speech.load();
  await speech.stopTTS();
  await speech.clearTTSAudio();
  expect(speech.draft).toBe("An unfinished thought.");
  messages.dispose();
});

it("previews a snapshot of unsaved speech options without forwarding transport or saving", async () => {
  const service = bindings();
  const PreviewVoice = vi.fn(service.PreviewVoice);
  const messages = new SessionMessages();
  const speech = new SpeechState({ ...service, PreviewVoice }, messages);
  const draft = structuredClone(settings);
  draft.savedConnections.selected = { speech: "draft-connection" };
  draft.textToSpeech = {
    ...draft.textToSpeech,
    enabled: true,
    model: "draft-model",
    voice: "draft-voice",
    speed: 1.5,
    timeoutSeconds: 42,
    options: { language: "ja", instructions: "Warm delivery" },
  };
  await speech.previewVoice(draft);
  draft.textToSpeech.voice = "later-edit";
  draft.textToSpeech.options.instructions = "Later instructions";
  expect(PreviewVoice).toHaveBeenCalledExactlyOnceWith({
    connectionID: "draft-connection",
    enabled: true,
    modelProfile: draft.textToSpeech.modelProfile,
    model: "draft-model",
    voice: "draft-voice",
    speed: 1.5,
    timeoutSeconds: 42,
    options: { language: "ja", instructions: "Warm delivery" },
  });
  messages.dispose();
});

it("owns only the matching speech failure and does not resurrect dismissed errors", () => {
  const messages = new SessionMessages();
  const speech = new SpeechState(bindings(), messages);
  const failed = {
    ...speech.status,
    generation: 3,
    phase: TTSPhase.Failed,
    message: "Playback failed",
  };
  speech.applyStatus(failed);
  expect(messages.isSpeechFailure(3)).toBe(true);
  messages.dismissError();
  speech.applyStatus(failed);
  expect(messages.error).toBe("");
  speech.applyStatus({ ...failed, generation: 4 });
  expect(messages.isSpeechFailure(4)).toBe(true);
  speech.applyStatus({ ...failed, generation: 5, phase: TTSPhase.Generating });
  expect(messages.error).toBe("");
  speech.applyStatus({ ...failed, generation: 5 });
  messages.fail(new Error("Save failed"));
  speech.applyStatus({ ...failed, generation: 6, phase: TTSPhase.Generating });
  expect(messages.error).toBe("Save failed");
  expect(messages.isSpeechFailure(5)).toBe(false);
});
