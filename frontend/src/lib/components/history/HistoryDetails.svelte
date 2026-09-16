<script lang="ts">
  import HistoryOutcomeBadge from "./HistoryOutcomeBadge.svelte";
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import CircleMinusIcon from "@lucide/svelte/icons/circle-minus";
  import CircleXIcon from "@lucide/svelte/icons/circle-x";
  import ClockIcon from "@lucide/svelte/icons/clock";
  import AudioLinesIcon from "@lucide/svelte/icons/audio-lines";
  import FileAudioIcon from "@lucide/svelte/icons/file-audio";
  import MicIcon from "@lucide/svelte/icons/mic";
  import SparklesIcon from "@lucide/svelte/icons/sparkles";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import { Badge } from "$lib/components/ui/badge";
  import { Separator } from "$lib/components/ui/separator";
  import SidebarHeader from "$lib/components/shell/SidebarHeader.svelte";
  import {
    HistoryProcessingStatus,
    HistoryResponseMode,
    HistorySource,
    InsertionMode,
    RecordingMode,
    type HistoryEntry,
  } from "$lib/state";
  import { processingProfileName } from "$lib/utils/processingProfiles";
  import HistoryResponseMetadata from "./HistoryResponseMetadata.svelte";

  let { entry, embedded = false }: { entry: HistoryEntry; embedded?: boolean } =
    $props();
  const uid = $props.id();

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
    source === HistorySource.HistorySourceAudioFile
      ? "Audio file"
      : "Voice dictation";

  const insertionModeLabel = (value?: string): string => {
    if (value === InsertionMode.DirectInput) return "Direct input";
    if (value === InsertionMode.ManualCopy) return "Manual copy";
    return "Not available";
  };

  const processingLabel = (status: HistoryProcessingStatus): string => {
    if (status === HistoryProcessingStatus.HistoryProcessingCompleted)
      return "Completed";
    if (status === HistoryProcessingStatus.HistoryProcessingFailed)
      return "Raw fallback";
    if (status === HistoryProcessingStatus.HistoryProcessingCancelled)
      return "Cancelled";
    if (status === HistoryProcessingStatus.HistoryProcessingPending)
      return "Pending";
    return "Not requested";
  };

  const processingVariant = (
    status: HistoryProcessingStatus,
  ): "default" | "secondary" => {
    if (status === HistoryProcessingStatus.HistoryProcessingFailed)
      return "secondary";
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
    class="inline-flex min-h-5 max-w-full items-center gap-1.5 font-mono text-xs leading-tight font-medium tabular-nums text-foreground"
  >
    <span class="min-w-0 [overflow-wrap:anywhere]"
      >{duration(value)}{suffix ? ` ${suffix}` : ""}</span
    >
  </span>
{/snippet}

{#snippet endpointValue(value?: string)}
  {#if value}
    <span class="mono-chip font-normal">
      {value}
    </span>
  {:else}
    <span class="font-normal text-muted-foreground">Not available</span>
  {/if}
{/snippet}

{#snippet modelValue(value?: string)}
  {#if value}
    <span class="mono-chip">
      {value}
    </span>
  {:else}
    <span class="font-normal text-muted-foreground">Not available</span>
  {/if}
{/snippet}

{#snippet statusValue(label: string, status: FieldStatus)}
  <span
    class={`inline-flex min-h-5 max-w-full items-center gap-1.5 rounded-sm px-2 py-0.5 text-xs leading-tight font-medium [&>svg]:shrink-0 ${fieldStatusClass(status)}`}
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
    <span class="min-w-0 [overflow-wrap:anywhere]">{label}</span>
  </span>
{/snippet}

{#snippet characterValue(value?: number, suffix?: string)}
  <span class="mono-run">
    <span class="min-w-0 [overflow-wrap:anywhere]"
      >{(value ?? 0).toLocaleString()}</span
    >
    {#if suffix}
      <span class="font-normal text-muted-foreground">{suffix}</span>
    {/if}
  </span>
{/snippet}

{#if entry}
  {@const details = entry.details}
  {@const processing = details.processing}
  <div
    class="history-details flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
  >
    {#if embedded}
      <SidebarHeader title="Transcription details" heading />
    {:else}
      <header class="shrink-0 border-b border-hairline bg-layer-fill px-3 py-3">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <h1 class="content-title">Transcription details</h1>
            <p class="content-meta mt-1">
              {sourceLabel(details.source)} · run #{entry.id.toLocaleString()}
            </p>
          </div>
          <HistoryOutcomeBadge outcome={entry.outcome} />
        </div>
      </header>
    {/if}

    <!-- The overflow region needs keyboard focus for scrolling. -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div
      tabindex="0"
      role="region"
      aria-label="Run information"
      class="flex min-h-0 min-w-0 flex-1 flex-col gap-4 overflow-y-auto overscroll-contain p-3"
    >
      <section
        class="content-summary flex min-w-0 flex-col gap-3"
        aria-labelledby={`${uid}-run-details-heading`}
      >
        <div class="flex items-center gap-2">
          <ClockIcon class="content-section-icon" aria-hidden="true" />
          <svelte:element
            this={embedded ? "h3" : "h2"}
            id={`${uid}-run-details-heading`}
            class="content-section-title">Run</svelte:element
          >
        </div>
        <dl class="details-grid">
          {#if embedded}
            <dt class="text-muted-foreground">Source</dt>
            <dd>{sourceLabel(details.source)}</dd>
            <dt class="text-muted-foreground">Run</dt>
            <dd>#{entry.id.toLocaleString()}</dd>
            <dt class="text-muted-foreground">Delivery outcome</dt>
            <dd>
              <HistoryOutcomeBadge outcome={entry.outcome} />
            </dd>
          {/if}
          <dt class="text-muted-foreground">Started</dt>
          <dd class="break-words">{dateTime(details.startedAt)}</dd>
          <dt class="text-muted-foreground">Completed</dt>
          <dd class="break-words">
            {dateTime(details.completedAt)}
          </dd>
          <dt class="text-muted-foreground">Total elapsed</dt>
          <dd>
            {@render durationValue(details.elapsedMilliseconds)}
          </dd>
          <dt class="text-muted-foreground">
            {details.source === HistorySource.HistorySourceVoice &&
            details.silenceTrimming
              ? "Audio submitted"
              : "Audio length"}
          </dt>
          <dd>
            {@render durationValue(details.audioDurationMilliseconds)}
          </dd>
          <dt class="text-muted-foreground">Characters delivered</dt>
          <dd>
            {@render characterValue(
              processing.deliveredCharacters ?? entry.characterCount,
            )}
          </dd>
          <dt class="text-muted-foreground">Delivery mode</dt>
          <dd>
            {insertionModeLabel(details.insertionMode)}
          </dd>
        </dl>
      </section>

      <Separator />

      <section
        class="flex min-w-0 flex-col gap-3"
        aria-labelledby={`${uid}-request-details-heading`}
      >
        <div class="flex items-center gap-2">
          <AudioLinesIcon class="content-section-icon" aria-hidden="true" />
          <svelte:element
            this={embedded ? "h3" : "h2"}
            id={`${uid}-request-details-heading`}
            class="content-section-title">Speech recognition</svelte:element
          >
        </div>
        <dl class="details-grid">
          <dt class="text-muted-foreground">Server</dt>
          <dd>{@render endpointValue(details.server)}</dd>
          <dt class="text-muted-foreground">Route</dt>
          <dd class="break-all">{details.route}</dd>
          <dt class="text-muted-foreground">Authentication</dt>
          <dd>
            {details.authenticationMode === "none" ? "None" : "API key"}
          </dd>
          <dt class="text-muted-foreground">Model</dt>
          <dd>{@render modelValue(details.model)}</dd>
          <dt class="text-muted-foreground">Language</dt>
          <dd>
            {details.language === "auto"
              ? "Automatic detection"
              : details.language || "Server default"}
          </dd>
          <dt class="text-muted-foreground">Response</dt>
          <dd>
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
            <dd>
              {details.streamFallbackReason.replaceAll("_", " ")}
            </dd>
          {/if}
          <dt class="text-muted-foreground">Request time</dt>
          <dd>
            {@render durationValue(details.transcriptionMilliseconds)}
          </dd>
          {#if details.requestTimeoutSeconds}
            <dt class="text-muted-foreground">Request timeout</dt>
            <dd>
              {@render durationValue(details.requestTimeoutSeconds * 1000)}
            </dd>
          {/if}
          {#if details.errorKind}
            <dt class="text-muted-foreground">Terminal error</dt>
            <dd>{details.errorKind}</dd>
          {/if}
        </dl>
        {#if details.transcription}
          <HistoryResponseMetadata
            response={details.transcription}
            stage="transcription"
          />
        {/if}
      </section>

      <Separator />

      {#if details.source === HistorySource.HistorySourceAudioFile}
        <section
          class="flex min-w-0 flex-col gap-3"
          aria-labelledby={`${uid}-source-details-heading`}
        >
          <div class="flex items-center gap-2">
            <FileAudioIcon class="content-section-icon" aria-hidden="true" />
            <svelte:element
              this={embedded ? "h3" : "h2"}
              id={`${uid}-source-details-heading`}
              class="content-section-title">Audio file</svelte:element
            >
          </div>
          <dl class="details-grid">
            <dt class="text-muted-foreground">Filename</dt>
            <dd class="break-all">
              {details.fileName || "Not available"}
            </dd>
            <dt class="text-muted-foreground">File size</dt>
            <dd>{bytes(details.fileSize)}</dd>
            <dt class="text-muted-foreground">Upload time</dt>
            <dd>
              {@render durationValue(details.uploadMilliseconds)}
            </dd>
          </dl>
        </section>
      {:else}
        <section
          class="flex min-w-0 flex-col gap-3"
          aria-labelledby={`${uid}-source-details-heading`}
        >
          <div class="flex items-center gap-2">
            <MicIcon class="content-section-icon" aria-hidden="true" />
            <svelte:element
              this={embedded ? "h3" : "h2"}
              id={`${uid}-source-details-heading`}
              class="content-section-title">Voice capture</svelte:element
            >
          </div>
          <dl class="details-grid">
            <dt class="text-muted-foreground">Microphone</dt>
            <dd class="break-words">
              {details.microphone || "Not available"}
            </dd>
            <dt class="text-muted-foreground">Recording control</dt>
            <dd>
              {details.recordingMode === RecordingMode.RecordingHold
                ? "Hold to talk"
                : "Toggle"}
            </dd>
            <dt class="text-muted-foreground">Recording length</dt>
            <dd>
              {@render durationValue(details.captureDurationMilliseconds)}
            </dd>
            <dt class="text-muted-foreground">VAD</dt>
            <dd>
              {@render statusValue(
                details.vadEnabled ? details.vadMode || "On" : "Off",
                details.vadEnabled ? "positive" : "inactive",
              )}
            </dd>
            {#if details.vadEnabled}
              <dt class="text-muted-foreground">Indicator delay</dt>
              <dd>
                {@render durationValue(details.vadActivitySilenceMilliseconds)}
              </dd>
              <dt class="text-muted-foreground">Silence trimming</dt>
              {#if details.silenceTrimming}
                <dd class="flex flex-wrap items-center gap-1.5">
                  {@render statusValue("On", "positive")}
                  {@render durationValue(
                    details.speechPaddingMilliseconds,
                    "padding",
                  )}
                </dd>
              {:else}
                <dd>
                  {@render statusValue("Off", "inactive")}
                </dd>
              {/if}
              <dt class="text-muted-foreground">Automatic stop</dt>
              {#if details.autoStopEnabled && !details.autoStopActive}
                <dd>
                  {@render statusValue(
                    "Inactive in hold mode",
                    "informational",
                  )}
                </dd>
              {:else if details.autoStopEnabled}
                <dd class="flex flex-wrap items-center gap-1.5">
                  {@render statusValue("On", "positive")}
                  {@render durationValue(
                    details.autoStopSilenceMilliseconds,
                    "pause",
                  )}
                  {@render durationValue(
                    details.autoStopMinimumSpeechMilliseconds,
                    "speech",
                  )}
                </dd>
              {:else}
                <dd>
                  {@render statusValue("Off", "inactive")}
                </dd>
              {/if}
              {#if details.autoStopActive}
                <dt class="text-muted-foreground">Stop trigger</dt>
                <dd>
                  {@render statusValue(
                    details.autoStopped ? "Silence" : "Manual or limit",
                    details.autoStopped ? "positive" : "informational",
                  )}
                </dd>
              {/if}
            {/if}
            <dt class="text-muted-foreground">Silence splitting</dt>
            <dd>
              {@render statusValue(
                details.silenceSplitting ? "On" : "Off",
                details.silenceSplitting ? "positive" : "inactive",
              )}
            </dd>
            <dt class="text-muted-foreground">Checkpoints</dt>
            <dd>
              {details.segmentCount?.toLocaleString() ?? "None"}
            </dd>
            <dt class="text-muted-foreground">Duration limit</dt>
            <dd>
              {@render statusValue(
                details.durationLimitReached ? "Reached" : "Within limit",
                details.durationLimitReached ? "warning" : "positive",
              )}
            </dd>
          </dl>

          {#if details.segments && details.segments.length > 0}
            <div class="min-w-0 rounded-sm border border-hairline">
              <div
                class="content-kicker checkpoint-heading grid grid-cols-[2.5rem_repeat(3,minmax(0,1fr))] gap-2 bg-well px-3 py-2"
                aria-hidden="true"
              >
                <span>#</span>
                <span>Audio</span>
                <span>Boundary</span>
                <span class="text-right">Request</span>
              </div>
              {#each details.segments as segment (segment.number)}
                <dl
                  class="checkpoint-row grid grid-cols-[2.5rem_repeat(3,minmax(0,1fr))] gap-2 border-t border-hairline px-3 py-2 text-xs tabular-nums"
                >
                  <div class="min-w-0">
                    <dt class="checkpoint-label text-muted-foreground">
                      Checkpoint
                    </dt>
                    <dd>{segment.number}</dd>
                  </div>
                  <div class="min-w-0">
                    <dt class="checkpoint-label text-muted-foreground">
                      Audio
                    </dt>
                    <dd>{@render durationValue(segment.audioMilliseconds)}</dd>
                  </div>
                  <div class="min-w-0">
                    <dt class="checkpoint-label text-muted-foreground">
                      Boundary
                    </dt>
                    <dd class="[overflow-wrap:anywhere]">
                      {segment.boundary.replaceAll("_", " ")}
                    </dd>
                  </div>
                  <div class="checkpoint-request min-w-0 text-right">
                    <dt class="checkpoint-label text-muted-foreground">
                      Request
                    </dt>
                    <dd>
                      {@render durationValue(segment.requestMilliseconds)}
                    </dd>
                  </div>
                </dl>
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

        <section
          class="flex min-w-0 flex-col gap-3"
          aria-labelledby={`${uid}-processing-details-heading`}
        >
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <SparklesIcon class="content-section-icon" aria-hidden="true" />
              <svelte:element
                this={embedded ? "h3" : "h2"}
                id={`${uid}-processing-details-heading`}
                class="content-section-title"
              >
                Post-processing
              </svelte:element>
            </div>
            <Badge
              variant={processingVariant(processing.status)}
              class={processingBadgeClass(processing.status)}
            >
              {processingLabel(processing.status)}
            </Badge>
          </div>
          <dl class="details-grid">
            <dt class="text-muted-foreground">Server</dt>
            <dd>
              {@render endpointValue(processing.server)}
            </dd>
            <dt class="text-muted-foreground">Model</dt>
            <dd>{@render modelValue(processing.model)}</dd>
            <dt class="text-muted-foreground">Profile</dt>
            <dd>
              {processingProfileName([], processing.preset)}
            </dd>
            <dt class="text-muted-foreground">Elapsed</dt>
            <dd>
              {@render durationValue(processing.elapsedMilliseconds)}
            </dd>
            {#if processing.timeoutSeconds}
              <dt class="text-muted-foreground">Timeout</dt>
              <dd>
                {@render durationValue(processing.timeoutSeconds * 1000)}
              </dd>
            {/if}
            <dt class="text-muted-foreground">Characters</dt>
            <dd class="flex flex-wrap items-center gap-1.5">
              {@render characterValue(processing.rawCharacterCount, "raw")}
              {#if processing.processedCharacters !== undefined}
                {@render characterValue(
                  processing.processedCharacters,
                  "processed",
                )}
              {/if}
            </dd>
            {#if processing.styling}
              <dt class="text-muted-foreground">S1-mini controls</dt>
              <dd>
                {processing.styling} · {processing.structure} · {processing.context}
              </dd>
            {/if}
            {#if processing.errorKind}
              <dt class="text-muted-foreground">Fallback reason</dt>
              <dd class="text-warning!">{processing.errorKind}</dd>
            {/if}
          </dl>
          {#if processing.response}
            <HistoryResponseMetadata
              response={processing.response}
              stage="processing"
            />
          {/if}
        </section>
      {/if}
    </div>
  </div>
{/if}

<style>
  .history-details {
    container: history-details / inline-size;
  }

  .checkpoint-label {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }

  @container history-details (max-width: 420px) {
    .checkpoint-heading {
      display: none;
    }

    .checkpoint-row {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 0.5rem 0.75rem;
    }

    .checkpoint-label {
      position: static;
      width: auto;
      height: auto;
      overflow: visible;
      clip-path: none;
      white-space: normal;
    }

    .checkpoint-request {
      text-align: left;
    }
  }
</style>
