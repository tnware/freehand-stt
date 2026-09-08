/**
 * How long a success notice stays up. Confirmations are transient and nothing
 * is lost when one goes away; errors stay until they are dismissed or the next
 * action replaces them.
 */
const NOTICE_MS = 6000;

/** Shared presentation-only notices; no feature state. */
export class SessionMessages {
  info = $state("");
  notice = $state("");
  error = $state("");
  #speechFailureGeneration = $state<number | null>(null);
  #infoTimer: ReturnType<typeof setTimeout> | undefined;
  #noticeTimer: ReturnType<typeof setTimeout> | undefined;

  clear() {
    this.dismissError();
    this.dismissInfo();
    this.dismissNotice();
  }

  dismissInfo() {
    clearTimeout(this.#infoTimer);
    this.#infoTimer = undefined;
    this.info = "";
  }

  dismissNotice() {
    clearTimeout(this.#noticeTimer);
    this.#noticeTimer = undefined;
    this.notice = "";
  }

  dismissError() {
    this.error = "";
    this.#speechFailureGeneration = null;
  }

  /** Shows an actionable renderer failure in the existing visible channel. */
  reportFailure(message: string) {
    this.dismissNotice();
    this.error = message;
    this.#speechFailureGeneration = null;
  }

  /** Lets a visible playback surface own this exact failure without hiding other action errors. */
  reportSpeechFailure(message: string, generation: number) {
    this.reportFailure(message);
    this.#speechFailureGeneration = generation;
  }

  isSpeechFailure(generation: number): boolean {
    return !!this.error && this.#speechFailureGeneration === generation;
  }

  /** Shows a transient explanation of system behaviour, separate from success. */
  reportInfo(message: string) {
    clearTimeout(this.#infoTimer);
    this.info = message;
    this.#infoTimer = setTimeout(() => {
      this.info = "";
      this.#infoTimer = undefined;
    }, NOTICE_MS);
  }

  /** Shows a confirmation that takes itself down again. */
  announce(message: string) {
    clearTimeout(this.#noticeTimer);
    this.notice = message;
    this.#noticeTimer = setTimeout(() => {
      this.notice = "";
      this.#noticeTimer = undefined;
    }, NOTICE_MS);
  }

  /** Records a caught value as the visible error. */
  fail(cause: unknown) {
    this.reportFailure(cause instanceof Error ? cause.message : String(cause));
  }

  dispose() {
    this.clear();
  }
}
