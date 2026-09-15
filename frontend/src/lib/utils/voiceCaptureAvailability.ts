import type { Status } from "$bindings/managedruntime";
import {
  connectionStatusLabel,
  connectionSucceeded,
  type TaskConnectionDetails,
} from "$lib/utils/connection";
import { runtimePresentation } from "$lib/utils/managedRuntime";

export interface VoiceCaptureAvailability {
  label: string;
  detail: string;
  blocked: boolean;
  busy: boolean;
  attention: boolean;
  action: "check" | "runtime" | "";
}

/** Metadata failures describe availability; they cannot prove inference is unusable. */
export function manualVoiceAvailability(
  connection: TaskConnectionDetails,
  checked: boolean,
): VoiceCaptureAvailability {
  const checking =
    connection.loading ||
    connection.busy ||
    (!!connection.selected && !checked);
  if (checking)
    return {
      label: "Checking connection",
      detail: "Checking server availability…",
      blocked: true,
      busy: true,
      attention: false,
      action: "",
    };
  if (connection.stale || !connection.result)
    return {
      label: "Connection availability unknown",
      detail: connection.stale
        ? "The saved Voice connection changed. Check again or try recording."
        : "Could not confirm server availability. Check again or try recording.",
      blocked: false,
      busy: false,
      attention: true,
      action: connection.selected ? "check" : "",
    };
  if (!connectionSucceeded(connection.result) || !connection.result.reachable)
    return {
      label: "Connection check failed",
      detail: `${connectionSucceeded(connection.result) ? "Could not confirm server availability" : connectionStatusLabel(connection.result)}. Check again or try recording.`,
      blocked: false,
      busy: false,
      attention: true,
      action: connection.selected ? "check" : "",
    };
  return {
    label: "",
    detail: "Server reachable",
    blocked: false,
    busy: false,
    attention: false,
    action: "",
  };
}

/** Only the selected runtime's authoritative running state admits a new recording. */
export function managedVoiceAvailability(
  status: Status | null | undefined,
  now: number,
  { pending = "", busy = false, checking = false, error = "" } = {},
): VoiceCaptureAvailability {
  if (!status)
    return {
      label:
        error || !checking ? "Runtime status unavailable" : "Checking runtime",
      detail:
        error ||
        (checking
          ? "Reading local runtime status."
          : "Open local runtimes to review the selected runtime."),
      blocked: true,
      busy: checking && !error,
      attention: !checking || !!error,
      action: checking && !error ? "" : "runtime",
    };
  const view = runtimePresentation(status, now, pending);
  const working =
    busy ||
    !!pending ||
    status.operation?.outcome === "running" ||
    ["installing", "starting", "stopping"].includes(status.state);
  const problem = working ? "" : error || view.error;
  const ready = view.ready && !working;
  return {
    label: working
      ? status.state === "starting" && pending !== "Cancelling"
        ? "Starting runtime"
        : view.activity
      : problem
        ? "Runtime needs attention"
        : ready
          ? ""
          : status.state === "stopped" || status.state === "installed"
            ? "Runtime stopped"
            : view.label,
    detail: view.startup || problem || (ready ? "Local runtime running" : ""),
    blocked: !ready,
    busy: working,
    attention: !!problem || (!working && !ready),
    action: ready ? "" : "runtime",
  };
}
