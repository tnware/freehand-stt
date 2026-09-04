import { describe, expect, it } from "vitest";
import { compareTranscriptText } from "$lib/utils/textDiff";

describe("compareTranscriptText", () => {
  it("marks removed raw words and added processed words", () => {
    const comparison = compareTranscriptText(
      "So actually let's deploy on Tuesday.",
      "Let's deploy on Thursday.",
    );

    expect(comparison.highlighted).toBe(true);
    expect(
      comparison.raw.filter((part) => part.kind === "removed").map((part) => part.text).join(""),
    ).toContain("actually");
    expect(
      comparison.processed.filter((part) => part.kind === "added").map((part) => part.text).join(""),
    ).toContain("Thursday");
    expect(comparison.raw.map((part) => part.text).join("")).toBe(
      "So actually let's deploy on Tuesday.",
    );
    expect(comparison.processed.map((part) => part.text).join("")).toBe(
      "Let's deploy on Thursday.",
    );
  });

  it("returns unchanged text as a single equal segment", () => {
    const comparison = compareTranscriptText("Already clean.", "Already clean.");

    expect(comparison.raw).toEqual([{ text: "Already clean.", kind: "equal" }]);
    expect(comparison.processed).toEqual([{ text: "Already clean.", kind: "equal" }]);
  });

  it("highlights sparse changes in a long transcript", () => {
    const words = Array.from({ length: 2_000 }, (_, index) => `word-${index}`);
    const raw = words.join(" ");
    const cleanedWords = [...words];
    cleanedWords[250] = "corrected-250";
    cleanedWords[1_000] = "corrected-1000";
    cleanedWords[1_750] = "corrected-1750";
    const processed = cleanedWords.join(" ");
    const comparison = compareTranscriptText(raw, processed);

    expect(comparison.highlighted).toBe(true);
    expect(comparison.raw.map((part) => part.text).join("")).toBe(raw);
    expect(comparison.processed.map((part) => part.text).join("")).toBe(processed);
    expect(
      comparison.processed.filter((part) => part.kind === "added").map((part) => part.text),
    ).toEqual(["corrected-250", "corrected-1000", "corrected-1750"]);
  });

  it("preserves whitespace and punctuation exactly", () => {
    const raw = "Wait...  meet me\nTuesday at 9?";
    const processed = "Wait—meet me\nWednesday at 9:00.";
    const comparison = compareTranscriptText(raw, processed);

    expect(comparison.highlighted).toBe(true);
    expect(comparison.raw.map((part) => part.text).join("")).toBe(raw);
    expect(comparison.processed.map((part) => part.text).join("")).toBe(processed);
  });

  it("falls back without highlights when divergent input exceeds the edit budget", () => {
    const raw = Array.from({ length: 1_000 }, (_, index) => `raw-${index}`).join(" ");
    const processed = Array.from({ length: 1_000 }, (_, index) => `clean-${index}`).join(" ");
    const comparison = compareTranscriptText(raw, processed);

    expect(comparison.highlighted).toBe(false);
    expect(comparison.raw).toEqual([{ text: raw, kind: "equal" }]);
    expect(comparison.processed).toEqual([{ text: processed, kind: "equal" }]);
  });
});
