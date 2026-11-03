import { describe, expect, it } from "vitest";
import { Purpose, type Connection } from "$bindings/savedconnection";
import { ID } from "$bindings/compatibility";
import { AuthenticationMode } from "$bindings/config";
import { runtimeDeletionReferences } from "./runtimeDeletion";

describe("runtime deletion dependencies", () => {
  it("identifies active workflows and custom references without asking to delete built-ins", () => {
    const template: Connection = {
      id: "",
      name: "",
      builtIn: false,
      uses: [],
      hasCredential: false,
      details: {
        compatibilityProfile: ID.Generic,
        baseURL: "",
        authenticationMode: AuthenticationMode.AuthenticationModeNone,
        allowInsecureHTTP: false,
        healthPath: "",
        headers: {},
      },
    };
    const catalog = {
      entries: [
        {
          ...template,
          id: "built",
          name: "Built-in",
          builtIn: true,
          details: { ...template.details, managedInstanceID: "one" },
        },
        {
          ...template,
          id: "custom",
          name: "Custom speech",
          builtIn: false,
          details: { ...template.details, managedInstanceID: "one" },
        },
        {
          ...template,
          id: "other",
          name: "Other runtime",
          builtIn: false,
          details: { ...template.details, managedInstanceID: "two" },
        },
      ],
      selected: {
        [Purpose.Voice]: "built",
        [Purpose.Speech]: "custom",
        [Purpose.Transcription]: "other",
      },
    };
    const before = structuredClone(catalog);
    expect(runtimeDeletionReferences(catalog, "one")).toEqual({
      workflows: ["Voice transcription", "Text to speech"],
      connections: ["Custom speech"],
    });
    expect(catalog).toEqual(before);
    expect(runtimeDeletionReferences(catalog, "missing")).toEqual({
      workflows: [],
      connections: [],
    });
  });
  it("does not invent dependencies with unavailable metadata or no instance", () => {
    expect(runtimeDeletionReferences(undefined, "one")).toEqual({
      workflows: [],
      connections: [],
    });
    expect(
      runtimeDeletionReferences({ entries: [], selected: {} }, ""),
    ).toEqual({ workflows: [], connections: [] });
  });
});
