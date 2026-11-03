import type { Catalog } from "$bindings/savedconnection";
import { connectionWorkflows } from "./connectionChoices";

/** Explain saved dependencies only; deletion admission remains backend-owned. */
export function runtimeDeletionReferences(
  catalog: Catalog | undefined,
  instanceID: string,
) {
  const references = instanceID
    ? (catalog?.entries ?? []).filter(
        (connection) => connection.details.managedInstanceID === instanceID,
      )
    : [];
  const ids = new Set(references.map((connection) => connection.id));
  return {
    workflows: connectionWorkflows
      .filter((workflow) => ids.has(catalog?.selected?.[workflow.id] ?? ""))
      .map((workflow) => workflow.label),
    connections: references
      .filter((connection) => !connection.builtIn)
      .map((connection) => connection.name),
  };
}
