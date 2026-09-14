import type { OutputChunk } from "$bindings/managedruntime/models";

// Only this boundary is asynchronous: xterm acknowledges after parsing a write.
export interface OutputTerminal {
  write(text: string, done: () => void): void;
  dispose(): void;
}

/** Projects the store's bounded snapshot, not an independent output history. */
export class ProcessOutputRenderer {
  #terminal: OutputTerminal | undefined;
  #chunks: readonly OutputChunk[] = [];
  #revision = -1;
  #generation = 0;
  #next = 0;
  #writing = false;
  #disposed = false;

  constructor(private readonly create: () => OutputTerminal) {}

  sync(chunks: readonly OutputChunk[], revision: number, enabled: boolean) {
    if (this.#disposed) return;
    if (!enabled) {
      this.#reset();
      return;
    }
    // A missing sequence means missing parser context. Never join across it.
    let start = 0;
    for (let i = 1; i < chunks.length; i++) {
      if (chunks[i].sequence !== chunks[i - 1].sequence + 1) start = i;
    }
    const retained = chunks.slice(start);
    const extendsPrevious = this.#chunks.every(
      (chunk, index) =>
        retained[index]?.sequence === chunk.sequence &&
        retained[index]?.text === chunk.text,
    );
    if (revision !== this.#revision || !extendsPrevious) this.#reset();
    this.#revision = revision;
    this.#chunks = retained;
    this.#terminal ??= this.create();
    this.#pump();
  }

  #reset() {
    this.#generation++;
    this.#terminal?.dispose();
    this.#terminal = undefined;
    this.#chunks = [];
    this.#next = 0;
    this.#writing = false;
  }

  #pump() {
    if (this.#writing || !this.#terminal || this.#next >= this.#chunks.length)
      return;
    const generation = this.#generation;
    const chunk = this.#chunks[this.#next++];
    this.#writing = true;
    // One bounded chunk in xterm's queue at a time. New polls replace the
    // retained snapshot instead of appending another unbounded write backlog.
    this.#terminal.write(chunk.text, () => {
      if (generation !== this.#generation || this.#disposed) return;
      this.#writing = false;
      this.#pump();
    });
  }

  dispose() {
    this.#disposed = true;
    this.#reset();
  }
}
