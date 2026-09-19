<script lang="ts">
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import { Input } from "$lib/components/ui/input";
  import { Button } from "$lib/components/ui/button";
  import { HistorySource, type HistoryEntry } from "$bindings/history";
  import type { HistorySourceFilter } from "$lib/utils/historyBrowser";
  import { cn } from "$lib/utils";

  let {
    entries,
    totalEntries,
    enabled,
    clearing = false,
    selectedID,
    query = $bindable(""),
    source = $bindable<HistorySourceFilter>("all"),
    onSelect,
    onOpenSettings,
    onClear,
  }: {
    entries: HistoryEntry[];
    totalEntries: HistoryEntry[];
    enabled: boolean;
    clearing?: boolean;
    selectedID?: number;
    query?: string;
    source?: HistorySourceFilter;
    onSelect: (id: number) => void;
    onOpenSettings: () => void;
    onClear: () => void;
  } = $props();
  const uid = $props.id();

  const filters = $derived([
    {
      value: "all" as const,
      label: "All",
      name: "All sources",
      count: totalEntries.length,
    },
    {
      value: "voice" as const,
      label: "Voice",
      name: "Voice",
      count: totalEntries.filter(
        (entry) => entry.details.source === HistorySource.HistorySourceVoice,
      ).length,
    },
    {
      value: "audio-file" as const,
      label: "Audio files",
      name: "Audio files",
      count: totalEntries.filter(
        (entry) =>
          entry.details.source === HistorySource.HistorySourceAudioFile,
      ).length,
    },
  ]);

  function sourceLabel(entry: HistoryEntry): "Voice" | "Audio file" {
    return entry.details.source === HistorySource.HistorySourceAudioFile
      ? "Audio file"
      : "Voice";
  }

  function completedDateTime(value: string): string {
    return new Date(value).toLocaleString([], {
      dateStyle: "medium",
      timeStyle: "short",
    });
  }

  function preview(entry: HistoryEntry): string {
    return (
      entry.text.trim().replace(/\s+/g, " ").slice(0, 160) ||
      "No transcript text"
    );
  }
</script>

<aside
  class="history-sidebar flex min-h-0 w-[252px] shrink-0 flex-col overflow-hidden border-r border-hairline bg-layer-fill"
  aria-label="History browser"
>
  <SidebarHeader title="History" />
  <div class="shrink-0 space-y-2 border-b border-hairline px-3.5 py-3">
    <Input
      type="search"
      bind:value={query}
      aria-label="Search history"
      placeholder="Search history"
    />
    <div
      class="flex flex-wrap items-center gap-1"
      role="group"
      aria-label="Filter history by source"
    >
      {#each filters as filter (filter.value)}
        <button
          type="button"
          class={cn(
            "flex min-h-7 min-w-0 items-center justify-center gap-1 rounded-sm px-1.5 text-xs transition-colors hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring",
            source === filter.value
              ? "bg-accent-wash-strong text-accent-text"
              : "text-muted-foreground",
          )}
          aria-label={filter.name}
          aria-describedby={`${uid}-${filter.value}-count`}
          aria-pressed={source === filter.value}
          onclick={() => (source = filter.value)}
        >
          <span class="whitespace-nowrap">{filter.label}</span>
          <span class="text-xs tabular-nums" aria-hidden="true"
            >{filter.count}</span
          >
          <span id={`${uid}-${filter.value}-count`} class="sr-only"
            >{filter.count}
            {filter.count === 1 ? "transcript" : "transcripts"}</span
          >
        </button>
      {/each}
    </div>
  </div>

  <div
    class="shrink-0 px-3.5 py-2 text-xs text-muted-foreground"
    role="status"
    aria-atomic="true"
  >
    {entries.length === totalEntries.length
      ? `${entries.length} ${entries.length === 1 ? "transcript" : "transcripts"}`
      : `${entries.length} of ${totalEntries.length} transcripts`}
  </div>
  <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain">
    {#if entries.length}
      <ul class="m-0 list-none p-0">
        {#each entries as entry (entry.id)}
          {@const dateTime = completedDateTime(entry.completedAt)}
          <li>
            <button
              type="button"
              class={cn(
                "flex w-full min-w-0 flex-col gap-1 border-l-2 px-3 py-2.5 text-left transition-colors hover:bg-subtle-fill-hover focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-ring",
                selectedID === entry.id
                  ? "border-accent-text bg-accent-wash-strong"
                  : "border-transparent",
              )}
              aria-label={`View ${sourceLabel(entry)} transcript from ${dateTime}`}
              aria-current={selectedID === entry.id ? "true" : undefined}
              onclick={() => onSelect(entry.id)}
            >
              <span
                class="flex w-full min-w-0 items-center justify-between gap-2"
              >
                <span
                  class="shrink-0 text-xs font-medium text-secondary-foreground"
                  >{sourceLabel(entry)}</span
                >
                <time
                  datetime={entry.completedAt}
                  class="truncate text-xs text-muted-foreground"
                  title={dateTime}>{dateTime}</time
                >
              </span>
              {#if entry.details.fileName}
                <span
                  class="w-full truncate text-xs text-muted-foreground"
                  title={entry.details.fileName}>{entry.details.fileName}</span
                >
              {/if}
              <span
                class="line-clamp-2 w-full break-words text-[13px] leading-relaxed text-secondary-foreground"
                >{preview(entry)}</span
              >
            </button>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="px-3.5 py-3 text-xs leading-relaxed text-muted-foreground">
        {totalEntries.length
          ? "No transcripts match these filters."
          : enabled
            ? "Completed transcripts will appear here."
            : "Turn history on to keep transcripts for this session."}
      </p>
    {/if}
  </div>

  <footer class="shrink-0 space-y-2 border-t border-hairline px-3.5 py-3">
    <p class="text-xs leading-relaxed text-muted-foreground">
      {enabled ? "Kept in memory until you quit." : "History is turned off."}
    </p>
    <div class="flex flex-wrap items-center gap-1">
      <Button variant="outline" size="xs" onclick={onOpenSettings}
        >History settings</Button
      >
      <Button
        variant="ghost"
        size="xs"
        disabled={clearing || totalEntries.length === 0}
        aria-busy={clearing}
        onclick={onClear}>Clear history</Button
      >
    </div>
  </footer>
</aside>
