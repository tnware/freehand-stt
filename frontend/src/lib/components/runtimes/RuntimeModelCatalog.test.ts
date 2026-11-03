import { describe, expect, it, vi } from "vitest";
import { render } from "svelte/server";
import {
  ProviderID,
  type Model,
  type InstanceStatus,
} from "$bindings/managedruntime";
import { Role, ID as CompatibilityID } from "$bindings/compatibility";
import { ID } from "$bindings/modelprofile";
import { settings } from "$lib/stores/session-fixtures";
import { runtimePresentation } from "$lib/utils/managedRuntime";
import RuntimeModelCatalog from "./RuntimeModelCatalog.svelte";

describe("unavailable runtime model catalog", () => {
  it("keeps browsing read-only and never recommends installing an unsupported runtime", () => {
    const model: Model = {
      id: "small",
      name: "Whisper Small",
      description: "Completed transcription",
      sizeBytes: 1,
      installed: false,
      recommended: true,
      realtime: false,
      profile: "generic",
      contracts: [],
    };
    const action = vi.fn();
    const html = render(RuntimeModelCatalog, {
      props: {
        models: [model],
        row: undefined,
        entry: {
          id: ProviderID.WhisperCPP,
          name: "whisper.cpp",
          version: "fixture",
          supported: false,
          unavailableReason: "No managed macOS binary is available.",
          backends: [],
          models: [model],
        },
        installed: false,
        locked: false,
        busy: false,
        cancelling: false,
        view: runtimePresentation(undefined),
        onRefresh: action,
        onGet: action,
        onSelect: action,
        onRemove: action,
        onCancel: action,
        onDisableSpeech: action,
      },
    }).body;
    expect(html).toContain("This runtime is unavailable on this machine");
    expect(html).not.toContain("Install this runtime first");
    expect(html).not.toContain("Install the runtime to download");
    const buttons = html.match(/<button\b[^>]*>[\s\S]*?<\/button>/g) ?? [];
    for (const label of ["Refresh catalog", "Download", "Select"]) {
      const button = buttons.find(
        (candidate) => candidate.replace(/<[^>]*>/g, "").trim() === label,
      );
      expect(button, label).toBeDefined();
      expect(button?.split(">")[0]).toMatch(/\bdisabled\b/);
      expect(button?.split(">")[0]).toContain(
        "No managed macOS binary is available.",
      );
    }
    // Models are a named section, separate from the surrounding runtime settings.
    const headingID = /<section[^>]*aria-labelledby="([^"]+)"/.exec(html)?.[1];
    expect(headingID).toBeDefined();
    expect(html).toContain(`<h3 id="${headingID}"`);
    // Family and model disclosure controls keep separate, connected children.
    const family = buttons.find((button) =>
      button.includes('aria-label="Small models"'),
    );
    const details = buttons.find((button) =>
      button.includes('aria-label="Model details: Whisper Small"'),
    );
    expect(family).toContain('aria-expanded="true"');
    expect(details).toContain('aria-expanded="false"');
    const familyID = /aria-controls="([^"]+)"/.exec(family ?? "")?.[1];
    const detailsID = /aria-controls="([^"]+)"/.exec(details ?? "")?.[1];
    expect(familyID).toBeDefined();
    expect(detailsID).toBeDefined();
    expect(familyID).not.toEqual(detailsID);
    expect(html).toContain(`id="${familyID}"`);
    expect(html).toContain(`id="${detailsID}" hidden`);
    expect(action).not.toHaveBeenCalled();
  });
});

it.each(["stopped", "running"])(
  "describes speech deselection accurately while %s and preserves the running guard",
  (state) => {
    const model: Model = {
      id: "magpie-tts",
      name: "Magpie",
      description: "Speech synthesis",
      sizeBytes: 1,
      installed: true,
      recommended: false,
      realtime: false,
      profile: ID.MagpieTTS,
      contracts: [
        {
          role: Role.Speech,
          compatibilityProfile: CompatibilityID.NeMoSpeechV1,
          modelProfile: ID.MagpieTTS,
          behavior: settings.modelProfiles.speech![0],
        },
      ],
    };
    const row: InstanceStatus = {
      instance: {
        id: "nemo",
        name: "NeMo",
        provider: ProviderID.NeMoSpeechCPP,
        model: "asr",
        speechModel: model.id,
        autoStart: false,
      },
      activeModel: "",
      status: {
        acquisition: { phase: "", bytes: 0, totalBytes: 0 },
        operation: { id: 0, kind: "", model: "", outcome: "", error: "" },
        supported: true,
        state,
        enabled: true,
        selectedModel: "asr",
        selectedSpeechModel: model.id,
        realtime: false,
        backend: "cpu",
        version: "fixture",
        progress: 0,
        phase: "",
        error: "",
        models: [model],
      },
    };
    const action = vi.fn();
    const html = render(RuntimeModelCatalog, {
      props: {
        models: [model],
        row,
        entry: {
          id: ProviderID.NeMoSpeechCPP,
          name: "NeMo",
          version: "fixture",
          supported: true,
          backends: [],
          models: [model],
        },
        installed: true,
        locked: false,
        busy: false,
        cancelling: false,
        view: runtimePresentation(row.status),
        onRefresh: action,
        onGet: action,
        onSelect: action,
        onRemove: action,
        onCancel: action,
        onDisableSpeech: action,
      },
    }).body;
    const disable = (
      html.match(/<button\b[^>]*>[\s\S]*?<\/button>/g) ?? []
    ).find((button) => button.replace(/<[^>]*>/g, "").trim() === "Disable");
    expect(disable).toBeDefined();
    expect(disable).toContain(
      'title="Deselect this speech model. Downloaded files are kept."',
    );
    expect(disable).not.toContain("Text to speech first");
    expect(/\sdisabled(?:\s|=|$)/.test(disable!.split(">")[0])).toBe(
      state === "running",
    );
    expect(action).not.toHaveBeenCalled();
  },
);
