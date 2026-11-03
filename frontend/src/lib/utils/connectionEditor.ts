import type { Connection } from "$bindings/savedconnection";

/** HTTP header names are case-insensitive. Reject a collision without losing either value. */
export function renameConnectionHeader(
  headers: Record<string, string | undefined>,
  old: string,
  next: string,
): Record<string, string | undefined> | null {
  if (
    Object.keys(headers).some(
      (key) => key !== old && key.toLowerCase() === next.toLowerCase(),
    )
  )
    return null;
  return Object.fromEntries(
    Object.entries(headers).map(([key, value]) => [
      key === old ? next : key,
      value,
    ]),
  );
}

/** Names are not identities: runtime-owned and manual connections may share one. */
export function savedConnectionID(
  entries: Pick<Connection, "id" | "builtIn">[],
  editedID: string | undefined,
  existingIDs: ReadonlySet<string>,
): string | undefined {
  if (editedID)
    return entries.find((entry) => entry.id === editedID && !entry.builtIn)?.id;
  const added = entries.filter(
    (entry) => !entry.builtIn && !existingIDs.has(entry.id),
  );
  return added.length === 1 ? added[0].id : undefined;
}
