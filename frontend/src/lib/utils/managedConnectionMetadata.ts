import { Purpose } from "$bindings/savedconnection";
import type { InstanceStatus } from "$bindings/managedruntime";
import type { Settings } from "$lib/state";

export interface RuntimeMetadataSource {
  statusFor: (id: string) => InstanceStatus | undefined;
  pendingFor: (id: string) => string;
}

/** Metadata follows the selected process lifecycle, independently of drafts. */
export function managedConnectionMetadata(
  settings: Settings | null,
  purpose: Purpose,
  runtime?: RuntimeMetadataSource,
) {
  if (!settings || !runtime) return null;
  const feature =
    purpose === Purpose.Voice
      ? settings.voiceTranscription
      : purpose === Purpose.Transcription
        ? settings
        : purpose === Purpose.Cleanup
          ? settings.postProcessing
          : settings.textToSpeech;
  const id = feature.managedInstanceID;
  if (!id) return null;
  const row = runtime.statusFor(id);
  const pending = runtime.pendingFor(id);
  const state = row?.status.state ?? "loading";
  const starting =
    state === "starting" || pending === "Start" || pending === "Restart";
  const stopping =
    state === "stopping" || pending === "Stop" || pending === "Cancelling";
  const ready =
    row?.status.supported === true &&
    state === "running" &&
    !starting &&
    !stopping;
  return {
    ready,
    key: JSON.stringify([
      id,
      state,
      row?.instance.model,
      row?.status.operation?.id,
      starting,
      stopping,
    ]),
    label: starting
      ? "Starting runtime"
      : stopping
        ? "Stopping runtime"
        : state === "loading"
          ? "Reading runtime status"
          : "Runtime unavailable",
    detail: starting
      ? "Waiting for model loading and warm-up. The connection will be checked when the runtime is ready."
      : stopping
        ? "The connection will be checked after the runtime starts again."
        : state === "loading"
          ? "Waiting for the local runtime status."
          : "Start the selected local runtime to check its connection.",
  };
}
