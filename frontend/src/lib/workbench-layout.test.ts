import { afterEach, describe, expect, it, vi } from "vitest";
import { WorkbenchLayout } from "./workbench-layout.svelte";

afterEach(() => vi.unstubAllGlobals());

describe("workbench layout preferences", () => {
  it("accepts only bounded layout preferences from storage", () => {
    vi.stubGlobal("localStorage", {
      getItem: () =>
        JSON.stringify({
          primaryOpen: false,
          bottomOpen: "false",
          secondaryOpen: true,
          tab: "unknown",
          bottomSize: 400,
          secondarySize: -100,
          outputInstanceID: "untrusted-runtime",
          diagnosticWorkflow: "file",
        }),
    });
    const layout = new WorkbenchLayout();
    layout.restore();
    expect(layout.primaryOpen).toBe(false);
    expect(layout.bottomOpen).toBe(true);
    expect(layout.tab).toBe("recent");
    expect(layout.bottomSize).toBe(55);
    expect(layout.secondarySize).toBe(30);
    expect(layout.outputInstanceID).toBe("");
    expect(layout.diagnosticWorkflow).toBe("voice");
  });

  it("persists layout without runtime identifiers or contributed content", () => {
    const setItem = vi.fn();
    vi.stubGlobal("localStorage", { setItem });
    const layout = new WorkbenchLayout();
    layout.outputInstanceID = "private-runtime";
    layout.tab = "output";
    layout.claimNotifications("modal");
    layout.claimNotifications("error");
    layout.save();
    expect(JSON.parse(setItem.mock.calls[0][1])).toEqual({
      primaryOpen: true,
      bottomOpen: true,
      secondaryOpen: true,
      tab: "output",
      bottomSize: 30,
      secondarySize: 40,
    });
  });

  it("keeps defaults when optional storage is malformed or unavailable", () => {
    vi.stubGlobal("localStorage", {
      getItem: () => "{",
      setItem: () => {
        throw Error("blocked");
      },
    });
    const layout = new WorkbenchLayout();
    expect(() => {
      layout.restore();
      layout.save();
    }).not.toThrow();
    expect(layout.tab).toBe("recent");
    expect(layout.bottomOpen).toBe(true);
  });
});

describe("workbench notification ownership", () => {
  it("keeps notifications paused until every modal releases its claim", () => {
    const layout = new WorkbenchLayout();
    const releaseFirst = layout.claimNotifications("modal");
    const releaseSecond = layout.claimNotifications("modal");
    expect(layout.notificationsPaused).toBe(true);

    releaseFirst();
    releaseFirst();
    expect(layout.notificationsPaused).toBe(true);

    releaseSecond();
    expect(layout.notificationsPaused).toBe(false);
    expect(layout.errorOwned).toBe(false);
  });

  it("returns errors to the remaining owner as dialogs and editors tear down", () => {
    const layout = new WorkbenchLayout();
    const releaseEditor = layout.claimNotifications("error");
    expect(layout.errorOwned).toBe(true);
    expect(layout.notificationsPaused).toBe(false);

    const releaseDialog = layout.claimNotifications("modal");
    releaseDialog();
    expect(layout.notificationsPaused).toBe(false);
    expect(layout.errorOwned).toBe(true);

    const releaseNextDialog = layout.claimNotifications("modal");
    releaseEditor();
    expect(layout.errorOwned).toBe(false);
    expect(layout.notificationsPaused).toBe(true);
    releaseNextDialog();
    expect(layout.notificationsPaused).toBe(false);

    const nextLayout = new WorkbenchLayout();
    expect(nextLayout.notificationsPaused).toBe(false);
    expect(nextLayout.errorOwned).toBe(false);
  });
});
