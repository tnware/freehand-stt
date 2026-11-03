import { describe, expect, it, vi } from "vitest";
import { render } from "svelte/server";
import { ProviderID, type Model } from "$bindings/managedruntime";
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
    expect(action).not.toHaveBeenCalled();
  });
});
