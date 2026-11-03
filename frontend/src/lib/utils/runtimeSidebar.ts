import type {
  InstanceStatus,
  ProviderDescriptor,
} from "$bindings/managedruntime";
import {
  backendLabel,
  isSpeechRuntimeModel,
  runtimePresentation,
} from "./managedRuntime";

/** Read-only setup guidance. A recommendation never becomes a saved selection. */
export function runtimeSidebarContext(
  entry: ProviderDescriptor,
  row?: InstanceStatus,
  pending = "",
  problem = "",
) {
  const status = row?.status;
  const view = runtimePresentation(status, undefined, pending);
  const models = (
    (status?.models?.length ? status.models : entry.models) ?? []
  ).filter((model) =>
    entry.models?.some((qualified) => qualified.id === model.id),
  );
  const selectedID = row?.instance.model ?? "";
  const selected = models.find((model) => model.id === selectedID);
  const recommended = entry.models?.find(
    (model) => model.recommended && !isSpeechRuntimeModel(model),
  );
  const speechID = row?.instance.speechModel ?? "";
  const speech = models.find((model) => model.id === speechID);
  const modelName = (id: string) =>
    models.find((model) => model.id === id)?.name ||
    entry.models?.find((model) => model.id === id)?.name ||
    id;
  const model = selectedID
    ? { label: "Selected model", name: modelName(selectedID), id: selectedID }
    : recommended
      ? {
          label: "Recommended model",
          name: recommended.name,
          id: recommended.id,
        }
      : { label: "Model", name: "No model selected", id: "" };
  const speechModel = speechID
    ? {
        label: "Selected speech model",
        name: modelName(speechID),
        id: speechID,
      }
    : null;
  // Only the installed backend is evidence about this runtime. Descriptor
  // backend lists are build options, not a probe of the user's hardware.
  const binary = status?.backend
    ? [backendLabel(status.backend), status.version].filter(Boolean).join(" · ")
    : "";
  let heading = "Next step";
  let next = "Install runtime";
  let detail = "Then download a model to use locally.";

  if (!entry.supported || status?.supported === false) {
    heading = "Unavailable";
    next = "Choose another runtime";
    detail =
      entry.unavailableReason ||
      "This runtime is not supported on this machine.";
  } else if (view.busy) {
    heading = "In progress";
    next =
      status?.state === "installing" && view.activity === "Working"
        ? "Installing runtime"
        : view.activity;
    detail = [view.operationModel, view.transferred]
      .filter(Boolean)
      .join(" · ");
  } else if (problem || view.error || status?.state === "error") {
    heading = "Needs attention";
    next = "Review runtime error";
    detail = "Check runtime details before retrying the failed step.";
  } else if (view.ready) {
    heading = "Runtime ready";
    next = "Running locally";
    detail = "Available to workflows that use this runtime in Connections.";
  } else if (view.installed) {
    if (!selectedID || !selected || (speechID && !speech)) {
      next = "Choose a model";
      detail = "Review your selection in the model catalog.";
    } else if (!selected.installed) {
      next = "Download selected model";
      detail =
        "The runtime is installed. Its selected model still needs a download.";
    } else if (speech && !speech.installed) {
      next = "Download speech model";
      detail =
        "The optional speech model must also be downloaded before starting.";
    } else {
      next = "Start runtime";
      detail = "Load the selected models to make this runtime available.";
    }
  }
  return { model, speechModel, binary, heading, next, detail };
}
