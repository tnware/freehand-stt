import { SETTINGS_SECTIONS, type SettingsSectionID } from "./navigation";
import { connectionSection } from "./utils/connectionChoices";
import type {
  ConnectionManagerRequest,
  SettingsRequest,
} from "$bindings/windowing";
import type { Purpose } from "$bindings/savedconnection";

export const TASK_CONNECTION_NAVIGATION = Symbol("task-connection-navigation");

// Navigation contains no credentials. The shared Session remains the sole draft owner.
export class ShellNavigation {
  open = $state(false);
  active = $state<SettingsSectionID>("general");
  origin = $state("");
  get saveReturnsToTask() {
    return this.origin !== "";
  }
  acceptRequest(request: SettingsRequest, blocked: boolean) {
    if (
      blocked ||
      !SETTINGS_SECTIONS.some((section) => section.id === request.section)
    )
      return false;
    if (request.origin && !["voice", "file", "tts"].includes(request.origin))
      return false;
    this.connection = null;
    this.open = false;
    this.origin = request.origin;
    if (request.connection)
      this.openConnection(request.connection, request.origin);
    else
      this.openSettings(request.section as SettingsSectionID, request.origin);
    return true;
  }
  connection = $state<ConnectionManagerRequest | null>(null);
  private returnSection: SettingsSectionID = "general";
  private connectionOrigin: "catalog" | "task" = "catalog";
  openSettings(section: SettingsSectionID, origin = this.origin) {
    if (section === "connections") this.connectionOrigin = "catalog";
    if (!this.open) this.origin = origin;
    this.open = true;
    this.active = section;
  }
  openConnection(request: ConnectionManagerRequest, origin = this.origin) {
    if (this.connection) return; // Revealing the editor never replaces its live draft.
    this.returnSection = this.open
      ? this.active
      : request.purpose
        ? connectionSection(request.purpose)
        : "connections";
    this.openSettings("connections", origin);
    this.connectionOrigin =
      this.returnSection === "connections" ? "catalog" : "task";
    this.connection = request;
  }
  returnFromConnection(purpose?: Purpose) {
    this.connection = null;
    this.active =
      this.connectionOrigin === "catalog"
        ? purpose
          ? connectionSection(purpose)
          : "connections"
        : this.returnSection;
  }
  done() {
    this.connection = null;
    this.open = false;
    this.origin = "";
    this.active = "general";
    this.returnSection = "general";
    this.connectionOrigin = "catalog";
  }
}
// Reject before the parent stores a continuation; the manager rejects the same busy states.
export function acceptConnectionClose(
  editor: { saving: boolean; managedConnectionTesting: boolean },
  accept: () => void,
) {
  if (editor.saving || editor.managedConnectionTesting) return false;
  accept();
  return true;
}
export async function saveConnectionAndContinue(
  save: (purpose?: Purpose) => Promise<boolean>,
  purpose: Purpose | undefined,
  next: () => void,
) {
  if (await save(purpose)) next();
}
