import { describe, expect, it } from "vitest";
import { settingsValidationIssue } from "./settingsValidation";

const failure = (field: string, message = "Check this setting.") =>
  new Error(message, {
    cause: { kind: "settings_validation", field, message },
  });

describe("Go validation presentation targets", () => {
  it.each([
    ["maxDurationSeconds", "audio", "max-duration"],
    ["vadActivitySilenceMilliseconds", "audio", "vad-activity-silence"],
    ["speechPaddingMilliseconds", "audio", "speech-padding"],
    ["autoStopSilenceMilliseconds", "audio", "automatic-stop-silence"],
    [
      "autoStopMinimumSpeechMilliseconds",
      "audio",
      "automatic-stop-minimum-speech",
    ],
    ["segmentSeconds", "audio", "segment-duration"],
    ["segmentSilenceMilliseconds", "audio", "segment-silence"],
    ["transcriptionTimeoutSeconds", "server", "transcription-timeout"],
    ["fileTranscriptionTimeoutSeconds", "server", "file-transcription-timeout"],
    ["postProcessing.timeoutSeconds", "processing", "cleanup-timeout"],
    [
      "postProcessing.systemPrompt",
      "processing",
      "post-processing-custom-instruction",
    ],
    ["textToSpeech.timeoutSeconds", "speech", "tts-timeout"],
    ["textToSpeech.voice", "speech", "tts-voice"],
    ["textToSpeech.speed", "speech", "tts-speed"],
    ["toggleShortcut", "shortcuts", null],
    ["overlayEnabled", "overlay", null],
    ["appearanceMode", "general", null],
  ])("maps %s to its settings surface", (field, section, control) => {
    expect(settingsValidationIssue(failure(field!))).toEqual({
      field,
      section,
      control,
      message: "Check this setting.",
    });
  });

  it("uses Go's message rather than deriving a range or parsing error text", () => {
    expect(
      settingsValidationIssue(
        failure("maxDurationSeconds", "Limit differs with segmentation."),
      )?.message,
    ).toBe("Limit differs with segmentation.");
  });

  it.each([
    "unknown",
    "__proto__",
    "toString",
    '#max-duration, input[type="password"]',
  ])("never turns unknown field %s into a selector", (field) => {
    expect(settingsValidationIssue(failure(field))).toBeNull();
  });
  it.each([
    null,
    "failure",
    new Error("ordinary error"),
    new Error("malformed", {
      cause: { field: "maxDurationSeconds", message: "Bad" },
    }),
  ])("leaves unstructured failures to the ordinary error channel", (error) => {
    expect(settingsValidationIssue(error)).toBeNull();
  });
});
