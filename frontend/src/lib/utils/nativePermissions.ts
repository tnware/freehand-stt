import type { PermissionStatus } from "$bindings/input";
import type { PermissionKind } from "$lib/stores/nativePermissions.svelte";
type PermissionRow = {
  kind: PermissionKind;
  label: string;
  description: string;
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
      description: "Record speech and preview microphone input.",
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
      description: "Check the destination, insert text, and capture shortcuts.",
      state: status.accessibility ? "Allowed" : "Not allowed",
      action: status.accessibility ? null : "settings",
    },
    {
      kind: "keyboard",
      label: "Input Monitoring",
      description: "Capture shortcuts and use hold-to-talk.",
      state: status.keyboard ? "Allowed" : "Not allowed",
      action: status.keyboard ? null : "settings",
    },
  ];
}
