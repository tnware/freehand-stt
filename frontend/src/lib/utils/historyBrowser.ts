import { HistorySource, type HistoryEntry } from "$bindings/history";

export type HistorySourceFilter = "all" | "voice" | "audio-file";

/** Filter only the retained entries supplied by the history owner. */
export function filterHistoryEntries(
  entries: readonly HistoryEntry[],
  query: string,
  source: HistorySourceFilter,
): HistoryEntry[] {
  const terms = query.toLowerCase().trim().split(/\s+/).filter(Boolean);
  const wantedSource =
    source === "voice"
      ? HistorySource.HistorySourceVoice
      : HistorySource.HistorySourceAudioFile;

  return entries.filter((entry) => {
    if (source !== "all" && entry.details.source !== wantedSource) return false;
    if (terms.length === 0) return true;
    const searchable = [
      entry.text,
      entry.rawText,
      entry.processedText ?? "",
      entry.details.fileName ?? "",
    ]
      .join("\n")
      .toLowerCase();
    return terms.every((term) => searchable.includes(term));
  });
}

/** Keep a selection while it remains visible; otherwise show the first match. */
export function selectedHistoryEntry(
  entries: readonly HistoryEntry[],
  id: number | null,
): HistoryEntry | undefined {
  return entries.find((entry) => entry.id === id) ?? entries[0];
}
