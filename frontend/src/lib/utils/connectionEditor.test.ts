import { describe, expect, it } from "vitest";
import { renameConnectionHeader, savedConnectionID } from "./connectionEditor";

describe("connection header editing", () => {
  it.each(["X-Other", "x-other", "X-OTHER"])(
    "rejects duplicate %s without mutating either row",
    (next) => {
      const headers = { "X-First": "first", "X-Other": "other" };
      expect(renameConnectionHeader(headers, "X-First", next)).toBeNull();
      expect(headers).toEqual({ "X-First": "first", "X-Other": "other" });
    },
  );
  it("allows changing the case of the current name and preserves row order", () => {
    expect(
      renameConnectionHeader(
        { "X-First": "first", "X-Other": "other" },
        "X-First",
        "x-first",
      ),
    ).toEqual({ "x-first": "first", "X-Other": "other" });
  });
});

describe("saved connection identity", () => {
  const entries = [
    { id: "builtin", builtIn: true, name: "Local speech" },
    { id: "manual", builtIn: false, name: "Local speech" },
  ];
  it("returns the edited manual ID even when a built-in has the same name", () => {
    expect(
      savedConnectionID(entries, "manual", new Set(["builtin", "manual"])),
    ).toBe("manual");
  });
  it("returns the newly added manual ID, ignoring built-in additions", () => {
    expect(savedConnectionID(entries, undefined, new Set())).toBe("manual");
  });
  it("does not guess an identity when no unique new manual entry exists", () => {
    expect(
      savedConnectionID(entries, undefined, new Set(["manual"])),
    ).toBeUndefined();
    expect(
      savedConnectionID(
        [...entries, { id: "other", builtIn: false }],
        undefined,
        new Set(),
      ),
    ).toBeUndefined();
  });
});
