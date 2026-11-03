import { describe, expect, it } from "vitest";
import { ProviderID, type BinaryOptions } from "$bindings/managedruntime";
import { runtimeBinaryContext } from "./runtimeBinaryContext";

const metal: BinaryOptions = {
  provider: ProviderID.LlamaCPP,
  os: "darwin",
  architecture: "arm64",
  supported: true,
  recommendedBackend: "metal",
  reason: "Apple Silicon GPU available.",
  options: [
    {
      backend: "cpu",
      supported: true,
      available: true,
      reason: "CPU available",
    },
    {
      backend: "metal",
      supported: true,
      available: true,
      reason: "Metal available",
    },
  ],
};

describe("runtime binary presentation", () => {
  it("keeps installed CPU separate from advisory Metal recommendation without modifying either source", () => {
    const status = {
      backend: "cpu",
      version: "installed-v1",
      state: "running",
    };
    const before = structuredClone({ status, metal });
    const result = runtimeBinaryContext(status, metal);
    expect(result).toMatchObject({
      current: "CPU",
      currentID: "cpu",
      currentLabel: "Running with",
      version: "installed-v1",
      recommendation: "Apple GPU (Metal)",
    });
    expect(
      result.choices.map((choice) => [
        choice.backend,
        choice.current,
        choice.recommended,
      ]),
    ).toEqual([
      ["cpu", true, false],
      ["metal", false, true],
    ]);
    expect({ status, metal }).toEqual(before);
  });
  it("does not claim a recommended backend or version is installed", () => {
    const result = runtimeBinaryContext(undefined, metal);
    expect(result.current).toBe("Not installed");
    expect(result.currentID).toBe("");
    expect(result.version).toBe("");
    expect(result.choices.every((choice) => !choice.current)).toBe(true);
    expect(
      runtimeBinaryContext(
        { backend: "cpu", version: "old", state: "not_installed" },
        metal,
      ).currentID,
    ).toBe("");
  });
  it("labels stopped installed binaries without claiming active execution", () => {
    expect(
      runtimeBinaryContext(
        { backend: "metal", version: "v1", state: "stopped" },
        metal,
      ),
    ).toMatchObject({
      currentLabel: "Installed backend",
      current: "Apple GPU (Metal)",
    });
  });
  it("never promotes an unavailable recommendation or substitutes it for installed CPU", () => {
    for (const options of [
      { ...metal, supported: false },
      {
        ...metal,
        options: metal.options!.map((option) => ({
          ...option,
          available: option.backend === "cpu",
        })),
      },
      { ...metal, recommendedBackend: "missing" },
    ]) {
      const result = runtimeBinaryContext(
        { backend: "cpu", version: "v1", state: "installed" },
        options,
      );
      expect(result.current).toBe("CPU");
      expect(result.recommendation).toBe("");
    }
  });
});
