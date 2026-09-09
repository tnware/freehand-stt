<script lang="ts">
  import { transcriptReader, type TranscriptContent } from "$lib/utils/transcriptReader";
  import { cn } from "$lib/utils";

  let {
    content,
    label,
    class: className,
  }: {
    content: TranscriptContent;
    label: string;
    class?: string;
  } = $props();
</script>

<div
  use:transcriptReader={content}
  class={cn(
    "transcript-text select-text focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring",
    className,
  )}
  tabindex="0"
  role="textbox"
  aria-readonly="true"
  aria-multiline="true"
  aria-label={label}
  aria-keyshortcuts="Control+A Meta+A"
></div>

<style>
  .transcript-text :global(.diff-removed),
  .transcript-text :global(.diff-added) {
    border-radius: 0.2rem;
    box-decoration-break: clone;
    -webkit-box-decoration-break: clone;
  }
  .transcript-text :global(.diff-removed) {
    background-color: color-mix(in srgb, var(--destructive) 14%, transparent);
    color: var(--destructive);
    text-decoration: line-through;
    text-decoration-thickness: 1px;
  }
  .transcript-text :global(.diff-added) {
    background-color: color-mix(in srgb, var(--success) 17%, transparent);
    text-decoration: underline;
    text-decoration-color: color-mix(in srgb, var(--success) 55%, transparent);
    text-decoration-thickness: 2px;
    text-underline-offset: 0.14em;
  }
</style>
