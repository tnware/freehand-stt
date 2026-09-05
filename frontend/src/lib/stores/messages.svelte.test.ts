import { describe, expect, it } from "vitest";
import { SessionMessages } from "./messages.svelte";

describe("SessionMessages", () => {
  it("uses the visible error channel and clears a stale notice", () => {
    const messages = new SessionMessages();
    messages.notice = "Earlier confirmation";

    messages.reportFailure("Unable to close the window.");

    expect(messages.notice).toBe("");
    expect(messages.error).toBe("Unable to close the window.");
  });
});
