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
  return {
    label: !status
      ? "Status unavailable"
      : !supported
        ? "Windows only"
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
      status && Number.isFinite(status.progress) && status.progress >= 0
        ? Math.round(Math.min(1, status.progress) * 100)
        : null,
  };
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
