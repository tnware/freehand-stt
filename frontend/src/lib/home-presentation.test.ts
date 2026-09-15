import { readFileSync } from "node:fs";
import { render } from "svelte/server";
import { describe, expect, it } from "vitest";
import CurrentResult from "$lib/components/home/CurrentResult.svelte";
import QuickControls from "$lib/components/home/QuickControls.svelte";
import TextToSpeech from "$lib/components/home/TextToSpeech.svelte";
import { TTSPhase, TTSSource, type TTSStatus } from "$lib/state";
import { settings } from "$lib/stores/session-fixtures";

const read = (path: string) =>
  readFileSync(new URL(path, import.meta.url), "utf8");

const idleSpeech: TTSStatus = {
  generation: 0,
  phase: TTSPhase.Idle,
  positionMilliseconds: 0,
  durationMilliseconds: 0,
  canPause: false,
  canResume: false,
  canRestart: false,
  canSeek: false,
  canStop: false,
  canSave: false,
  canClear: false,
};

describe("home task presentation", () => {
  it("uses Freehand as the accessible document and updater title", () => {
    expect(read("../../index.html")).toContain("<title>Freehand</title>");
    expect(read("../../../internal/app/app.go")).toContain(
      'Title: "Freehand — Software Update"',
    );
  });

  it("renders manual-copy guidance with the delivery controls", () => {
    const { body } = render(QuickControls, {
      props: {
        settings: { ...settings, autoInsert: false },
        section: "delivery",
        devices: [],
        onUpdate: async () => true,
        onOpenAudioSettings: () => {},
        onOpenDeliverySettings: () => {},
      },
    });
    expect(body).toContain("Direct input off uses manual copy.");
  });

  it.each(["file", "voice"] as const)(
    "shows a task-appropriate empty %s badge",
    (mode) => {
      const { body } = render(CurrentResult, {
        props: {
          resultKey: "empty",
          mode,
          text: "",
          working: false,
          canCopy: false,
          onCopy: async () => false,
          onClear: () => {},
        },
      });
      expect(body).toContain(
        mode === "file" ? "No transcript yet" : "Nothing recorded yet",
      );
      if (mode === "file") expect(body).not.toContain("Nothing recorded yet");
    },
  );
  it.each([undefined, TTSSource.SourceCompose])(
    "labels idle TTS as locally ready to generate (%s)",
    (source) => {
      const status: TTSStatus = { ...idleSpeech, source };
      const noop = () => {};
      const { body } = render(TextToSpeech, {
        props: {
          settings: {
            ...settings.textToSpeech,
            enabled: true,
            baseURL: "https://tts.test/v1",
            model: "speech",
            voice: "voice",
          },
          status,
          onSpeak: noop,
          onPause: noop,
          onResume: noop,
          onRestart: noop,
          onStop: noop,
          onSave: noop,
          onClear: noop,
          onOpenSettings: noop,
        },
      });
      expect(body).toContain("Ready to generate");
      expect(body).toContain("Local configuration only");
      expect(body).not.toContain("Reachable");
    },
  );
  it("offers an accessible settings action when speech is unconfigured", () => {
    const noop = () => {};
    const { body } = render(TextToSpeech, {
      props: {
        settings: { ...settings.textToSpeech, enabled: false },
        status: idleSpeech,
        onSpeak: noop,
        onPause: noop,
        onResume: noop,
        onRestart: noop,
        onStop: noop,
        onSave: noop,
        onClear: noop,
        onOpenSettings: noop,
      },
    });
    expect(body).toContain("Configure speech");
  });
});
