import type { BinaryOptions, Status } from "$bindings/managedruntime";
import { backendLabel } from "./managedRuntime";

/** Installed status and advisory host metadata are independent sources. */
export function runtimeBinaryContext(
  status?: Pick<Status, "backend" | "version" | "state">,
  options?: BinaryOptions | null,
) {
  const installed = !!status?.backend && status.state !== "not_installed";
  const currentID = installed ? status.backend : "";
  const recommended = options?.supported
    ? options.options?.find(
        (option) =>
          option.backend === options.recommendedBackend &&
          option.supported &&
          option.available,
      )
    : undefined;
  return {
    currentID,
    current: currentID ? backendLabel(currentID) : "Not installed",
    currentLabel:
      status?.state === "running" ? "Running with" : "Installed backend",
    version: installed ? status?.version || "" : "",
    recommendation: recommended ? backendLabel(recommended.backend) : "",
    choices: (options?.options ?? []).map((option) => ({
      ...option,
      label: backendLabel(option.backend),
      current: !!currentID && option.backend === currentID,
      recommended: !!recommended && option.backend === recommended.backend,
    })),
  };
}
