<script lang="ts">
  import TrashIcon from "@lucide/svelte/icons/trash-2";
  import HistoryList from "$lib/components/history/HistoryList.svelte";
  import { Switch } from "$lib/components/ui/switch";
  import { Button } from "$lib/components/ui/button";
  import SettingRow from "$lib/components/settings/SettingRow.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import type { HistoryEntry, HistoryTextVersion, Settings } from "$lib/state";

  let {
    settings = $bindable(),
    enabled,
    entries,
    onCopy,
    onCopyVersion,
    onDelete,
    onClear,
  }: {
    settings: Settings;
    enabled: boolean;
    entries: HistoryEntry[];
    onCopy: (id: number) => Promise<boolean>;
    onCopyVersion: (id: number, version: HistoryTextVersion) => Promise<boolean>;
    onDelete: (id: number) => Promise<boolean>;
    onClear: () => void;
  } = $props();
</script>

<div class="flex flex-col gap-4">
  <SettingsCard>
    <SettingRow
      title="Keep transcript history"
      description="Off by default. Keeps up to 20 finalized transcripts or 2 MiB in memory, whichever limit is reached first. Raw and processed versions share that limit."
    >
      {#snippet control()}
        <Switch
          id="history-enabled"
          bind:checked={settings.historyEnabled}
          aria-label="Keep transcript history"
        />
      {/snippet}
    </SettingRow>
  </SettingsCard>

  <div class="flex items-center justify-between gap-4 px-1">
    <p class="text-[13px] leading-relaxed text-muted-foreground">
      {enabled ? "Retention is active." : "Retention is off."} Turning it off or quitting clears
      every entry. History is never written to disk.
    </p>
    <Button variant="outline" size="sm" disabled={entries.length === 0} onclick={onClear}>
      <TrashIcon data-icon="inline-start" />
      Clear history
    </Button>
  </div>

  <div class="min-h-44 overflow-hidden rounded-xl bg-card shadow-lift">
    <HistoryList {entries} clamp={false} scrollable={false} {onCopy} {onCopyVersion} {onDelete} />
  </div>
</div>
