import { readFileSync } from "node:fs";
import { expect, it } from "vitest";

it("routes picker entry and Voice inventory through the shared metadata owner", () => {
  const picker = readFileSync(
    new URL(
      "../components/settings/RuntimeModelPicker.svelte",
      import.meta.url,
    ),
    "utf8",
  );
  const voice = readFileSync(
    new URL(
      "../components/home/VoiceTranscriptionSettings.svelte",
      import.meta.url,
    ),
    "utf8",
  );
  expect(picker).toContain("onEnter?.()");
  expect(picker).toContain("Loading model list");
  expect(picker).toContain("Could not load the model list");
  expect(voice).not.toContain("ConnectionService");
  expect(voice).not.toContain("testedID");
  expect(voice).toContain("editor.connectionMetadataStatus(Purpose.Voice)");
  expect(voice).not.toContain("Choose the model loaded by this server");
});
