import { describe, expect, it } from "vitest";
import { render } from "svelte/server";
import {
  ModelAcquisitionMethod,
  type ModelSource,
} from "$bindings/managedruntime";
import RuntimeModelDetails from "./RuntimeModelDetails.svelte";

const source: ModelSource = {
  acquisitionMethod: ModelAcquisitionMethod.FreehandHuggingFace,
  metadataOrigin: "fixture",
  repository: "publisher/model",
  repositoryURL: "https://huggingface.co/publisher/model",
  revision: "model-revision",
  filename: "model.gguf",
  sha256: "model-checksum",
  companions: [
    {
      acquisitionMethod: ModelAcquisitionMethod.FreehandHuggingFace,
      metadataOrigin: "fixture",
      repository: "publisher/companion",
      revision: "companion-revision",
      filename: "companion.bin",
      sha256: "companion-checksum",
    },
  ],
};

describe("catalog model information", () => {
  it("shows useful facts before the technical disclosure and preserves every download pin", () => {
    const html = render(RuntimeModelDetails, {
      props: {
        modelID: "small.en",
        variant: "English only · Standard file",
        source,
      },
    }).body;
    const overview = html.slice(0, html.indexOf("<details"));
    for (const fact of [
      "Model ID",
      "small.en",
      "English only · Standard file",
      "publisher/model",
      "model.gguf",
      "Includes 1 companion files",
    ])
      expect(overview).toContain(fact);
    expect(overview).toContain('href="https://huggingface.co/publisher/model"');
    expect(html).toContain("Model download details");
    for (const pin of [
      "model-revision",
      "model-checksum",
      "companion.bin",
      "publisher/companion",
      "companion-revision",
      "companion-checksum",
    ])
      expect(html).toContain(pin);
  });
  it("keeps delegated acquisition truthful and does not invent a download link", () => {
    const html = render(RuntimeModelDetails, {
      props: {
        modelID: "nemotron",
        source: {
          ...source,
          acquisitionMethod: ModelAcquisitionMethod.NeMoModelManager,
          repositoryURL: undefined,
          companions: [],
        },
      },
    }).body;
    expect(html).toContain("NeMo’s built-in model manager");
    expect(html).toContain("Index repository");
    expect(html).not.toContain("Downloaded by Freehand");
    expect(html).not.toContain("<a ");
  });
  it("does not invent source details when no metadata exists", () => {
    const html = render(RuntimeModelDetails, {
      props: { modelID: "unknown" },
    }).body;
    expect(html).toContain("unknown");
    expect(html).not.toContain("Model download details");
    expect(html).not.toContain("Source");
  });
});
