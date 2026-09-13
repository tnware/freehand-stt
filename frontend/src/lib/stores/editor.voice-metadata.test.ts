import { readFileSync } from "node:fs";
import { expect, it } from "vitest";

// Source contract only: does not mount bits-ui or prove browser focus/popup
// persistence. It guards the owner prop that disabled the open combobox.
it("keeps Voice manual model entry unlocked during metadata discovery", () => {
  const voice = readFileSync(new URL("../components/home/VoiceTranscriptionSettings.svelte", import.meta.url), "utf8");
  const busy = voice.match(/const busy = \$derived\(([\s\S]*?)\);/)?.[1];
  expect(busy).toBeDefined();
  expect(busy).not.toContain("voiceConnectionTesting");
  expect(busy).toContain("disabled");
  expect(busy).toContain("editor.saving");
  expect(busy).toContain('editor.isQuickSettingsPending("voice-transcription")');
  expect(voice).toContain("disabled={busy || !connectionID}");
  expect(voice).toContain("busy={testing}");
  const picker = readFileSync(new URL("../components/settings/RuntimeModelPicker.svelte", import.meta.url), "utf8");
  expect(picker).toContain("const locked = $derived(disabled || choosing)");
  expect(picker).toContain("disabled={locked || busy}");
  expect(picker).toContain("disabled={locked}");
});
