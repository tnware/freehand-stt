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
  #infoTimer: ReturnType<typeof setTimeout> | undefined;
  #noticeTimer: ReturnType<typeof setTimeout> | undefined;

  clear() {
    this.error = "";
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
  }

  /** Shows an actionable renderer failure in the existing visible channel. */
  reportFailure(message: string) {
    this.dismissNotice();
    this.error = message;
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
    this.error = String(cause);
  }

  dispose() {
    this.clear();
  }
}
