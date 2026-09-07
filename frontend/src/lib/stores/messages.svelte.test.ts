import { describe, expect, it } from "vitest";
import { SessionMessages } from "./messages.svelte";

describe("SessionMessages", () => {
  it("shows a caught error's message without its framework class prefix", () => {
    const messages = new SessionMessages();
    const error = new Error("The setting could not be saved.");
    error.name = "RuntimeError";
    messages.fail(error);
    expect(messages.error).toBe("The setting could not be saved.");
    messages.fail("Retry this action.");
    expect(messages.error).toBe("Retry this action.");
  });
  it("uses the visible error channel and clears a stale notice", () => {
    const messages = new SessionMessages();
    messages.notice = "Earlier confirmation";

    messages.reportFailure("Unable to close the window.");

    expect(messages.notice).toBe("");
    expect(messages.error).toBe("Unable to close the window.");
  });
});
