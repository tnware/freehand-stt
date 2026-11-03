import { describe, expect, it } from "vitest";
import { render } from "svelte/server";
import { Purpose } from "$bindings/savedconnection";
import { settings as fixtureSettings } from "$lib/stores/session-fixtures";
import ModelSelectionSummary from "./ModelSelectionSummary.svelte";

const renderSummary = (purpose: Purpose, draft = false) => {
  const settings = structuredClone(fixtureSettings);
  return {
    settings,
    html: () =>
      render(ModelSelectionSummary, { props: { settings, purpose, draft } })
        .body,
  };
};

describe("model selection summary", () => {
  it("distinguishes settings drafts from the active engine", () => {
    expect(renderSummary(Purpose.Voice, true).html()).toContain(
      "Selected engine and model",
    );
    expect(renderSummary(Purpose.Voice).html()).toContain(
      "Active engine and model",
    );
  });
  it("guides an unconfigured workflow without implying an engine is connected", () => {
    const view = renderSummary(Purpose.Voice);
    view.settings.savedConnections.selected = { voice: "" };
    expect(view.html()).toContain("Choose a connection");
    expect(view.html()).toContain("Then choose the model");
  });
  it("shows the selected workflow model without altering any settings", () => {
    const view = renderSummary(Purpose.Voice);
    view.settings.voiceTranscription.model = "my-voice-model";
    view.settings.savedConnections = {
      selected: { voice: "voice-server" },
      entries: [
        {
          id: "voice-server",
          name: "My server",
          builtIn: false,
          uses: [Purpose.Voice],
          hasCredential: false,
          details: view.settings.voiceTranscription,
        },
      ],
    };
    const before = structuredClone(view.settings);
    expect(view.html()).toContain("my-voice-model");
    expect(view.settings).toEqual(before);
  });
  it("keeps disabled cleanup clearly inactive", () => {
    const view = renderSummary(Purpose.Cleanup);
    view.settings.postProcessing.enabled = false;
    expect(view.html()).toContain("This workflow is off");
    expect(view.html()).toContain("Selected engine and model");
    expect(view.html()).not.toContain("Active engine and model");
  });
});
