import type { Instance, Model, Status } from "$bindings/managedruntime";
import { runtimeModelsDownloaded } from "./managedRuntime";

/** Saved selections identify models; only live runtime inventory proves downloads. */
export function runtimeConnectionModels(
  instance?: Instance,
  status?: Status,
  catalog: readonly Model[] = [],
) {
  const selectedModel = (id: string | undefined) => {
    if (!id) return null;
    const model = status?.models?.find((candidate) => candidate.id === id);
    const downloading =
      status?.operation?.kind === "download" &&
      status.operation.outcome === "running" &&
      status.operation.model === id;
    return {
      id,
      name:
        model?.name ||
        catalog.find((candidate) => candidate.id === id)?.name ||
        id,
      downloaded: model?.installed === true,
      missing: !!model && !model.installed,
      downloading,
      unavailable: !!status && !model,
      label: !status
        ? "Download status unavailable"
        : downloading
          ? "Downloading"
          : !model
            ? "Not in runtime catalog"
            : model.installed
              ? "Downloaded"
              : "Not downloaded",
    };
  };
  const primary = selectedModel(instance?.model);
  const speech = selectedModel(instance?.speechModel);
  const downloaded =
    !!instance &&
    !!status &&
    runtimeModelsDownloaded(instance, status.models ?? []);
  const summary = () => {
    if (!primary) return "No model selected";
    if (!status) return "Model status unavailable";
    if (primary.unavailable) return "Selected model unavailable";
    if (speech?.unavailable) return "Speech model unavailable";
    if (primary.downloading) return "Downloading selected model";
    if (speech?.downloading) return "Downloading speech model";
    if (downloaded) return speech ? "Models downloaded" : "Model downloaded";
    if (primary.missing)
      return speech?.missing ? "Models not downloaded" : "Model not downloaded";
    return "Speech model not downloaded";
  };
  return { primary, speech, downloaded, summary: summary() };
}
