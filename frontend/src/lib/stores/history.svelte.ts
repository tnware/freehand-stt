import type { HistoryEntry, HistoryTextVersion } from "$lib/state";
import type { SessionMessages } from "./messages.svelte";
import type * as HistoryBindings from "$bindings/history/service";
export type HistoryStateService = Pick<
  typeof HistoryBindings,
  | "TranscriptHistory"
  | "CopyHistoryEntry"
  | "CopyHistoryEntryVersion"
  | "DeleteHistoryEntry"
  | "ClearHistory"
>;

/** Owns the history projection and refresh/mutation ordering. */
export class HistoryState {
  readonly #service: HistoryStateService;
  readonly #messages: SessionMessages;
  readonly #onRefreshed: () => void;
  constructor(
    service: HistoryStateService,
    messages: SessionMessages,
    onRefreshed: () => void = () => {},
  ) {
    this.#service = service;
    this.#messages = messages;
    this.#onRefreshed = onRefreshed;
  }
  entries = $state<HistoryEntry[]>([]);
  #historyRequest = 0;

  async refresh() {
    const request = ++this.#historyRequest;
    try {
      const history = (await this.#service.TranscriptHistory()) ?? [];
      if (request === this.#historyRequest) {
        this.entries = history;
        this.#onRefreshed();
      }
    } catch (cause) {
      if (request === this.#historyRequest) this.#messages.fail(cause);
    }
  }

  async copyHistoryEntry(id: number): Promise<boolean> {
    this.#messages.clear();
    try {
      await this.#service.CopyHistoryEntry(id);
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    }
  }

  async copyHistoryEntryVersion(
    id: number,
    version: HistoryTextVersion,
  ): Promise<boolean> {
    this.#messages.clear();
    try {
      await this.#service.CopyHistoryEntryVersion(id, version);
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    }
  }

  async deleteHistoryEntry(id: number): Promise<boolean> {
    this.#messages.clear();
    try {
      await this.#service.DeleteHistoryEntry(id);
      this.#historyRequest++;
      this.entries = this.entries.filter((entry) => entry.id !== id);
      return true;
    } catch (cause) {
      this.#messages.fail(cause);
      return false;
    }
  }

  async clearHistory() {
    this.#messages.clear();
    try {
      await this.#service.ClearHistory();
      this.#historyRequest++;
      this.entries = [];
      this.#messages.announce("Transcript history cleared.");
    } catch (cause) {
      this.#messages.fail(cause);
    }
  }
}
