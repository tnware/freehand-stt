import type { Model, Status } from "$bindings/managedruntime";

export function runtimePresentation(status: Status | null | undefined) {
  const supported = status?.supported === true;
  const labels: Record<string, string> = {
    not_installed: "Not installed",
    installing: "Installing",
    installed: "Installed",
    starting: "Starting",
    running: "Running",
    stopping: "Stopping",
    stopped: "Stopped",
    error: "Needs attention",
  };
  const installed =
    !!status &&
    !["not_installed", "installing"].includes(status.state) &&
    !!(
      status.backend ||
      ["installed", "starting", "running", "stopping", "stopped"].includes(
        status.state,
      )
    );
  const ready = supported && status.state === "running";
  const selected = status?.models?.find(
    (model) => model.id === status.selectedModel,
  );
  const operation = status?.operation;
  const acquisition = status?.acquisition;
  const active = operation?.outcome === "running";
  const phaseLabels: Record<string, string> = {
    preparing: "Preparing download",
    downloading: "Downloading model",
    verifying: "Verifying model",
  };
  const operationLabels: Record<string, string> = {
    install: "Installing runtime",
    download: "Downloading model",
    start: "Starting runtime",
    startup: "Starting runtime",
    stop: "Stopping runtime",
    catalog: "Refreshing catalog",
    remove: "Removing runtime files",
    remove_model: "Removing model",
  };
  const activity =
    active && operation.kind === "download"
      ? (phaseLabels[acquisition?.phase ?? ""] ?? "Preparing download")
      : (operationLabels[status?.phase ?? ""] ?? "Working");
  const completionLabels: Record<string, string> = {
    install: "Runtime installed.",
    download: "Model downloaded and verified.",
    start: "Runtime ready.",
    startup: "Runtime ready.",
    stop: "Runtime stopped.",
    remove: "Runtime files removed.",
    remove_model: "Model removed.",
  };
  const completion =
    !operation?.id || active
      ? ""
      : operation.outcome === "succeeded"
        ? (completionLabels[operation.kind] ?? "")
        : operation.outcome === "cancelled"
          ? "Operation cancelled."
          : operation.outcome === "failed"
            ? operation.error || "Operation failed. Review the error and retry."
            : "";
  const modelName = status?.models?.find(
    (model) => model.id === operation?.model,
  )?.name;
  const measured =
    active &&
    operation.kind === "download" &&
    acquisition?.phase === "downloading";
  const bytes = acquisition?.bytes ?? 0;
  const total = acquisition?.totalBytes ?? 0;
  return {
    activity,
    completion,
    operationModel: modelName ?? "",
    transferred: measured
      ? `${transferSize(bytes)}${total > 0 ? ` / ${transferSize(total)}` : " downloaded"}`
      : "",
    label: !status
      ? "Status unavailable"
      : !supported
        ? "Windows only"
        : active
          ? activity
          : (labels[status.state] ?? "Checking status"),
    installed,
    ready,
    selected,
    step: ready
      ? 3
      : selected?.installed || status?.state === "starting"
        ? 2
        : installed
          ? 1
          : 0,
    percent:
      active && operation.kind === "download"
        ? measured && total > 0
          ? Math.floor(Math.min(1, bytes / total) * 100)
          : null
        : status && Number.isFinite(status.progress) && status.progress >= 0
          ? Math.round(Math.min(1, status.progress) * 100)
          : null,
  };
}
function transferSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 MB";
  return bytes >= 1_000_000_000
    ? `${(bytes / 1_000_000_000).toFixed(2)} GB`
    : `${(bytes / 1_000_000).toFixed(1)} MB`;
}
export function modelSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "Size not reported";
  return bytes >= 1_000_000_000
    ? `${(bytes / 1_000_000_000).toFixed(1)} GB`
    : `${Math.ceil(bytes / 1_000_000)} MB`;
}
export function catalogGroups(models: Model[]) {
  const sorted = [...models].sort(
    (a, b) =>
      Number(b.recommended) - Number(a.recommended) ||
      a.name.localeCompare(b.name),
  );
  return [
    { label: "Downloaded", models: sorted.filter((m) => m.installed) },
    {
      label: "Available to download",
      models: sorted.filter((m) => !m.installed),
    },
  ];
}
