import { State, type Status } from "$lib/state";

/** States from which the record control may be pressed. */
const TOGGLEABLE: State[] = [State.Idle, State.Failed, State.Recording];

export const canToggleRecording = (status: Status, busy: boolean): boolean =>
  !busy && TOGGLEABLE.includes(status.state);

export const isRecording = (status: Status): boolean =>
  status.state === State.Recording;

/**
 * Copy-required preserves the transcript for explicit action. It may mean
 * manual copy, rejected capture/validation, or incomplete dispatch. This flag
 * alone proves neither focus movement nor that no text was posted.
 */
export const isCopyRequired = (status: Status): boolean =>
  status.state === State.Failed && status.canCopy && !status.startRejected;

export const isFailure = (status: Status): boolean =>
  status.state === State.Failed && (!status.canCopy || !!status.startRejected);

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
 * failure carries the reason, and copy-required preserves the backend's bounded
 * delivery explanation without inventing a focus or dispatch outcome.
 */
export const statusMessage = (status: Status): string => {
  if (isCopyRequired(status)) {
    return (
      status.message ||
      "Copy the transcript and paste it where you want it. Check the target for any text already inserted before pasting."
    );
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
