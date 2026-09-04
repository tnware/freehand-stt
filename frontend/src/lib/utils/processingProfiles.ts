import { PostProcessingPreset, type ProfileDescriptor } from "$lib/state";

export function processingProfile(
  profiles: ProfileDescriptor[],
  id: PostProcessingPreset | string | undefined,
): ProfileDescriptor | undefined {
  return profiles.find((profile) => profile.id === id);
}

export function processingProfileName(
  profiles: ProfileDescriptor[],
  id: PostProcessingPreset | string | undefined,
): string {
  return (
    processingProfile(profiles, id)?.name ??
    (id === PostProcessingPreset.PostProcessingPresetS1Mini
      ? "S1-mini by Superwhisper"
      : id === PostProcessingPreset.PostProcessingPresetGeneric || !id
        ? "Custom instruction"
        : id)
  );
}

export function instructionBytes(instruction: string): number {
  return new TextEncoder().encode(instruction).length;
}

export function instructionError(instruction: string, maximumBytes: number): string {
  if (!instruction.trim()) return "Enter an instruction before saving this profile.";
  if (maximumBytes > 0 && instructionBytes(instruction) > maximumBytes) {
    return `Keep the instruction at or below ${maximumBytes.toLocaleString()} UTF-8 bytes.`;
  }
  return "";
}

export function s1MiniControlLine(processor: {
  styling: string;
  structure: string;
  context: string;
}): string {
  return `[Styling: ${processor.styling}] [Structure: ${processor.structure}] [Context: ${processor.context}]`;
}
