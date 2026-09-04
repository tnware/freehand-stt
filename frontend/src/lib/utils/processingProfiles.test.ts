import { describe, expect, it } from "vitest";
import { PostProcessingPreset, type ProfileDescriptor } from "$lib/state";
import {
  instructionBytes,
  instructionError,
  processingProfileName,
  s1MiniControlLine,
} from "$lib/utils/processingProfiles";

const profiles: ProfileDescriptor[] = [
  {
    id: PostProcessingPreset.PostProcessingPresetGeneric,
    name: "Custom instruction",
    description: "Custom",
    instructionEditable: true,
  },
];

describe("processing profile presentation", () => {
  it("uses backend profile names and safe fallbacks", () => {
    expect(
      processingProfileName(profiles, PostProcessingPreset.PostProcessingPresetGeneric),
    ).toBe("Custom instruction");
    expect(
      processingProfileName(profiles, PostProcessingPreset.PostProcessingPresetS1Mini),
    ).toBe("S1-mini by Superwhisper");
    expect(processingProfileName(profiles, undefined)).toBe("Custom instruction");
  });

  it("validates the backend UTF-8 byte limit rather than JavaScript length", () => {
    expect(instructionBytes("é")).toBe(2);
    expect(instructionError("   ", 8192)).toContain("Enter an instruction");
    expect(instructionError("a".repeat(8192), 8192)).toBe("");
    expect(instructionError("é".repeat(4097), 8192)).toContain("8,192 UTF-8 bytes");
  });

  it("renders the exact S1-mini control-line shape", () => {
    expect(
      s1MiniControlLine({ styling: "formal", structure: "lists", context: "email" }),
    ).toBe("[Styling: formal] [Structure: lists] [Context: email]");
  });
});
