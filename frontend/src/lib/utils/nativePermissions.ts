import type { PermissionStatus } from "$bindings/input";
import type { PermissionKind } from "$lib/stores/nativePermissions.svelte";
type PermissionRow = {
  kind: PermissionKind;
  label: string;
  state: string;
  action: "request" | "settings" | null;
};
export function permissionRows(
  status: PermissionStatus | null,
): PermissionRow[] {
  if (!status?.required) return [];
  return [
    {
      kind: "microphone",
      label: "Microphone",
      state:
        status.microphone === "authorized"
          ? "Allowed"
          : status.microphone === "not-determined"
            ? "Not requested"
            : status.microphone === "restricted"
              ? "Restricted"
              : "Denied",
      action:
        status.microphone === "authorized"
          ? null
          : status.microphone === "not-determined"
            ? "request"
            : "settings",
    },
    {
      kind: "accessibility",
      label: "Accessibility",
      state: status.accessibility ? "Allowed" : "Not allowed",
      action: status.accessibility ? null : "settings",
    },
    {
      kind: "keyboard",
      label: "Input Monitoring",
      state: status.keyboard ? "Allowed" : "Not allowed",
      action: status.keyboard ? null : "settings",
    },
  ];
}
