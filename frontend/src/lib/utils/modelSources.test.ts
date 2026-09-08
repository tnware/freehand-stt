import { describe, expect, it } from "vitest";
import { modelSources } from "./modelSources";

describe("model provenance labels", () => {
  it("keeps saved settings distinct from current server discovery", () => {
    expect(modelSources("one", ["one"], ["one"])).toBe("Saved · Server");
    expect(modelSources("one", [], ["one"])).toBe("Saved");
    expect(modelSources("one", ["one"], [])).toBe("Server");
    expect(modelSources("one", [], [], ["one"])).toBe("Edited");
  });
  it("does not label an unlisted current or manually entered model as a server result", () => {
    expect(modelSources("private/model", ["other/model"], [])).toBe("Manual ID");
    expect(modelSources("ONE", ["one"], [])).toBe("Manual ID");
  });
});
