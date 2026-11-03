import { Role } from "$bindings/compatibility";
import type {
  InstanceStatus,
  ProviderDescriptor,
} from "$bindings/managedruntime";
import { runtimeModelsDownloaded, runtimePresentation } from "./managedRuntime";

export type RuntimeSetupPurpose = "speech" | "cleanup";
export type RuntimeSetupIntent = {
  purpose: RuntimeSetupPurpose;
  providerID: string;
};

/** Navigation guidance only. It never selects, downloads or enables a model. */
export function runtimeSetupGuidance(
  purpose: RuntimeSetupPurpose,
  entry: ProviderDescriptor,
  row?: InstanceStatus,
  pending = "",
  problem = "",
) {
  const role = purpose === "speech" ? Role.Speech : Role.PostProcessing;
  const label = purpose === "speech" ? "speech" : "cleanup";
  const section = purpose === "speech" ? "speech" : "processing";
  const settingsLabel =
    purpose === "speech" ? "Speech settings" : "Cleanup settings";
  const models = (
    row?.status.models?.length ? row.status.models : (entry.models ?? [])
  ).filter((model) =>
    entry.models?.some((qualified) => qualified.id === model.id),
  );
  const qualified = (id: string | undefined) =>
    models.find(
      (model) =>
        model.id === id &&
        model.contracts?.some((contract) => contract.role === role),
    );
  const selected = qualified(
    purpose === "speech" ? row?.instance.speechModel : row?.instance.model,
  );
  const view = runtimePresentation(row?.status, undefined, pending);
  const supported = entry.supported && row?.status.supported !== false;
  const available = models.some((model) =>
    model.contracts?.some((contract) => contract.role === role),
  );
  let next = `Install the runtime for ${label}`;
  let detail = `After installation, choose a ${label} model below, download it, then select its connection in ${settingsLabel}.`;
  let chooseModel = false;
  if (!supported || !available) {
    next = `Choose another runtime for ${label}`;
    detail = !supported
      ? entry.unavailableReason ||
        "This runtime is unavailable on this machine."
      : `This runtime has no qualified ${label} models.`;
  } else if (view.busy) {
    next = view.activity;
    detail = "Wait for this operation, or cancel it before changing models.";
  } else if (problem || view.error || row?.status.state === "error") {
    next = "Review the runtime error";
    detail = `Resolve the error before continuing ${label} setup.`;
  } else if (view.installed) {
    if (!selected) {
      next =
        row?.status.state === "running"
          ? `Stop the runtime to choose a ${label} model`
          : `Choose a ${label} model`;
      detail =
        purpose === "speech"
          ? "Select a model under Text to speech. NeMo also loads its selected transcription model; both need a download."
          : "Select a cleanup model below, then download it explicitly.";
      chooseModel = row?.status.state !== "running";
    } else if (!selected.installed) {
      next = `Download the selected ${label} model`;
      detail = `${selected.name} is selected but not downloaded.`;
    } else if (row && !runtimeModelsDownloaded(row.instance, models)) {
      next = "Download the remaining selected model";
      detail =
        "NeMo loads the selected transcription and speech models together. Both must be downloaded before Start.";
    } else if (view.ready) {
      next = `Choose its built-in connection in ${settingsLabel}`;
      detail = `The runtime is running. Return to ${settingsLabel} and choose its built-in connection for ${label}.`;
    } else {
      next = "Start the runtime";
      detail = `Then return to ${settingsLabel} and choose its built-in connection. This does not enable ${label} automatically.`;
    }
  }
  return {
    role,
    label,
    section,
    settingsLabel,
    selected,
    next,
    detail,
    chooseModel,
  } as const;
}
