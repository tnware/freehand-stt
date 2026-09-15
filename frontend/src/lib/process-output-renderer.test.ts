import { expect, it } from "vitest";
import type { OutputChunk } from "$bindings/managedruntime/models";
import {
  ProcessOutputRenderer,
  type OutputTerminal,
} from "./process-output-renderer";

// The only fake is xterm's asynchronous parsing boundary. Deliberately deliver
// callbacks even after disposal: an old acknowledgement must not revive output.
class DelayedTerminal implements OutputTerminal {
  content = "";
  disposed = false;
  pending: { text: string; done: () => void }[] = [];
  write(text: string, done: () => void) {
    this.pending.push({ text, done });
  }
  acknowledge() {
    const next = this.pending.shift();
    if (!next) throw new Error("No pending write");
    if (!this.disposed) this.content += next.text;
    next.done();
  }
  dispose() {
    this.disposed = true;
    this.content = "";
  }
}
const chunk = (sequence: number, text = `${sequence}\n`): OutputChunk => ({
  sequence,
  text,
  timestamp: 0,
  stream: "stdout",
});

it("bounds pending writes and forgets evicted, cleared and revoked output before late acknowledgements", () => {
  const terminals: DelayedTerminal[] = [];
  const renderer = new ProcessOutputRenderer(() => {
    const terminal = new DelayedTerminal();
    terminals.push(terminal);
    return terminal;
  });
  renderer.sync([], 0, false);
  expect(terminals).toHaveLength(0);
  renderer.sync([chunk(1), chunk(2)], 1, true);
  const first = terminals[0];
  renderer.sync([chunk(1), chunk(2), chunk(3)], 1, true);
  expect(first.pending).toHaveLength(1);
  first.acknowledge();
  expect(first.content).toBe("1\n");
  expect(first.pending).toHaveLength(1);

  // Eviction rebuilds from the retained snapshot, never from terminal scrollback.
  renderer.sync([chunk(2), chunk(3)], 1, true);
  const retained = terminals[1];
  expect(first.disposed).toBe(true);
  first.acknowledge();
  expect(retained.pending).toHaveLength(1);
  retained.acknowledge();
  retained.acknowledge();
  expect(retained.content).toBe("2\n3\n");

  // A cursor gap resets parser/style state and retains only the contiguous tail.
  renderer.sync([chunk(2), chunk(3), chunk(5)], 1, true);
  const gap = terminals[2];
  expect(retained.disposed).toBe(true);
  gap.acknowledge();
  expect(gap.content).toBe("5\n");

  // Explicit reset must work even if Svelte coalesces empty + new snapshots.
  renderer.sync([chunk(5, "new launch\n")], 2, true);
  const cleared = terminals[3];
  expect(gap.disposed).toBe(true);
  renderer.sync([], 3, false);
  expect(cleared.disposed).toBe(true);
  cleared.acknowledge();
  expect(cleared.content).toBe("");
  renderer.sync([chunk(5, "renewed consent\n")], 4, true);
  const reopened = terminals[4];
  reopened.acknowledge();
  expect(reopened.content).toBe("renewed consent\n");
  renderer.dispose();
  expect(reopened.disposed).toBe(true);
  renderer.sync([chunk(6)], 5, true);
  expect(terminals).toHaveLength(5);
});

it("rebuilds highlighting from retained output on toggles and discards old stream context", () => {
  const terminals: DelayedTerminal[] = [];
  const renderer = new ProcessOutputRenderer(() => {
    const terminal = new DelayedTerminal();
    terminals.push(terminal);
    return terminal;
  });
  const prefix = chunk(1, '{"event":"http.request",');
  const suffix = chunk(2, '"status":503}\n');
  renderer.sync([prefix, suffix], 1, true);
  terminals[0].acknowledge();
  expect(terminals[0].content).toContain("\u001b[34m");

  // Evicting a prefix cannot classify a retained suffix as that same record.
  renderer.sync([suffix], 1, true);
  terminals[0].acknowledge();
  terminals[1].acknowledge();
  expect(terminals[1].content).toBe(suffix.text);

  renderer.sync([prefix, suffix], 2, true, false);
  terminals[2].acknowledge();
  terminals[2].acknowledge();
  expect(terminals[2].content).toBe(prefix.text + suffix.text);
  renderer.sync([prefix, suffix], 2, true, true);
  expect(terminals[2].disposed).toBe(true);
  terminals[3].acknowledge();
  terminals[3].acknowledge();
  expect(terminals[3].content).toContain("\u001b[31m503");
  renderer.dispose();
});
