import { CancellablePromise } from "@wailsio/runtime";
import { describe, expect, it, vi } from "vitest";
import { TTSPhase, TTSSource, HistoryTextVersion, type TTSStatus } from "$lib/state";
import { settings, idle, serviceWithStatus } from "./session-fixtures";
import { SessionMessages } from "./messages.svelte";
import { SpeechState } from "./speech.svelte";

const bindings = () => serviceWithStatus(() => CancellablePromise.resolve(idle)).speech;

it.each(["saved", "cancelled", "failed"])(
  "guards saving until a %s outcome without blocking playback",
  async (outcome) => {
    const pending = CancellablePromise.withResolvers<boolean>();
    const SaveAudio = vi.fn(() => pending.promise);
    const Pause = vi.fn(() => CancellablePromise.resolve());
    const messages = new SessionMessages();
    const speech = new SpeechState({ ...bindings(), SaveAudio, Pause }, messages);
    try {
      const saving = speech.saveTTSAudio();
      expect(speech.saving).toBe(true);
      messages.reportInfo("An unrelated action notice");
      await speech.saveTTSAudio();
      expect(SaveAudio).toHaveBeenCalledTimes(1);
      expect(messages.info).toBe("An unrelated action notice");
      await speech.pauseTTS();
      expect(Pause).toHaveBeenCalledTimes(1);
      // The save guard belongs to the dialog/write, not the visible playback generation.
      speech.applyStatus({
        ...speech.status,
        generation: 2,
        phase: TTSPhase.Completed,
        canSave: true,
      });
      await speech.saveTTSAudio();
      expect(SaveAudio).toHaveBeenCalledTimes(1);
      if (outcome === "failed") pending.reject(new Error("Audio could not be saved"));
      else pending.resolve(outcome === "saved");
      await saving;
      expect(speech.saving).toBe(false);
      expect(messages.notice).toBe(
        outcome === "saved" ? "Generated speech saved as a WAV file." : "",
      );
      expect(messages.error).toBe(outcome === "failed" ? "Audio could not be saved" : "");
      SaveAudio.mockImplementation(() => CancellablePromise.resolve(false));
      await speech.saveTTSAudio();
      expect(SaveAudio).toHaveBeenCalledTimes(2);
      expect(speech.saving).toBe(false);
      expect(messages.error).toBe("");
    } finally {
      messages.dispose();
    }
  },
);

it("admits only one Listen request across result and history controls and recovers after rejection", async () => {
  const pending = CancellablePromise.withResolvers<void>();
  const PlayVoiceTranscript = vi.fn(() => pending.promise);
  const PlayHistoryEntry = vi.fn(() => CancellablePromise.resolve());
  const PlayFileTranscript = vi.fn(() => CancellablePromise.resolve());
  const messages = new SessionMessages();
  const speech = new SpeechState(
    { ...bindings(), PlayVoiceTranscript, PlayHistoryEntry, PlayFileTranscript },
    messages,
  );
  try {
    const first = speech.listenVoiceTranscript(7);
    expect(speech.listening).toEqual({ source: TTSSource.SourceVoice });
    expect(speech.canListen).toBe(false);
    await speech.listenVoiceTranscript(7);
    await speech.listenHistoryEntry(3, HistoryTextVersion.HistoryTextRaw);
    await speech.listenFileTranscript();
    expect(PlayVoiceTranscript).toHaveBeenCalledTimes(1);
    expect(PlayHistoryEntry).not.toHaveBeenCalled();
    expect(PlayFileTranscript).not.toHaveBeenCalled();
    pending.reject(new Error("Speech is unavailable"));
    await first;
    expect(speech.listening).toBeNull();
    expect(speech.canListen).toBe(true);
    expect(messages.error).toContain("Speech is unavailable");
    await speech.listenFileTranscript();
    expect(PlayFileTranscript).toHaveBeenCalledTimes(1);
    speech.applyStatus({ ...speech.status, generation: 1, phase: TTSPhase.Generating });
    await speech.listenFileTranscript();
    expect(PlayFileTranscript).toHaveBeenCalledTimes(1);
    for (const phase of [
      TTSPhase.Playing,
      TTSPhase.Paused,
      TTSPhase.Completed,
      TTSPhase.Failed,
      TTSPhase.Cancelled,
    ]) {
      speech.applyStatus({ ...speech.status, phase });
      expect(speech.canListen).toBe(true);
    }
    await speech.listenHistoryEntry(3, HistoryTextVersion.HistoryTextRaw);
    expect(PlayHistoryEntry).toHaveBeenCalledExactlyOnceWith(3, HistoryTextVersion.HistoryTextRaw);
  } finally {
    messages.dispose();
  }
});

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

it("prevents duplicate composer submissions while a command is pending", async () => {
  const pending = CancellablePromise.withResolvers<void>();
  const SpeakText = vi.fn(() => pending.promise);
  const messages = new SessionMessages();
  const speech = new SpeechState({ ...bindings(), SpeakText }, messages);
  const first = speech.speakText("Draft");
  await speech.speakText("Draft");
  expect(SpeakText).toHaveBeenCalledTimes(1);
  expect(speech.submitting).toBe(true);
  pending.resolve();
  await first;
  expect(speech.submitting).toBe(false);
});

it("does not apply a seek acknowledgement to replacement audio", async () => {
  const pending = CancellablePromise.withResolvers<TTSStatus>();
  const Seek = vi.fn(() => pending.promise);
  const speech = new SpeechState({ ...bindings(), Seek }, new SessionMessages());
  const original = { ...speech.status, generation: 8, canSeek: true, durationMilliseconds: 10000 };
  speech.applyStatus(original);
  const request = { generation: 8, positionMilliseconds: 5000 };
  const seeking = speech.seekTTS(request);
  await speech.seekTTS(request);
  expect(Seek).toHaveBeenCalledTimes(1);
  speech.applyStatus({ ...original, generation: 9, positionMilliseconds: 0 });
  pending.resolve({ ...original, positionMilliseconds: 5000 });
  await seeking;
  expect(speech.status.generation).toBe(9);
  expect(speech.status.positionMilliseconds).toBe(0);
  await speech.seekTTS(request);
  expect(Seek).toHaveBeenCalledTimes(1);
  expect(speech.seeking).toBe(false);
});
