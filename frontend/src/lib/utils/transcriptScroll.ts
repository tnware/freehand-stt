export interface TranscriptScrollOptions {
  key: string;
  content: string;
  streaming: boolean;
  jump: number;
  onFollowingChange: (following: boolean) => void;
}

/** Follow appended text until the reader scrolls back. Owns no transcript state. */
export function followTranscript(node: HTMLElement, initial: TranscriptScrollOptions) {
  let options = initial;
  let following = true;
  let previousTop = node.scrollTop;
  let frame = 0;
  let resetPending = false;
  let finishPending = false;
  const atBottom = () => node.scrollHeight - node.clientHeight - node.scrollTop <= 24;
  function setFollowing(value: boolean) {
    following = value;
    options.onFollowingChange(value);
  }
  function scroll() {
    if (atBottom()) setFollowing(true);
    else if (node.scrollTop < previousTop) setFollowing(false);
    previousTop = node.scrollTop;
  }
  function schedule(reset = false, finish = false) {
    resetPending ||= reset;
    finishPending ||= finish;
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      if (resetPending && !options.streaming) node.scrollTop = 0;
      else if (following && (options.streaming || finishPending))
        node.scrollTop = node.scrollHeight;
      resetPending = false;
      finishPending = false;
      previousTop = node.scrollTop;
    });
  }
  node.addEventListener("scroll", scroll, { passive: true });
  const observer = new ResizeObserver(() => schedule());
  observer.observe(node);
  // The content wrapper is stable even when provisional text becomes final.
  if (node.firstElementChild) observer.observe(node.firstElementChild);
  schedule(true);
  return {
    update(next: TranscriptScrollOptions) {
      const changed = next.key !== options.key;
      const jump = next.jump !== options.jump;
      const finish = options.streaming && !next.streaming;
      options = next;
      if (changed || jump) setFollowing(true);
      schedule(changed, finish || jump);
    },
    destroy() {
      cancelAnimationFrame(frame);
      observer.disconnect();
      node.removeEventListener("scroll", scroll);
    },
  };
}
