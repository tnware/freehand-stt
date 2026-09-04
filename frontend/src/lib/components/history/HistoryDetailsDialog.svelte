<script lang="ts">
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import CircleMinusIcon from "@lucide/svelte/icons/circle-minus";
  import CircleXIcon from "@lucide/svelte/icons/circle-x";
  import ClockIcon from "@lucide/svelte/icons/clock";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import { Badge } from "$lib/components/ui/badge";
  import * as Dialog from "$lib/components/ui/dialog";
  import { Separator } from "$lib/components/ui/separator";
  import {
    HistoryOutcome,
    HistoryProcessingStatus,
    HistoryResponseMode,
    HistorySource,
    InsertionMode,
    RecordingMode,
    type HistoryEntry,
  } from "$lib/state";
  import { processingProfileName } from "$lib/utils/processingProfiles";
  import HistoryResponseMetadata from "./HistoryResponseMetadata.svelte";

  let {
    entry,
    returnFocus = null,
    onClose,
  }: {
    entry?: HistoryEntry;
    returnFocus?: HTMLElement | null;
    onClose: () => void;
  } = $props();

  function restoreFocus(event: Event) {
    const target = returnFocus;
    if (!target?.isConnected) return;
    event.preventDefault();
    queueMicrotask(() => target.focus());
  }

  const dateTime = (value?: string): string =>
    value
      ? new Date(value).toLocaleString([], {
          dateStyle: "medium",
          timeStyle: "medium",
        })
      : "Not available";

  const duration = (milliseconds?: number): string => {
    if (milliseconds === undefined || milliseconds < 0) return "Not available";
    if (milliseconds < 1000) return `${milliseconds.toLocaleString()} ms`;
    const seconds = milliseconds / 1000;
    if (seconds < 60) return `${seconds.toFixed(seconds < 10 ? 1 : 0)} s`;
    const minutes = Math.floor(seconds / 60);
    const remainder = Math.round(seconds % 60);
    return `${minutes}m ${remainder}s`;
  };

  const bytes = (value?: number): string => {
    if (value === undefined || value < 0) return "Not available";
    if (value < 1024) return `${value.toLocaleString()} B`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`;
    return `${(value / (1024 * 1024)).toFixed(1)} MiB`;
  };

  const sourceLabel = (source: HistorySource): string =>
    source === HistorySource.HistorySourceAudioFile ? "Audio file" : "Voice dictation";

  const insertionModeLabel = (value?: string): string => {
    if (value === InsertionMode.DirectInput) return "Direct input";
    if (value === InsertionMode.ManualCopy) return "Manual copy";
    return "Not available";
  };

  const outcomeLabel = (outcome: HistoryOutcome): string => {
    if (outcome === HistoryOutcome.HistoryCopyRequired) return "Copy required";
    if (outcome === HistoryOutcome.HistoryFailed) return "Delivery failed";
    if (outcome === HistoryOutcome.HistoryTranscribed) return "Transcribed";
    if (outcome === HistoryOutcome.HistoryCancelled) return "Cancelled";
    return "Inserted";
  };

  const processingLabel = (status: HistoryProcessingStatus): string => {
    if (status === HistoryProcessingStatus.HistoryProcessingCompleted) return "Completed";
    if (status === HistoryProcessingStatus.HistoryProcessingFailed) return "Raw fallback";
    if (status === HistoryProcessingStatus.HistoryProcessingCancelled) return "Cancelled";
    if (status === HistoryProcessingStatus.HistoryProcessingPending) return "Pending";
    return "Not requested";
  };

  const badgeVariant = (
    outcome: HistoryOutcome,
  ): "default" | "secondary" | "destructive" => {
    if (outcome === HistoryOutcome.HistoryFailed) return "destructive";
    if (
      outcome === HistoryOutcome.HistoryCancelled ||
      outcome === HistoryOutcome.HistoryTranscribed
    )
      return "secondary";
    return "default";
  };

  const outcomeBadgeClass = (outcome: HistoryOutcome): string => {
    if (
      outcome === HistoryOutcome.HistoryInserted ||
      outcome === HistoryOutcome.HistoryTranscribed
    )
      return "bg-success/10 text-success";
    if (outcome === HistoryOutcome.HistoryCopyRequired)
      return "bg-primary/10 text-primary";
    return "";
  };

  const processingVariant = (
    status: HistoryProcessingStatus,
  ): "default" | "secondary" => {
    if (status === HistoryProcessingStatus.HistoryProcessingFailed) return "secondary";
    if (
      status === HistoryProcessingStatus.HistoryProcessingCancelled ||
      status === HistoryProcessingStatus.HistoryProcessingNotRequested
    )
      return "secondary";
    return "default";
  };

  const processingBadgeClass = (status: HistoryProcessingStatus): string => {
    if (status === HistoryProcessingStatus.HistoryProcessingCompleted)
      return "bg-success/10 text-success";
    if (status === HistoryProcessingStatus.HistoryProcessingFailed)
      return "bg-warning/10 text-warning";
    return "";
  };

  type FieldStatus = "positive" | "inactive" | "warning" | "informational";

  const fieldStatusClass = (status: FieldStatus): string => {
    if (status === "positive") return "bg-success/10 text-success";
    if (status === "warning") return "bg-warning/10 text-warning";
    if (status === "informational") return "bg-primary/10 text-primary";
    return "bg-muted text-muted-foreground";
  };
</script>

{#snippet durationValue(value?: number, suffix?: string)}
  <span
    class="inline-flex h-5 items-center gap-1.5 rounded-md border border-hairline bg-layer-fill px-2 font-mono text-[10.5px] leading-none font-medium tabular-nums text-foreground"
  >
    <ClockIcon class="size-3 text-muted-foreground" aria-hidden="true" />
    {duration(value)}{suffix ? ` ${suffix}` : ""}
  </span>
{/snippet}

{#snippet endpointValue(value?: string)}
  {#if value}
    <span
      class="inline-block min-h-5 max-w-full rounded-md border border-hairline bg-muted/45 px-2.5 py-[2px] font-mono text-[10.5px] leading-[14px] font-normal text-foreground break-all"
    >
      {value}
    </span>
  {:else}
    <span class="font-normal text-muted-foreground">Not available</span>
  {/if}
{/snippet}

{#snippet modelValue(value?: string)}
  {#if value}
    <span
      class="inline-block min-h-5 max-w-full rounded-md bg-primary/10 px-2.5 py-[2px] font-mono text-[10.5px] leading-[14px] font-medium text-primary break-all"
    >
      {value}
    </span>
  {:else}
    <span class="font-normal text-muted-foreground">Not available</span>
  {/if}
{/snippet}

{#snippet statusValue(label: string, status: FieldStatus)}
  <span
    class={`inline-flex h-5 items-center gap-1.5 rounded-full px-2 text-[10.5px] leading-none font-medium whitespace-nowrap ${fieldStatusClass(status)}`}
  >
    {#if status === "positive"}
      <CircleCheckIcon class="size-3" aria-hidden="true" />
    {:else if status === "warning"}
      <TriangleAlertIcon class="size-3" aria-hidden="true" />
    {:else if status === "informational"}
      <CircleMinusIcon class="size-3" aria-hidden="true" />
    {:else}
      <CircleXIcon class="size-3" aria-hidden="true" />
    {/if}
    {label}
  </span>
{/snippet}

{#snippet characterValue(value?: number, suffix?: string)}
  <span
    class="inline-flex h-5 items-center gap-1.5 rounded-md border border-hairline bg-layer-fill px-2 font-mono text-[10.5px] leading-none font-semibold tabular-nums text-foreground shadow-[inset_0_-1px_0_var(--hairline)]"
  >
    {(value ?? 0).toLocaleString()}
    {#if suffix}
      <span class="font-normal text-muted-foreground">{suffix}</span>
    {/if}
  </span>
{/snippet}

<Dialog.Root open={entry !== undefined} onOpenChange={(open) => !open && onClose()}>
  {#if entry}
    {@const details = entry.details}
    {@const processing = details.processing}
    <Dialog.Content
      class="flex max-h-[calc(100vh-2rem)] flex-col gap-0 overflow-hidden bg-dialog-surface p-0 shadow-xl ring-dialog-stroke sm:max-w-[560px]"
      onCloseAutoFocus={restoreFocus}
    >
      <Dialog.Header class="shrink-0 border-b border-hairline bg-layer-fill px-5 py-3.5 pr-14">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <Dialog.Title class="text-base font-semibold">Transcription details</Dialog.Title>
            <Dialog.Description class="mt-1 font-mono text-[10.5px]">
              {sourceLabel(details.source)} · run #{entry.id.toLocaleString()}
            </Dialog.Description>
          </div>
          <Badge
            variant={badgeVariant(entry.outcome)}
            class={outcomeBadgeClass(entry.outcome)}
          >
            {outcomeLabel(entry.outcome)}
          </Badge>
        </div>
      </Dialog.Header>

      <div class="flex min-h-0 flex-col gap-4 overflow-y-auto p-5 [&_dd]:my-0.5 [&_dd]:min-w-0 [&_dd]:font-medium [&_dd]:text-foreground/90 [&_dt]:my-0.5 [&_dt]:text-[12px]">
        <section class="flex flex-col gap-3" aria-labelledby="run-details-heading">
          <div class="flex items-center gap-2">
            <span class="size-1.5 rounded-full bg-primary" aria-hidden="true"></span>
            <h3 id="run-details-heading" class="text-[13px] font-semibold">Run</h3>
          </div>
          <dl class="grid grid-cols-[minmax(8rem,auto)_minmax(0,1fr)] gap-x-5 text-[12.5px]">
            <dt class="text-muted-foreground">Started</dt>
            <dd class="text-right break-words">{dateTime(details.startedAt)}</dd>
            <dt class="text-muted-foreground">Completed</dt>
            <dd class="text-right break-words">{dateTime(details.completedAt)}</dd>
            <dt class="text-muted-foreground">Total elapsed</dt>
            <dd class="text-right">{@render durationValue(details.elapsedMilliseconds)}</dd>
            <dt class="text-muted-foreground">
              {details.source === HistorySource.HistorySourceVoice && details.silenceTrimming
                ? "Audio submitted"
                : "Audio length"}
            </dt>
            <dd class="text-right">{@render durationValue(details.audioDurationMilliseconds)}</dd>
            <dt class="text-muted-foreground">Characters delivered</dt>
            <dd class="text-right">
              {@render characterValue(processing.deliveredCharacters ?? entry.characterCount)}
            </dd>
            <dt class="text-muted-foreground">Delivery mode</dt>
            <dd class="text-right">{insertionModeLabel(details.insertionMode)}</dd>
          </dl>
        </section>

        <Separator />

        <section class="flex flex-col gap-3" aria-labelledby="request-details-heading">
          <div class="flex items-center gap-2">
            <span class="size-1.5 rounded-full bg-primary" aria-hidden="true"></span>
            <h3 id="request-details-heading" class="text-[13px] font-semibold">Speech recognition</h3>
          </div>
          <dl class="grid grid-cols-[minmax(8rem,auto)_minmax(0,1fr)] gap-x-5 text-[12.5px]">
            <dt class="text-muted-foreground">Server</dt>
            <dd class="text-right">{@render endpointValue(details.server)}</dd>
            <dt class="text-muted-foreground">Route</dt>
            <dd class="text-right break-all">{details.route}</dd>
            <dt class="text-muted-foreground">Authentication</dt>
            <dd class="text-right">{details.authenticationMode === "none" ? "None" : "API key"}</dd>
            <dt class="text-muted-foreground">Model</dt>
            <dd class="text-right">{@render modelValue(details.model)}</dd>
            <dt class="text-muted-foreground">Language</dt>
            <dd class="text-right">{details.language || "Automatic"}</dd>
            <dt class="text-muted-foreground">Response</dt>
            <dd class="text-right">
              {#if details.responseMode === HistoryResponseMode.HistoryResponseStreamed}
                {@render statusValue(
                  details.buffered ? "Streamed · buffered" : "Streamed",
                  details.buffered ? "warning" : "informational",
                )}
              {:else}
                {@render statusValue("Completed", "positive")}
              {/if}
            </dd>
            {#if details.streamFallbackReason}
              <dt class="text-muted-foreground">Streaming fallback</dt>
              <dd class="text-right">{details.streamFallbackReason.replaceAll("_", " ")}</dd>
            {/if}
            <dt class="text-muted-foreground">Request time</dt>
            <dd class="text-right">{@render durationValue(details.transcriptionMilliseconds)}</dd>
            {#if details.requestTimeoutSeconds}
              <dt class="text-muted-foreground">Request timeout</dt>
              <dd class="text-right">{@render durationValue(details.requestTimeoutSeconds * 1000)}</dd>
            {/if}
            {#if details.errorKind}
              <dt class="text-muted-foreground">Terminal error</dt>
              <dd class="text-right">{details.errorKind}</dd>
            {/if}
          </dl>
          {#if details.transcription}
            <HistoryResponseMetadata response={details.transcription} stage="transcription" />
          {/if}
        </section>

        <Separator />

        {#if details.source === HistorySource.HistorySourceAudioFile}
          <section class="flex flex-col gap-3" aria-labelledby="source-details-heading">
            <div class="flex items-center gap-2">
              <span class="size-1.5 rounded-full bg-primary" aria-hidden="true"></span>
              <h3 id="source-details-heading" class="text-[13px] font-semibold">Audio file</h3>
            </div>
            <dl class="grid grid-cols-[minmax(8rem,auto)_minmax(0,1fr)] gap-x-5 text-[12.5px]">
              <dt class="text-muted-foreground">Filename</dt>
              <dd class="text-right break-all">{details.fileName || "Not available"}</dd>
              <dt class="text-muted-foreground">File size</dt>
              <dd class="text-right">{bytes(details.fileSize)}</dd>
              <dt class="text-muted-foreground">Upload time</dt>
              <dd class="text-right">{@render durationValue(details.uploadMilliseconds)}</dd>
            </dl>
          </section>
        {:else}
          <section class="flex flex-col gap-3" aria-labelledby="source-details-heading">
            <div class="flex items-center gap-2">
              <span class="size-1.5 rounded-full bg-primary" aria-hidden="true"></span>
              <h3 id="source-details-heading" class="text-[13px] font-semibold">Voice capture</h3>
            </div>
            <dl class="grid grid-cols-[minmax(8rem,auto)_minmax(0,1fr)] gap-x-5 text-[12.5px]">
              <dt class="text-muted-foreground">Microphone</dt>
              <dd class="text-right break-words">{details.microphone || "Not available"}</dd>
              <dt class="text-muted-foreground">Recording control</dt>
              <dd class="text-right">
                {details.recordingMode === RecordingMode.RecordingHold ? "Hold to talk" : "Toggle"}
              </dd>
              <dt class="text-muted-foreground">Recording length</dt>
              <dd class="text-right">{@render durationValue(details.captureDurationMilliseconds)}</dd>
              <dt class="text-muted-foreground">VAD</dt>
              <dd class="text-right">
                {@render statusValue(
                  details.vadEnabled ? details.vadMode || "On" : "Off",
                  details.vadEnabled ? "positive" : "inactive",
                )}
              </dd>
              {#if details.vadEnabled}
                <dt class="text-muted-foreground">Indicator delay</dt>
                <dd class="text-right">{@render durationValue(details.vadActivitySilenceMilliseconds)}</dd>
                <dt class="text-muted-foreground">Silence trimming</dt>
                {#if details.silenceTrimming}
                  <dd class="flex flex-wrap items-center justify-end gap-1.5 text-right">
                    {@render statusValue("On", "positive")}
                    {@render durationValue(details.speechPaddingMilliseconds, "padding")}
                  </dd>
                {:else}
                  <dd class="text-right">{@render statusValue("Off", "inactive")}</dd>
                {/if}
                <dt class="text-muted-foreground">Automatic stop</dt>
                {#if details.autoStopEnabled && !details.autoStopActive}
                  <dd class="text-right">{@render statusValue("Inactive in hold mode", "informational")}</dd>
                {:else if details.autoStopEnabled}
                  <dd class="flex flex-wrap items-center justify-end gap-1.5 text-right">
                    {@render statusValue("On", "positive")}
                    {@render durationValue(details.autoStopSilenceMilliseconds, "pause")}
                    {@render durationValue(details.autoStopMinimumSpeechMilliseconds, "speech")}
                  </dd>
                {:else}
                  <dd class="text-right">{@render statusValue("Off", "inactive")}</dd>
                {/if}
                {#if details.autoStopActive}
                  <dt class="text-muted-foreground">Stop trigger</dt>
                  <dd class="text-right">
                    {@render statusValue(
                      details.autoStopped ? "Silence" : "Manual or limit",
                      details.autoStopped ? "positive" : "informational",
                    )}
                  </dd>
                {/if}
              {/if}
              <dt class="text-muted-foreground">Silence splitting</dt>
              <dd class="text-right">
                {@render statusValue(
                  details.silenceSplitting ? "On" : "Off",
                  details.silenceSplitting ? "positive" : "inactive",
                )}
              </dd>
              <dt class="text-muted-foreground">Checkpoints</dt>
              <dd class="text-right">{details.segmentCount?.toLocaleString() ?? "None"}</dd>
              <dt class="text-muted-foreground">Duration limit</dt>
              <dd class="text-right">
                {@render statusValue(
                  details.durationLimitReached ? "Reached" : "Within limit",
                  details.durationLimitReached ? "warning" : "positive",
                )}
              </dd>
            </dl>

            {#if details.segments && details.segments.length > 0}
              <div class="overflow-hidden rounded-lg border border-hairline">
                <div class="grid grid-cols-[2.5rem_1fr_1fr_1fr] gap-2 bg-layer-fill px-3 py-2 font-mono text-[10px] text-muted-foreground">
                  <span>#</span>
                  <span>Audio</span>
                  <span>Boundary</span>
                  <span class="text-right">Request</span>
                </div>
                {#each details.segments as segment (segment.number)}
                  <div class="grid grid-cols-[2.5rem_1fr_1fr_1fr] gap-2 border-t border-hairline px-3 py-2 text-[11px]">
                    <span>{segment.number}</span>
                    <span>{@render durationValue(segment.audioMilliseconds)}</span>
                    <span>{segment.boundary.replaceAll("_", " ")}</span>
                    <span class="text-right">{@render durationValue(segment.requestMilliseconds)}</span>
                  </div>
                {/each}
              </div>
              {#if details.segmentsTruncated}
                <p class="text-xs text-muted-foreground">
                  Only the first 128 checkpoint summaries are retained.
                </p>
              {/if}
            {/if}
          </section>
        {/if}

        {#if processing.requested}
          <Separator />

          <section class="flex flex-col gap-3" aria-labelledby="processing-details-heading">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <span class="size-1.5 rounded-full bg-primary" aria-hidden="true"></span>
                <h3 id="processing-details-heading" class="text-[13px] font-semibold">Post-processing</h3>
              </div>
              <Badge
                variant={processingVariant(processing.status)}
                class={processingBadgeClass(processing.status)}
              >
                {processingLabel(processing.status)}
              </Badge>
            </div>
            <dl class="grid grid-cols-[minmax(8rem,auto)_minmax(0,1fr)] gap-x-5 text-[12.5px]">
              <dt class="text-muted-foreground">Server</dt>
              <dd class="text-right">{@render endpointValue(processing.server)}</dd>
              <dt class="text-muted-foreground">Model</dt>
              <dd class="text-right">{@render modelValue(processing.model)}</dd>
              <dt class="text-muted-foreground">Profile</dt>
              <dd class="text-right">{processingProfileName([], processing.preset)}</dd>
              <dt class="text-muted-foreground">Elapsed</dt>
              <dd class="text-right">{@render durationValue(processing.elapsedMilliseconds)}</dd>
              {#if processing.timeoutSeconds}
                <dt class="text-muted-foreground">Timeout</dt>
                <dd class="text-right">{@render durationValue(processing.timeoutSeconds * 1000)}</dd>
              {/if}
              <dt class="text-muted-foreground">Characters</dt>
              <dd class="flex flex-wrap items-center justify-end gap-1.5 text-right">
                {@render characterValue(processing.rawCharacterCount, "raw")}
                {#if processing.processedCharacters !== undefined}
                  {@render characterValue(processing.processedCharacters, "processed")}
                {/if}
              </dd>
              {#if processing.styling}
                <dt class="text-muted-foreground">S1-mini controls</dt>
                <dd class="text-right">
                  {processing.styling} · {processing.structure} · {processing.context}
                </dd>
              {/if}
              {#if processing.errorKind}
                <dt class="text-muted-foreground">Fallback reason</dt>
                <dd class="text-right text-warning!">{processing.errorKind}</dd>
              {/if}
            </dl>
            {#if processing.response}
              <HistoryResponseMetadata response={processing.response} stage="processing" />
            {/if}
          </section>
        {/if}
      </div>

      <Dialog.Footer
        class="shrink-0 border-t border-hairline bg-layer-fill px-5 py-3"
        showCloseButton
      />
    </Dialog.Content>
  {/if}
</Dialog.Root>
