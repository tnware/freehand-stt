import { describe, expect, it } from "vitest";
import { permissionRows } from "./nativePermissions";
import type { PermissionStatus } from "$bindings/input";
const status: PermissionStatus = {
  required: true,
  microphone: "not-determined",
  accessibility: false,
  keyboard: false,
};

describe("permission actions", () => {
  it("pairs each permission with its functional explanation", () => {
    expect(
      permissionRows(status).map(({ kind, description }) => ({
        kind,
        description,
      })),
    ).toEqual([
      {
        kind: "microphone",
        description: "Record speech and preview microphone input.",
      },
      {
        kind: "accessibility",
        description:
          "Check the destination, insert text, and capture shortcuts.",
      },
      {
        kind: "keyboard",
        description: "Capture shortcuts and use hold-to-talk.",
      },
    ]);
  });

  it("offers an explicit request only for undecided microphone access", () => {
    expect(permissionRows(status)[0]).toMatchObject({
      kind: "microphone",
      action: "request",
      label: "Microphone",
    });
    expect(
      permissionRows({ ...status, microphone: "denied" })[0],
    ).toMatchObject({ action: "settings", state: "Denied" });
    expect(
      permissionRows({ ...status, microphone: "restricted" })[0],
    ).toMatchObject({ action: "settings", state: "Restricted" });
  });
  it("keeps authorized access passive and each recovery action scoped", () => {
    expect(
      permissionRows({
        ...status,
        microphone: "authorized",
        accessibility: true,
        keyboard: true,
      }).every((row) => row.action === null),
    ).toBe(true);
    expect(
      permissionRows(status)
        .slice(1)
        .map((row) => row.kind),
    ).toEqual(["accessibility", "keyboard"]);
    expect(permissionRows({ ...status, required: false })).toEqual([]);
    expect(permissionRows(null)).toEqual([]);
  });
});
