import { readFileSync } from "node:fs";
import { render } from "svelte/server";
import { describe, expect, it } from "vitest";
import CurrentResult from "$lib/components/home/CurrentResult.svelte";
import TextToSpeech from "$lib/components/home/TextToSpeech.svelte";
import { TTSPhase, TTSSource, type TTSStatus } from "$lib/state";
import { settings } from "$lib/stores/session-fixtures";

const read = (path: string) =>
  readFileSync(new URL(path, import.meta.url), "utf8");

describe("home task presentation", () => {
  it("uses Freehand as the accessible document and updater title", () => {
    expect(read("../../index.html")).toContain("<title>Freehand</title>");
    expect(read("../../../internal/app/app.go")).toContain(
      'Title: "Freehand — Software Update"',
    );
  });

  it("uses stable shared transcription and delivery labels", () => {
    expect(read("components/home/QuickSettings.svelte")).toContain(
      'label="Transcription"',
    );
    expect(read("components/home/ResultQuickSettings.svelte")).toContain(
      "Transcription settings",
    );
    expect(read("components/home/QuickControls.svelte")).toContain(
      "Direct input off uses manual copy.",
    );
    expect(read("components/home/TextToSpeech.svelte")).toContain(
      "Text to speech settings",
    );
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
      const status: TTSStatus = {
        generation: 0,
        phase: TTSPhase.Idle,
        source,
        positionMilliseconds: 0,
        durationMilliseconds: 0,
        canPause: false,
        canResume: false,
        canRestart: false,
        canStop: false,
        canSave: false,
        canClear: false,
      };
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
  it("wires the selected task and editor metadata into the footer", () => {
    const app = read("../App.svelte");
    expect(app).toMatch(
      /taskConnectionStatus\(inputMode,\s*session.editor,\s*now\)/,
    );
    expect(app).toContain("connectionState={footerStatus}");
  });
});
