import { describe, expect, it } from "vitest";
import {
  ShellNavigation,
  saveConnectionAndContinue,
  acceptConnectionClose,
} from "./shell-navigation.svelte";
import { Purpose } from "$bindings/savedconnection";

describe("dedicated Settings navigation", () => {
  it("rejects a reveal before replacing the accepted task origin", () => {
    const shell = new ShellNavigation();
    expect(
      shell.acceptRequest({ section: "speech", origin: "tts" }, false),
    ).toBe(true);
    expect(shell.acceptRequest({ section: "general", origin: "" }, true)).toBe(
      false,
    );
    expect(shell.origin).toBe("tts");
    expect(shell.active).toBe("speech");
    expect(shell.acceptRequest({ section: "general", origin: "" }, false)).toBe(
      true,
    );
    expect(shell.origin).toBe("");
    expect(shell.saveReturnsToTask).toBe(false);
  });
  it("connection setup stays in Settings before returning to a task", () => {
    const shell = new ShellNavigation();
    shell.acceptRequest(
      {
        section: "connections",
        origin: "tts",
        connection: { id: "", purpose: Purpose.Speech, create: true },
      },
      false,
    );
    shell.returnFromConnection(Purpose.Speech);
    expect(shell.active).toBe("speech");
    expect(shell.open).toBe(true);
    expect(shell.saveReturnsToTask).toBe(true);
  });
  it("routes direct catalog use to its purpose instead of a stale task", () => {
    const shell = new ShellNavigation();
    shell.openSettings("processing", "voice");
    shell.openConnection({ id: "", purpose: Purpose.Speech, create: true });
    shell.returnFromConnection(Purpose.Speech);
    expect(shell.active).toBe("processing");
    shell.openSettings("connections");
    shell.returnFromConnection(Purpose.Speech);
    expect(shell.active).toBe("speech");
    expect(shell.origin).toBe("voice");
  });
  it("routes a first direct catalog visit to Speech", () => {
    const shell = new ShellNavigation();
    shell.openSettings("connections", "tts");
    shell.returnFromConnection(Purpose.Speech);
    expect(shell.active).toBe("speech");
  });
  it("returns connection setup to the original task and Done to its workspace", () => {
    const shell = new ShellNavigation();
    shell.openSettings("speech", "tts");
    shell.openConnection({ id: "", purpose: Purpose.Speech, create: true });
    shell.returnFromConnection(Purpose.Speech);
    expect(shell.active).toBe("speech");
    expect(shell.origin).toBe("tts");
    shell.done();
    expect(shell.open).toBe(false);
    expect(shell.origin).toBe("");
    expect(shell.saveReturnsToTask).toBe(false);
  });
});
describe("connection close Save", () => {
  it("does not retain a rejected close while a deferred managed check runs", async () => {
    const editor = { saving: false, managedConnectionTesting: true };
    let finish!: () => void;
    const check = new Promise<void>((resolve) => {
      finish = resolve;
    });
    let afterConnection: (() => void) | null = null;
    const calls: string[] = [];
    const accepted = acceptConnectionClose(editor, () => {
      afterConnection = () => calls.push("hide");
      calls.push("request close");
    });
    expect(accepted).toBe(false);
    finish();
    await check;
    editor.managedConnectionTesting = false;
    await saveConnectionAndContinue(
      async () => true,
      Purpose.Speech,
      () => {
        calls.push("saved");
        afterConnection?.();
      },
    );
    expect(calls).toEqual(["saved"]);
    expect(acceptConnectionClose(editor, () => calls.push("new close"))).toBe(
      true,
    );
    expect(calls).toEqual(["saved", "new close"]);
  });
  it("rejects close during save before storing its continuation", () => {
    let stored = false;
    expect(
      acceptConnectionClose(
        { saving: true, managedConnectionTesting: false },
        () => {
          stored = true;
        },
      ),
    ).toBe(false);
    expect(stored).toBe(false);
  });
  it("preserves activation purpose and only continues after a successful transaction", async () => {
    const calls: unknown[] = [];
    await saveConnectionAndContinue(
      async (purpose) => {
        calls.push(purpose);
        return false;
      },
      Purpose.Speech,
      () => calls.push("leave"),
    );
    expect(calls).toEqual([Purpose.Speech]);
    await saveConnectionAndContinue(
      async (purpose) => {
        calls.push(purpose);
        return true;
      },
      Purpose.Speech,
      () => calls.push("leave"),
    );
    expect(calls).toEqual([Purpose.Speech, Purpose.Speech, "leave"]);
  });
});
