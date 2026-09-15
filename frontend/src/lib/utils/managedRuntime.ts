import type { Model, Status } from "$bindings/managedruntime";

export function backendLabel(backend: string): string {
  if (backend === "cpu") return "CPU";
  if (backend === "cuda") return "NVIDIA GPU (CUDA)";
  if (backend === "metal") return "Apple GPU (Metal)";
  return backend.toUpperCase();
}

export function runtimePresentation(
  status: Status | null | undefined,
  now = Date.now(),
  pending = "",
) {
  // Binding admission is asynchronous. Describe that request until the backend
  // publishes its own stage; never turn a pending action into optimistic state.
  const pendingLabels: Record<string, string> = {
    Start: "Starting runtime",
    Stop: "Stopping runtime",
    Restart: "Restarting runtime",
    Cancelling: "Cancelling operation",
    Install: "Installing runtime",
    Remove: "Removing runtime files",
    RefreshCatalog: "Refreshing catalog",
    "Switch runtime binary": "Switching runtime binary",
    "Download model": "Preparing download",
    "Remove model": "Removing model",
    "Save instance": "Saving runtime settings",
    "Delete instance": "Deleting runtime entry",
  };
  const pendingActivity = pending ? (pendingLabels[pending] ?? "Working") : "";
  const cancelling = pending === "Cancelling";
  const starting = status?.state === "starting";
  const stages: Record<string, string> = {
    verifying_runtime: "Verifying runtime",
    verifying_model: "Verifying selected model",
    launching: "Launching runtime",
    waiting_ready: "Waiting for runtime readiness",
    warming_up: "Warming up selected model",
    loading_warming: "Loading and warming up selected model",
  };
  const stage = starting
    ? (stages[status.startupProgress?.phase ?? ""] ?? "Starting runtime")
    : "";
  const since = status?.startupProgress?.startedAt;
  const startup =
    starting && !cancelling
      ? `${stage}${since && Number.isFinite(since) ? ` · ${Math.max(0, Math.floor((now - since) / 1000))}s in this stage` : ""}`
      : "";
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
    startup:
      starting || status?.state === "running"
        ? "Starting runtime"
        : "Checking runtime",
    restart: "Restarting runtime",
    stop: "Stopping runtime",
    catalog: "Refreshing catalog",
    remove: "Removing runtime files",
    remove_model: "Removing model",
  };
  const backendActivity =
    stage ||
    (status?.state === "stopping"
      ? "Stopping runtime"
      : active && operation.kind === "download"
        ? (phaseLabels[acquisition?.phase ?? ""] ?? "Preparing download")
        : (operationLabels[status?.phase || operation?.kind || ""] ??
          "Working"));
  const activity = cancelling
    ? pendingActivity
    : active || starting || status?.state === "stopping"
      ? backendActivity
      : pendingActivity || backendActivity;
  const completionLabels: Record<string, string> = {
    install: "Runtime installed.",
    download: "Model downloaded and verified.",
    start: "Runtime ready.",
    startup: status?.state === "running" ? "Runtime ready." : "",
    restart: "Runtime restarted.",
    stop: "Runtime stopped.",
    remove: "Runtime files removed.",
    remove_model: "Model removed.",
  };
  const completion =
    !operation?.id || active || pendingActivity
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
    !cancelling &&
    operation.kind === "download" &&
    acquisition?.phase === "downloading";
  const bytes = acquisition?.bytes ?? 0;
  const total = acquisition?.totalBytes ?? 0;
  return {
    backend: backendLabel(status?.backend ?? ""),
    activity,
    startup,
    completion,
    error:
      pendingActivity || active
        ? ""
        : status?.error ||
          (operation?.outcome === "failed"
            ? operation.error || "Operation failed. Review the error and retry."
            : ""),
    operationModel: pendingActivity && !active ? "" : (modelName ?? ""),
    transferred: measured
      ? `${transferSize(bytes)}${total > 0 ? ` / ${transferSize(total)}` : " downloaded"}`
      : "",
    label: !status
      ? "Status unavailable"
      : !supported
        ? "Unavailable on this platform"
        : active || pendingActivity
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
      starting || cancelling || (pendingActivity && !active)
        ? null
        : active && operation.kind === "download"
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
