import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

const read = (file: string) => readFileSync(new URL(`../components/${file}.svelte`, import.meta.url), "utf8");

describe("non-Voice picker metadata wiring", () => {
  it.each([["Transcription", "stt"], ["Cleanup", "processing"]])("routes Home %s entry through narrow callbacks", (role, prefix) => {
    const owner = read("home/ResultQuickSettings");
    expect(owner).toContain(`ensureConnectionMetadata(Purpose.${role}, true)`);
    expect(owner).toMatch(new RegExp(`connectionMetadataStatus\\(\\s*Purpose\\.${role},?\\s*\\)`));
    const quick = read("home/QuickSettings");
    expect(quick).toContain(`onEnter={onEnter${role}}`);
    expect(quick).toContain(`metadataStatus={${prefix}MetadataStatus}`);
    expect(quick).toContain(`${prefix}MetadataStatus = "idle"`);
    expect(quick).toContain(`onEnter${role}?: () => void`);
  });
  it("wires the Home readiness transcription picker too", () => {
    const source = read("home/HomeScreen");
    const block = source.split("<QuickSettings")[1].split("/>")[0];
    expect(block).toMatch(/ensureConnectionMetadata\(\s*Purpose\.Transcription,\s*true,?\s*\)/);
    expect(block).toMatch(/sttMetadataStatus=\{session\.editor\.connectionMetadataStatus\(\s*Purpose\.Transcription,?\s*\)\}/);
  });
  it("routes Home Speech entry to applied shared discovery", () => {
    const source = read("home/SpeechQuickSettings");
    expect(source).toContain("ensureConnectionMetadata(Purpose.Speech, true)");
    expect(source).toContain("metadataStatus={editor.connectionMetadataStatus(Purpose.Speech)}");
  });
  it.each([
    ["ServerSection", "Transcription"],
    ["ProcessingSection", "Cleanup"],
    ["SpeechSection", "Speech"],
  ])("threads safe entry and shared status through %s", (section, role) => {
    const screen = read("settings/SettingsScreen");
    const block = screen.split(`<${section}`)[1].split("/>")[0];
    expect(block).toMatch(new RegExp(`ensureConnectionMetadata\\(\\s*Purpose\\.${role},\\s*true,?\\s*\\)`));
    expect(block).toMatch(new RegExp(`connectionMetadataStatus\\(\\s*Purpose\\.${role},?\\s*\\)`));
    const source = read(`settings/sections/${section}`);
    expect(source).toContain("onEnter?: () => void");
    expect(source).toContain('metadataStatus = "idle"');
    expect(source).toContain("{onEnter}");
    expect(source).toContain("{metadataStatus}");
  });
  it("forwards speech entry without coupling reusable controls to an editor", () => {
    const source = read("settings/SpeechModelControls");
    expect(source).toContain("onEnter?: () => void");
    expect(source).toContain('metadataStatus = "idle"');
    expect(source).toContain("{onEnter}");
    expect(source).toContain("{metadataStatus}");
  });
});
