import { describe, expect, it } from "vitest";
import { orderMessages, type Message } from "$lib/utils/messages";

describe("orderMessages", () => {
  it("presents system state before action outcomes while preserving local order", () => {
    const messages: Message[] = [
      { id: "success", tone: "success", source: "action", text: "Transcript copied." },
      { id: "microphone", tone: "info", source: "system", text: "Microphone removed." },
      { id: "error", tone: "error", source: "action", text: "Endpoint rejected the request." },
      { id: "reveal", tone: "info", source: "system", text: "Existing window revealed." },
    ];

    expect(orderMessages(messages).map((message) => message.id)).toEqual([
      "microphone",
      "reveal",
      "success",
      "error",
    ]);
  });
});
