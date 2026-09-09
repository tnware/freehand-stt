<script lang="ts">
  import TranscriptText from "$lib/components/common/TranscriptText.svelte";
  import ArrowLeftRightIcon from "@lucide/svelte/icons/arrow-left-right";
  import CheckIcon from "@lucide/svelte/icons/check";
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import ClipboardIcon from "@lucide/svelte/icons/clipboard";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  import * as Menu from "$lib/components/ui/dropdown-menu";
  import InfoIcon from "@lucide/svelte/icons/info";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import TrashIcon from "@lucide/svelte/icons/trash-2";
  import Volume2Icon from "@lucide/svelte/icons/volume-2";
  import { onDestroy } from "svelte";
  import { CopyFeedback } from "$lib/utils/copyFeedback.svelte";
  import TooltipButton from "$lib/components/ui/button/TooltipButton.svelte";
  import { Badge } from "$lib/components/ui/badge";
  import { Button, buttonVariants } from "$lib/components/ui/button";
  import * as HistoryService from "$bindings/history/service";
  import {
    HistoryOutcome,
    HistoryProcessingStatus,
    HistorySource,
    HistoryTextVersion,
    TTSPhase,
    TTSSource,
    type HistoryEntry,
    type TTSStatus,
  } from "$lib/state";
  import { cn } from "$lib/utils";
  import { processingProfileName } from "$lib/utils/processingProfiles";
  import { compareTranscriptText } from "$lib/utils/textDiff";

  let {
    entries,
    emptyTitle = "Nothing kept yet",
    emptyDescription = "The next finalized transcript will appear here.",
    scrollable = true,
    maxHeight,
    live,
    onCopy,
    onCopyVersion,
    onDelete,
    onCopyLive,
    ttsEnabled = false,
    ttsAvailable = true,
    ttsStatus,
    onListen,
    onListenLive,
  }: {
    entries: HistoryEntry[];
    emptyTitle?: string;
    emptyDescription?: string;
    /** Home owns an independent history scroller; settings scrolls as one page. */
    scrollable?: boolean;
    maxHeight?: string;
    /** Ephemeral presentation of an active file run; retained history remains owned by Go. */
    live?: {
      generation: number;
      text: string;
      fileName: string;
      status: string;
      characterCount: number;
      working: boolean;
      canCopy: boolean;
      failed: boolean;
    };
    onCopy: (id: number) => Promise<boolean>;
    onCopyVersion?: (id: number, version: HistoryTextVersion) => Promise<boolean>;
    onDelete: (id: number) => Promise<boolean>;
    onCopyLive?: () => Promise<boolean>;
    ttsEnabled?: boolean;
    ttsAvailable?: boolean;
    ttsStatus?: TTSStatus;
    onListen?: (id: number, version: HistoryTextVersion) => void;
    onListenLive?: () => void;
  } = $props();

  const outcomeLabel = (outcome: HistoryOutcome): string => {
    if (outcome === HistoryOutcome.HistoryCopyRequired) return "copy required";
    if (outcome === HistoryOutcome.HistoryFailed) return "delivery failed";
    if (outcome === HistoryOutcome.HistoryTranscribed) return "audio file";
    if (outcome === HistoryOutcome.HistoryCancelled) return "cancelled";
    return "inserted";
  };

  const outcomeDot = (outcome: HistoryOutcome): string => {
    if (outcome === HistoryOutcome.HistoryFailed) return "bg-destructive";
    if (outcome === HistoryOutcome.HistoryCopyRequired) return "bg-primary";
    if (outcome === HistoryOutcome.HistoryCancelled) return "bg-muted-foreground/50";
    return "bg-success";
  };

  const outcomeBadgeClass = (outcome: HistoryOutcome): string => {
    if (outcome === HistoryOutcome.HistoryFailed) return "";
    if (outcome === HistoryOutcome.HistoryCopyRequired) return "bg-primary/10 text-primary";
    if (outcome === HistoryOutcome.HistoryInserted) return "bg-success/10 text-success";
    return "";
  };

  const completedDateTime = (completedAt: string): string =>
    new Date(completedAt).toLocaleString([], {
      dateStyle: "medium",
      timeStyle: "short",
    });

  const completedLabel = (completedAt: string): string => {
    const completed = new Date(completedAt);
    const now = new Date();
    if (completed.toDateString() === now.toDateString()) {
      return completed.toLocaleTimeString([], {
        hour: "numeric",
        minute: "2-digit",
      });
    }
    const yesterday = new Date(now);
    yesterday.setDate(now.getDate() - 1);
    if (completed.toDateString() === yesterday.toDateString()) return "Yesterday";
    return completed.toLocaleDateString([], { month: "short", day: "numeric" });
  };

  const characterLabel = (count: number): string =>
    `${count.toLocaleString()} ${count === 1 ? "char" : "chars"}`;

  const compactModel = (model?: string): string => {
    const value = model?.trim() ?? "";
    return value ? (value.split("/").at(-1) ?? value) : "unknown model";
  };

  const sourceMetadata = (entry: HistoryEntry): string => {
    if (entry.details.source === HistorySource.HistorySourceAudioFile) {
      return entry.details.fileName || "audio file";
    }
    const checkpoints = entry.details.segmentCount ?? 0;
    return checkpoints > 0
      ? `${checkpoints.toLocaleString()} ${checkpoints === 1 ? "checkpoint" : "checkpoints"}`
      : "voice";
  };

  const hasProcessing = (entry: HistoryEntry): boolean =>
    entry.processingStatus !== HistoryProcessingStatus.HistoryProcessingNotRequested;

  const hasProcessedTranscript = (entry: HistoryEntry): boolean =>
    entry.processingStatus === HistoryProcessingStatus.HistoryProcessingCompleted &&
    Boolean(entry.processedText);

  const processingLabel = (entry: HistoryEntry): string => {
    if (entry.processingStatus === HistoryProcessingStatus.HistoryProcessingPending)
      return "raw + processing";
    if (entry.processingStatus === HistoryProcessingStatus.HistoryProcessingCompleted)
      return "raw + cleaned";
    return "raw only";
  };

  // A new leading result resets presentation in both history views. Updates to
  // the same result (including cleanup) preserve the reader's manual choices.
  const newestID = $derived(live ? undefined : entries[0]?.id);
  let disclosure = $derived<{ open: number[]; comparing: number[] }>({
    open: newestID === undefined ? [] : [newestID],
    comparing: [],
  });
  const expanded = (id: number): boolean => disclosure.open.includes(id);
  function toggleExpanded(id: number) {
    const wasExpanded = expanded(id);
    disclosure = {
      open: wasExpanded
        ? disclosure.open.filter((entryID) => entryID !== id)
        : [...disclosure.open, id],
      comparing: wasExpanded
        ? disclosure.comparing.filter((entryID) => entryID !== id)
        : disclosure.comparing,
    };
  }

  function toggleComparison(id: number) {
    disclosure = {
      open: expanded(id) ? disclosure.open : [...disclosure.open, id],
      comparing: disclosure.comparing.includes(id)
        ? disclosure.comparing.filter((entryID) => entryID !== id)
        : [...disclosure.comparing, id],
    };
  }

  const feedback = new CopyFeedback();
  onDestroy(() => feedback.dispose());

  async function copyText(entry: HistoryEntry, version: HistoryTextVersion) {
    await feedback.copy(`${entry.id}:${version}`, () =>
      version === HistoryTextVersion.HistoryTextFinal || !onCopyVersion
        ? onCopy(entry.id)
        : onCopyVersion(entry.id, version),
    );
  }

  async function copyLive() {
    if (onCopyLive) await feedback.copy("live", onCopyLive);
  }

  const detailsAvailable = (entry: HistoryEntry): boolean =>
    entry.processingStatus !== HistoryProcessingStatus.HistoryProcessingPending &&
    Boolean(entry.details.completedAt);
  let detailsError = $state("");
  async function openDetails(entry: HistoryEntry) {
    detailsError = "";
    try {
      await HistoryService.OpenDetails(entry.id);
    } catch (cause) {
      detailsError = String(cause);
    }
  }

  let scrollContainer: HTMLDivElement;
  let previousNewestID: number | undefined;
  $effect(() => {
    const newestID = entries[0]?.id;
    if (
      scrollable &&
      newestID !== undefined &&
      previousNewestID !== undefined &&
      newestID > previousNewestID
    ) {
      scrollContainer.scrollTop = 0;
    }
    previousNewestID = newestID;
  });
</script>

<div
  bind:this={scrollContainer}
  style:max-height={maxHeight}
  class={cn("min-h-0 flex-1", scrollable && "overflow-y-auto overscroll-contain")}
>
  {#if entries.length === 0 && !live}
    <div class="flex min-h-40 flex-col items-center justify-center px-6 py-9 text-center">
      <p class="text-sm font-medium">{emptyTitle}</p>
      <p class="mt-1.5 max-w-md text-[12.5px] leading-relaxed text-muted-foreground">
        {emptyDescription}
      </p>
    </div>
  {:else}
    <div class="flex flex-col">
      {#if live}
        <article
          class="bg-primary/5 px-4 pt-3.5 pb-2"
          aria-label={live.working ? "Live audio file transcript" : "Audio file transcript result"}
        >
          <div
            class="-mx-4 -mt-3.5 flex min-h-8 min-w-0 items-center justify-between gap-3 border-b border-hairline bg-layer-fill px-4 py-1"
          >
            <div class="flex min-w-0 items-center gap-2">
              <span
                class={cn(
                  "size-2 shrink-0 rounded-full",
                  live.failed ? "bg-destructive" : live.working ? "bg-primary" : "bg-success",
                )}
              ></span>
              <Badge variant={live.failed ? "destructive" : "secondary"} class="font-mono">
                {#if live.working}
                  <LoaderCircleIcon class="animate-spin motion-reduce:animate-none" />
                {/if}
                {live.status}
              </Badge>
              <span class="truncate text-xs font-medium">{live.fileName}</span>
            </div>
            <span class="shrink-0 font-mono text-[10px] text-muted-foreground">
              {characterLabel(live.characterCount)}
            </span>
          </div>

          {#if live.text}
            <TranscriptText
              content={{ key: `file:${live.generation}`, text: live.text }}
              label="Audio file transcript"
              class="mt-2.5 min-h-5 mx-auto w-full max-w-[76ch] text-sm leading-7 break-words whitespace-pre-wrap"
            />
          {:else}
            <p class="mt-2.5 min-h-5 text-sm leading-7 text-muted-foreground">
              Transcript text will appear here as it arrives.
            </p>
          {/if}

          <div
            class="history-footer mt-1.5 flex min-h-6 min-w-0 items-center justify-between gap-2"
          >
            <span class="font-mono text-[10px] text-primary">audio file · {live.status}</span>
            <div class="flex items-center">
              {#if ttsEnabled && onListenLive && !live.working}
                <TooltipButton
                  variant="ghost"
                  size="icon-xs"
                  disabled={!ttsAvailable}
                  label="Listen to audio file transcript"
                  onclick={onListenLive}
                >
                  {#if ttsStatus?.source === TTSSource.SourceFile && ttsStatus.phase === TTSPhase.Generating}
                    <LoaderCircleIcon class="animate-spin motion-reduce:animate-none" />
                  {:else}
                    <Volume2Icon />
                  {/if}
                </TooltipButton>
              {/if}
              <TooltipButton
                variant="ghost"
                size="icon-xs"
                disabled={!live.canCopy}
                label={live.canCopy
                  ? "Copy audio file transcript"
                  : "Copy is available when transcription finishes"}
                onclick={() => void copyLive()}
              >
                {#if feedback.key === "live"}
                  <CheckIcon class="text-success" />
                {:else}
                  <ClipboardIcon />
                {/if}
              </TooltipButton>
            </div>
          </div>
        </article>
      {/if}

      {#each entries as entry (entry.id)}
        {@const isExpanded = expanded(entry.id)}
        {@const hasCleaned = hasProcessedTranscript(entry)}
        {@const isComparing = isExpanded && hasCleaned && disclosure.comparing.includes(entry.id)}
        {@const finalVersion = hasCleaned
          ? HistoryTextVersion.HistoryTextProcessed
          : HistoryTextVersion.HistoryTextFinal}
        <article
          class={cn(
            "history-entry group px-4 pt-3.5 pb-2 transition-colors",
            isExpanded && "bg-muted/20",
          )}
        >
          <div
            class={cn(
              "history-row-header -mx-4 -mt-3.5 flex min-h-8 w-[calc(100%+2rem)] min-w-0 items-center border-b border-hairline bg-layer-fill",
              isExpanded && "sticky top-0 z-10",
            )}
          >
            <button
              type="button"
              class="history-disclosure flex min-h-8 min-w-0 flex-1 items-center gap-3 px-4 py-1 text-left"
              aria-label={`${isExpanded ? "Collapse" : "Expand"} transcript from ${completedDateTime(entry.completedAt)}`}
              aria-expanded={isExpanded}
              aria-controls={`history-entry-${entry.id}-content`}
              onclick={() => toggleExpanded(entry.id)}
            >
              <span class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
                <span class={cn("size-2 shrink-0 rounded-full", outcomeDot(entry.outcome))}></span>
                {#if entry.id === newestID}<span class="text-xs font-medium text-muted-foreground"
                    >Latest</span
                  >{/if}
                <time
                  datetime={entry.completedAt}
                  title={completedDateTime(entry.completedAt)}
                  class="font-mono text-[11px] font-medium"
                >
                  {completedLabel(entry.completedAt)}
                </time>
                <Badge
                  variant={entry.outcome === HistoryOutcome.HistoryFailed
                    ? "destructive"
                    : "secondary"}
                  class={cn("font-mono", outcomeBadgeClass(entry.outcome))}
                >
                  {outcomeLabel(entry.outcome)}
                </Badge>
                {#if hasProcessing(entry)}
                  <Badge variant="secondary" class="font-mono text-primary">
                    {processingLabel(entry)}
                  </Badge>
                {/if}
              </span>
              <span
                class="disclosure-affordance grid size-7 shrink-0 place-items-center rounded-md text-muted-foreground"
              >
                <ChevronDownIcon
                  class={cn(
                    "size-4 transition-transform duration-150 motion-reduce:transition-none",
                    isExpanded && "rotate-180",
                  )}
                />
              </span>
            </button>
            {#if hasCleaned}
              <Button
                variant="ghost"
                size="xs"
                class={cn("mr-2 min-w-[4.75rem]", isComparing && "bg-control-fill-active")}
                aria-label={isComparing
                  ? "Show the final transcript"
                  : "Compare raw and cleaned transcripts"}
                aria-pressed={isComparing}
                onclick={() => toggleComparison(entry.id)}
              >
                <ArrowLeftRightIcon data-icon="inline-start" />
                Compare
              </Button>
            {/if}
          </div>

          <div id={`history-entry-${entry.id}-content`}>
            {#if isExpanded}
              {#if isComparing}
                {@const processedText = entry.processedText ?? entry.text}
                {@const comparison = compareTranscriptText(entry.rawText, processedText)}
                <div
                  class="comparison-layout mt-3 overflow-hidden rounded-lg border border-hairline bg-background/35"
                >
                  <section class="comparison-panel px-3 py-2.5" aria-label="Raw transcript">
                    <div class="mb-1.5 flex items-center justify-between gap-3">
                      <span
                        class="truncate font-mono text-[10px] tracking-[0.04em] text-muted-foreground uppercase"
                      >
                        Raw · {compactModel(entry.details.model)}
                      </span>
                      <TooltipButton
                        variant="ghost"
                        size="icon-xs"
                        label={feedback.key === `${entry.id}:${HistoryTextVersion.HistoryTextRaw}`
                          ? "Raw transcript copied"
                          : "Copy raw transcript"}
                        onclick={() => void copyText(entry, HistoryTextVersion.HistoryTextRaw)}
                      >
                        {#if feedback.key === `${entry.id}:${HistoryTextVersion.HistoryTextRaw}`}
                          <CheckIcon class="text-success" />
                        {:else}
                          <ClipboardIcon />
                        {/if}
                      </TooltipButton>
                    </div>
                    <TranscriptText
                      content={{
                        key: `${entry.id}:raw`,
                        text: entry.rawText,
                        parts: comparison.raw,
                      }}
                      label="Raw transcript text"
                      class="mx-auto w-full max-w-[76ch] text-sm leading-7 break-words whitespace-pre-wrap"
                    />
                  </section>

                  <section
                    class="comparison-panel comparison-cleaned border-t border-hairline bg-muted/25 px-3 py-2.5"
                    aria-label="Cleaned transcript"
                  >
                    <div class="mb-1.5 flex items-center justify-between gap-3">
                      <span
                        class="truncate font-mono text-[10px] tracking-[0.04em] text-muted-foreground uppercase"
                      >
                        Cleaned · {compactModel(entry.details.processing.model)}
                        {#if entry.details.processing.preset}
                          · {processingProfileName([], entry.details.processing.preset)}
                        {/if}
                      </span>
                      <TooltipButton
                        variant="ghost"
                        size="icon-xs"
                        label={feedback.key ===
                        `${entry.id}:${HistoryTextVersion.HistoryTextProcessed}`
                          ? "Cleaned transcript copied"
                          : "Copy cleaned transcript"}
                        onclick={() =>
                          void copyText(entry, HistoryTextVersion.HistoryTextProcessed)}
                      >
                        {#if feedback.key === `${entry.id}:${HistoryTextVersion.HistoryTextProcessed}`}
                          <CheckIcon class="text-success" />
                        {:else}
                          <ClipboardIcon />
                        {/if}
                      </TooltipButton>
                    </div>
                    <TranscriptText
                      content={{
                        key: `${entry.id}:processed`,
                        text: processedText,
                        parts: comparison.processed,
                      }}
                      label="Cleaned transcript text"
                      class="mx-auto w-full max-w-[76ch] text-sm leading-7 break-words whitespace-pre-wrap"
                    />
                  </section>
                </div>
              {:else}
                <TranscriptText
                  content={{ key: String(entry.id), text: entry.text }}
                  label={`Transcript from ${completedDateTime(entry.completedAt)}`}
                  class="mt-2.5 mx-auto w-full max-w-[76ch] text-sm leading-7 break-words whitespace-pre-wrap"
                />
              {/if}
            {:else}
              <button
                type="button"
                class="transcript-toggle -mx-2 mt-2 block w-[calc(100%+1rem)] rounded-md px-2 text-left"
                aria-label={`Expand transcript from ${completedDateTime(entry.completedAt)}`}
                aria-expanded="false"
                aria-controls={`history-entry-${entry.id}-content`}
                onclick={() => toggleExpanded(entry.id)}
              >
                <span class="line-clamp-1 text-sm leading-7 break-words">{entry.text}</span>
              </button>
            {/if}
          </div>

          {#if hasProcessing(entry) && !hasCleaned}
            <p
              class="mt-2 flex items-start gap-1.5 text-xs leading-relaxed text-muted-foreground"
              role="status"
            >
              {#if entry.processingStatus === HistoryProcessingStatus.HistoryProcessingPending}
                <LoaderCircleIcon
                  class="mt-0.5 size-3.5 shrink-0 animate-spin motion-reduce:animate-none"
                />
                <span>Cleaning up transcript…</span>
              {:else}
                <CircleAlertIcon class="mt-0.5 size-3.5 shrink-0 text-warning" />
                <span class="min-w-0 break-words"
                  >{entry.processingMessage || "The raw transcript was kept."}</span
                >
              {/if}
            </p>
          {/if}

          <div
            class="history-footer mt-1.5 flex min-h-6 min-w-0 items-center justify-between gap-2"
          >
            <div
              class="flex min-w-0 flex-1 items-center gap-1.5 font-mono text-[10px] text-muted-foreground"
            >
              <span class="shrink-0">{characterLabel(entry.characterCount)}</span>
              <span class="shrink-0" aria-hidden="true">·</span>
              <span class="max-w-48 truncate" title={sourceMetadata(entry)}
                >{sourceMetadata(entry)}</span
              >
            </div>

            <div class="history-actions flex shrink-0 items-center">
              <div class="history-utilities flex items-center">
                {#if ttsEnabled && onListen}
                  <TooltipButton
                    variant="ghost"
                    size="icon-xs"
                    disabled={!ttsAvailable}
                    label={ttsStatus?.historyID === entry.id &&
                    ttsStatus.phase === TTSPhase.Generating
                      ? "Generating speech for this transcript"
                      : "Listen to transcript"}
                    onclick={() => onListen(entry.id, finalVersion)}
                  >
                    {#if ttsStatus?.historyID === entry.id && ttsStatus.phase === TTSPhase.Generating}
                      <LoaderCircleIcon class="animate-spin motion-reduce:animate-none" />
                    {:else}
                      <Volume2Icon />
                    {/if}
                  </TooltipButton>
                {/if}
                <TooltipButton
                  variant="ghost"
                  size="icon-xs"
                  class="text-primary"
                  label={feedback.key === `${entry.id}:${finalVersion}`
                    ? "Transcript copied"
                    : hasCleaned
                      ? "Copy cleaned transcript"
                      : "Copy transcript"}
                  onclick={() => void copyText(entry, finalVersion)}
                >
                  {#if feedback.key === `${entry.id}:${finalVersion}`}
                    <CheckIcon class="text-success" />
                  {:else}
                    <ClipboardIcon />
                  {/if}
                </TooltipButton>
                <Menu.Root>
                  <Menu.Trigger
                    aria-label="Transcript actions"
                    class={buttonVariants({ variant: "ghost", size: "icon-xs" })}
                    ><EllipsisIcon /></Menu.Trigger
                  >
                  <Menu.Content align="end" class="w-64 max-w-[calc(100vw-24px)]">
                    <Menu.Item
                      disabled={!detailsAvailable(entry)}
                      onclick={() => void openDetails(entry)}
                      class="gap-2.5 px-3 py-2.5"
                    >
                      <InfoIcon />Transcription details
                    </Menu.Item>
                    <Menu.Separator />
                    <Menu.Item
                      variant="destructive"
                      onclick={() => void onDelete(entry.id)}
                      class="gap-2.5 px-3 py-2.5"
                    >
                      <TrashIcon />Remove from history
                    </Menu.Item>
                  </Menu.Content>
                </Menu.Root>
              </div>
            </div>
          </div>
        </article>
      {/each}
    </div>
  {/if}
</div>

{#if detailsError}
  <p role="alert" class="px-4 py-2 text-xs text-destructive">{detailsError}</p>
{/if}

<style>
  article {
    container-type: inline-size;
  }
  .history-disclosure:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: -2px;
  }
  .transcript-toggle:focus-visible {
    outline: 2px solid var(--ring);
    outline-offset: 2px;
  }
  .history-disclosure {
    transition:
      background-color 100ms ease,
      color 100ms ease;
  }
  .history-disclosure:hover {
    background-color: var(--control-fill-hover);
  }
  .history-disclosure:hover .disclosure-affordance {
    color: var(--foreground);
  }
  /* At the default 1080 px window the history column has enough room for two
     readable transcript columns. Narrow layouts keep the vertical flow. */
  @container (min-width: 540px) {
    .comparison-layout {
      display: grid;
      grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
      align-items: stretch;
    }
    .comparison-cleaned {
      border-top: 0;
      border-left: 1px solid var(--hairline);
    }
  }
  @container (max-width: 319px) {
    .history-footer {
      align-items: flex-start;
      flex-direction: column;
    }
    .history-actions {
      width: 100%;
      align-self: stretch;
    }
    .history-utilities {
      margin-left: auto;
    }
  }
</style>
