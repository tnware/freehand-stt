import { State, type Status } from "$lib/state";

/** States from which the record control may be pressed. */
const TOGGLEABLE: State[] = [State.Idle, State.Failed, State.Recording];

export const canToggleRecording = (status: Status, busy: boolean): boolean =>
  !busy && TOGGLEABLE.includes(status.state);

export const isRecording = (status: Status): boolean => status.state === State.Recording;

/**
 * Copy-required is not a failure. The coordinator reports it on the failed
 * state because insertion did not happen, but the transcript is intact and
 * waiting: focus moved, so refusing to type was the correct outcome. It reads
 * as an outcome to act on, never as something that broke.
 */
export const isCopyRequired = (status: Status): boolean =>
  status.state === State.Failed && status.canCopy;

export const isFailure = (status: Status): boolean =>
  status.state === State.Failed && !status.canCopy;

/**
 * What the progress rail under the transport is doing.
 *
 * The rail covers everything that happens after you stop speaking — sending,
 * transcribing, inserting — because from the outside those are one wait. It
 * occupies its three pixels in every state, so appearing and disappearing is a
 * change of opacity rather than of layout.
 */
export type RailPhase = "hidden" | "working" | "done" | "error";

export const railPhase = (status: Status): RailPhase => {
  if (isCopyRequired(status)) return "done";
  if (isFailure(status)) return "error";
  switch (status.state) {
    case State.Transcribing:
    case State.PostProcessing:
    case State.Ready:
    case State.Cancelling:
      return "working";
    default:
      return "hidden";
  }
};

/**
 * The line for the messages channel, or "" when the transport already says enough.
 *
 * Only two states need more words than the transport has room for: a genuine
 * failure carries the reason, and copy-required has to explain why nothing was
 * typed even though nothing broke.
 */
export const statusMessage = (status: Status): string => {
  if (isCopyRequired(status)) {
    return "Focus moved before the transcript came back, so nothing was typed. Copy it from here and paste it where you want it.";
  }
  if (isFailure(status)) {
    return status.message || "The recording could not be transcribed.";
  }
  return "";
};

/** Shortcut guidance describes how to begin, so it belongs only to idle. */
export const showIdleShortcutGuidance = (
  status: Status,
  shortcut: string,
): boolean => status.state === State.Idle && shortcut.length > 0;
