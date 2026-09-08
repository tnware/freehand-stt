/** Discovery is metadata, not proof that a model can perform this workflow. */
export function modelSources(
  model: string,
  server: readonly string[],
  saved: readonly string[],
  edited: readonly string[] = [],
): string {
  const sources = [];
  if (edited.includes(model)) sources.push("Edited");
  if (saved.includes(model)) sources.push("Saved");
  if (server.includes(model)) sources.push("Server");
  return sources.join(" · ") || "Manual ID";
}
