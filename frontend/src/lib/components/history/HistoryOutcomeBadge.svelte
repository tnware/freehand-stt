<script lang="ts">
  import { HistoryOutcome } from "$lib/state";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";

  let {
    outcome,
    fileLabel = "Transcribed",
  }: {
    outcome: HistoryOutcome;
    /** Compact history rows also identify the audio-file source. */
    fileLabel?: string;
  } = $props();

  const presentation = $derived.by(() => {
    switch (outcome) {
      case HistoryOutcome.HistoryCopyRequired:
        return { tone: "warning", label: "Copy required" } as const;
      case HistoryOutcome.HistoryFailed:
        return { tone: "danger", label: "Delivery failed" } as const;
      case HistoryOutcome.HistoryTranscribed:
        return { tone: "success", label: fileLabel } as const;
      case HistoryOutcome.HistoryCancelled:
        return { tone: "neutral", label: "Cancelled" } as const;
      case HistoryOutcome.HistoryInserted:
        return { tone: "success", label: "Inserted" } as const;
      default:
        return { tone: "neutral", label: "Unknown outcome" } as const;
    }
  });
</script>

<StatusBadge tone={presentation.tone}>{presentation.label}</StatusBadge>
