/** Transient copy feedback shared by result and history actions. */
export class CopyFeedback {
  key = $state("");
  #timer: ReturnType<typeof setTimeout> | undefined;
  #request = 0;
  #disposed = false;

  async copy(key: string, operation: () => Promise<boolean>) {
    const request = ++this.#request;
    const copied = await operation();
    if (!copied || this.#disposed || request !== this.#request) return;
    clearTimeout(this.#timer);
    this.key = key;
    this.#timer = setTimeout(() => {
      this.key = "";
    }, 1600);
  }

  dispose() {
    this.#disposed = true;
    clearTimeout(this.#timer);
    this.key = "";
  }
}
