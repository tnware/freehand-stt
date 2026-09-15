<script lang="ts">
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import FileTextIcon from "@lucide/svelte/icons/file-text";
  import GaugeIcon from "@lucide/svelte/icons/gauge";
  import HashIcon from "@lucide/svelte/icons/hash";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  import { Badge } from "$lib/components/ui/badge";
  import { Separator } from "$lib/components/ui/separator";
  import type {
    HistoryPerformanceDetails,
    HistoryResponseDetails,
    HistoryUsageDetails,
  } from "$lib/state";

  let {
    response,
    stage,
  }: {
    response: HistoryResponseDetails;
    stage: "transcription" | "processing";
  } = $props();

  const usage = $derived(response.usage);
  const performance = $derived(response.performance);
  const requests = $derived(Math.max(1, response.requestCount ?? 1));

  const duration = (milliseconds: number): string => {
    if (milliseconds < 1000) return `${milliseconds.toLocaleString()} ms`;
    const seconds = milliseconds / 1000;
    if (seconds < 60) return `${seconds.toFixed(seconds < 10 ? 1 : 0)} s`;
    const minutes = Math.floor(seconds / 60);
    const remainder = Math.round(seconds % 60);
    return `${minutes}m ${remainder}s`;
  };

  const responseDateTime = (value: number): string =>
    new Date(value * 1000).toLocaleString([], {
      dateStyle: "medium",
      timeStyle: "medium",
    });

  const decimal = (value: number, maximumFractionDigits = 2): string =>
    value.toLocaleString(undefined, { maximumFractionDigits });

  const reportedCost = (value: number): string =>
    value.toLocaleString(undefined, {
      minimumSignificantDigits: 1,
      maximumSignificantDigits: 8,
    });

  const coverageClass = (complete: boolean): string =>
    complete ? "bg-success/10 text-success" : "bg-warning/10 text-warning";

  const hasUsage = (usage: HistoryUsageDetails): boolean =>
    Boolean(usage.type) ||
    [
      usage.inputTokens,
      usage.outputTokens,
      usage.totalTokens,
      usage.audioInputTokens,
      usage.textInputTokens,
      usage.cachedInputTokens,
      usage.cacheWriteTokens,
      usage.reasoningOutputTokens,
      usage.audioSeconds,
      usage.reportedCost,
      usage.upstreamCost,
    ].some((value) => value != null);

  const hasPerformance = (performance: HistoryPerformanceDetails): boolean =>
    [
      performance.promptTokens,
      performance.promptMilliseconds,
      performance.promptMillisecondsPerToken,
      performance.promptTokensPerSecond,
      performance.generatedTokens,
      performance.generationMilliseconds,
      performance.generationMillisecondsPerToken,
      performance.generationTokensPerSecond,
      performance.cachedPromptTokens,
    ].some((value) => value != null);

  const hasIdentity = (details: HistoryResponseDetails): boolean =>
    Boolean(
      details.requestId ||
      details.responseId ||
      details.effectiveModel ||
      details.provider ||
      details.finishReason ||
      details.serviceTier ||
      details.systemFingerprint ||
      details.detectedLanguages?.length,
    ) || details.createdAtUnix != null;

  const usageVisible = $derived(
    response.serverAudioSeconds != null || hasUsage(usage),
  );
  const identityVisible = $derived(hasIdentity(response));
  const performanceVisible = $derived(hasPerformance(performance));
</script>

{#snippet durationValue(value: number)}
  <span
    class="inline-flex min-h-5 max-w-full flex-wrap items-center gap-1.5 font-mono text-xs leading-tight font-medium tabular-nums text-foreground"
  >
    <span class="min-w-0 [overflow-wrap:anywhere]">{duration(value)}</span>
  </span>
{/snippet}

{#snippet modelValue(value: string)}
  <span
    class="inline-block min-h-5 max-w-full rounded-sm bg-well px-1.5 py-0.5 font-mono text-xs leading-tight font-medium text-foreground break-all"
  >
    {value}
  </span>
{/snippet}

{#snippet tokenValue(value: number, suffix?: string)}
  <span
    class="inline-flex min-h-5 max-w-full flex-wrap items-center gap-1.5 font-mono text-xs leading-tight font-semibold tabular-nums text-foreground"
  >
    <span class="min-w-0 [overflow-wrap:anywhere]"
      >{value.toLocaleString()}</span
    >
    {#if suffix}
      <span class="font-normal text-muted-foreground">{suffix}</span>
    {/if}
  </span>
{/snippet}

{#snippet metricValue(value: number, suffix: string, digits = 2)}
  <span
    class="inline-flex min-h-5 max-w-full flex-wrap items-center gap-1.5 font-mono text-xs leading-tight font-medium tabular-nums text-foreground"
  >
    <span class="min-w-0 [overflow-wrap:anywhere]"
      >{decimal(value, digits)}</span
    >
    <span class="font-normal text-muted-foreground">{suffix}</span>
  </span>
{/snippet}

{#snippet costValue(value: number, suffix: string)}
  <span
    class="inline-flex min-h-5 max-w-full flex-wrap items-center gap-1.5 font-mono text-xs leading-tight font-medium tabular-nums text-foreground"
  >
    <span class="min-w-0 [overflow-wrap:anywhere]">{reportedCost(value)}</span>
    <span class="font-normal text-muted-foreground">{suffix}</span>
  </span>
{/snippet}

{#snippet identifierValue(value: string)}
  <span
    class="inline-block min-h-5 max-w-full rounded-sm bg-well px-1.5 py-0.5 font-mono text-xs leading-tight font-normal text-foreground break-all"
  >
    {value}
  </span>
{/snippet}

{#snippet coverageValue(reported = 0, requests = 1)}
  {@const complete = reported >= requests}
  <Badge
    variant="secondary"
    class={`h-auto min-h-5 max-w-full whitespace-normal [overflow-wrap:anywhere] ${coverageClass(complete)}`}
  >
    {#if complete}
      <CircleCheckIcon data-icon="inline-start" aria-hidden="true" />
    {:else}
      <TriangleAlertIcon data-icon="inline-start" aria-hidden="true" />
    {/if}
    <span class="min-w-0"
      >{reported.toLocaleString()} of {requests.toLocaleString()} requests</span
    >
  </Badge>
{/snippet}

<div class="response-metadata flex min-w-0 flex-col gap-3">
  <Separator />

  {#if usageVisible}
    <section class="flex flex-col gap-2" aria-label="Usage">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h4 class="content-kicker flex items-center gap-2">
          <HashIcon class="content-section-icon" aria-hidden="true" />Usage
        </h4>
        {#if response.usageReportCount || requests > 1}
          {@render coverageValue(response.usageReportCount ?? 0, requests)}
        {/if}
      </div>
      <dl
        class="response-grid grid grid-cols-[minmax(7rem,auto)_minmax(0,1fr)] gap-x-4 text-xs"
      >
        {#if usage.type}
          <dt class="text-muted-foreground">Basis</dt>
          <dd class="text-right">{usage.type.replaceAll("_", " ")}</dd>
        {/if}
        {#if usage.inputTokens != null}
          <dt class="text-muted-foreground">
            {stage === "processing" ? "Prompt tokens" : "Input tokens"}
          </dt>
          <dd class="text-right">{@render tokenValue(usage.inputTokens)}</dd>
        {/if}
        {#if usage.outputTokens != null}
          <dt class="text-muted-foreground">
            {stage === "processing" ? "Completion tokens" : "Output tokens"}
          </dt>
          <dd class="text-right">{@render tokenValue(usage.outputTokens)}</dd>
        {/if}
        {#if usage.totalTokens != null}
          <dt class="text-muted-foreground">Total tokens</dt>
          <dd class="text-right">{@render tokenValue(usage.totalTokens)}</dd>
        {/if}
        {#if usage.audioInputTokens != null}
          <dt class="text-muted-foreground">Audio input tokens</dt>
          <dd class="text-right">
            {@render tokenValue(usage.audioInputTokens)}
          </dd>
        {/if}
        {#if usage.textInputTokens != null}
          <dt class="text-muted-foreground">Text input tokens</dt>
          <dd class="text-right">
            {@render tokenValue(usage.textInputTokens)}
          </dd>
        {/if}
        {#if usage.cachedInputTokens != null}
          <dt class="text-muted-foreground">Cached input tokens</dt>
          <dd class="text-right">
            {@render tokenValue(usage.cachedInputTokens)}
          </dd>
        {/if}
        {#if usage.cacheWriteTokens != null}
          <dt class="text-muted-foreground">Cache write tokens</dt>
          <dd class="text-right">
            {@render tokenValue(usage.cacheWriteTokens)}
          </dd>
        {/if}
        {#if usage.reasoningOutputTokens != null}
          <dt class="text-muted-foreground">Reasoning tokens</dt>
          <dd class="text-right">
            {@render tokenValue(usage.reasoningOutputTokens)}
          </dd>
        {/if}
        {#if response.serverAudioSeconds != null}
          <dt class="text-muted-foreground">Audio duration</dt>
          <dd class="text-right">
            {@render durationValue(response.serverAudioSeconds * 1000)}
          </dd>
        {/if}
        {#if usage.audioSeconds != null}
          <dt class="text-muted-foreground">Billable audio</dt>
          <dd class="text-right">
            {@render durationValue(usage.audioSeconds * 1000)}
          </dd>
        {/if}
        {#if usage.reportedCost != null}
          <dt class="text-muted-foreground">Provider-reported cost</dt>
          <dd class="text-right">
            {@render costValue(usage.reportedCost, "reported units")}
          </dd>
        {/if}
        {#if usage.upstreamCost != null}
          <dt class="text-muted-foreground">Upstream cost</dt>
          <dd class="text-right">
            {@render costValue(usage.upstreamCost, "reported units")}
          </dd>
        {/if}
        {#if response.costReportCount && requests > 1}
          <dt class="text-muted-foreground">Cost coverage</dt>
          <dd class="text-right">
            {@render coverageValue(response.costReportCount, requests)}
          </dd>
        {/if}
      </dl>
    </section>
  {/if}

  {#if identityVisible}
    {#if usageVisible}
      <Separator />
    {/if}
    <section class="flex flex-col gap-2" aria-label="Request details">
      <h4 class="content-kicker flex items-center gap-2">
        <FileTextIcon class="content-section-icon" aria-hidden="true" />Request
        details
      </h4>
      <dl
        class="response-grid grid grid-cols-[minmax(7rem,auto)_minmax(0,1fr)] gap-x-4 text-xs"
      >
        {#if requests > 1}
          <dt class="text-muted-foreground">Requests</dt>
          <dd class="text-right">{requests.toLocaleString()}</dd>
        {/if}
        {#if response.effectiveModel}
          <dt class="text-muted-foreground">Effective model</dt>
          <dd class="text-right">
            {@render modelValue(response.effectiveModel)}
          </dd>
        {/if}
        {#if response.provider}
          <dt class="text-muted-foreground">Provider</dt>
          <dd class="text-right break-words">{response.provider}</dd>
        {/if}
        {#if response.finishReason}
          <dt class="text-muted-foreground">Finish reason</dt>
          <dd class="text-right">
            {response.finishReason.replaceAll("_", " ")}
          </dd>
        {/if}
        {#if response.serviceTier}
          <dt class="text-muted-foreground">Service tier</dt>
          <dd class="text-right">{response.serviceTier}</dd>
        {/if}
        {#if response.createdAtUnix != null}
          <dt class="text-muted-foreground">Created</dt>
          <dd class="text-right break-words">
            {responseDateTime(response.createdAtUnix)}
          </dd>
        {/if}
        {#if response.detectedLanguages?.length}
          <dt class="text-muted-foreground">Detected languages</dt>
          <dd class="text-right">{response.detectedLanguages.join(", ")}</dd>
        {/if}
        {#if response.requestId}
          <dt class="text-muted-foreground">Request ID</dt>
          <dd class="text-right">
            {@render identifierValue(response.requestId)}
          </dd>
        {/if}
        {#if response.responseId}
          <dt class="text-muted-foreground">Response ID</dt>
          <dd class="text-right">
            {@render identifierValue(response.responseId)}
          </dd>
        {/if}
        {#if response.systemFingerprint}
          <dt class="text-muted-foreground">System fingerprint</dt>
          <dd class="text-right">
            {@render identifierValue(response.systemFingerprint)}
          </dd>
        {/if}
      </dl>
    </section>
  {/if}

  {#if performanceVisible}
    <Separator />
    <section class="flex flex-col gap-2" aria-label="Runtime performance">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h4 class="content-kicker flex items-center gap-2">
          <GaugeIcon class="content-section-icon" aria-hidden="true" />
          Runtime performance
        </h4>
        {#if response.performanceReportCount || requests > 1}
          {@render coverageValue(
            response.performanceReportCount ?? 0,
            requests,
          )}
        {/if}
      </div>
      <dl
        class="response-grid grid grid-cols-[minmax(7rem,auto)_minmax(0,1fr)] gap-x-4 text-xs"
      >
        {#if performance.promptTokens != null}
          <dt class="text-muted-foreground">Prompt evaluated</dt>
          <dd class="text-right">
            {@render tokenValue(performance.promptTokens, "tokens")}
          </dd>
        {/if}
        {#if performance.promptMilliseconds != null}
          <dt class="text-muted-foreground">Prompt time</dt>
          <dd class="text-right">
            {@render durationValue(performance.promptMilliseconds)}
          </dd>
        {/if}
        {#if performance.promptTokensPerSecond != null}
          <dt class="text-muted-foreground">Prompt speed</dt>
          <dd class="text-right">
            {@render metricValue(performance.promptTokensPerSecond, "tok/s")}
          </dd>
        {/if}
        {#if performance.promptMillisecondsPerToken != null}
          <dt class="text-muted-foreground">Prompt latency</dt>
          <dd class="text-right">
            {@render metricValue(
              performance.promptMillisecondsPerToken,
              "ms/token",
            )}
          </dd>
        {/if}
        {#if performance.generatedTokens != null}
          <dt class="text-muted-foreground">Generated</dt>
          <dd class="text-right">
            {@render tokenValue(performance.generatedTokens, "tokens")}
          </dd>
        {/if}
        {#if performance.generationMilliseconds != null}
          <dt class="text-muted-foreground">Generation time</dt>
          <dd class="text-right">
            {@render durationValue(performance.generationMilliseconds)}
          </dd>
        {/if}
        {#if performance.generationTokensPerSecond != null}
          <dt class="text-muted-foreground">Generation speed</dt>
          <dd class="text-right">
            {@render metricValue(
              performance.generationTokensPerSecond,
              "tok/s",
            )}
          </dd>
        {/if}
        {#if performance.generationMillisecondsPerToken != null}
          <dt class="text-muted-foreground">Generation latency</dt>
          <dd class="text-right">
            {@render metricValue(
              performance.generationMillisecondsPerToken,
              "ms/token",
            )}
          </dd>
        {/if}
        {#if performance.cachedPromptTokens != null}
          <dt class="text-muted-foreground">Cached prompt</dt>
          <dd class="text-right">
            {@render tokenValue(performance.cachedPromptTokens, "tokens")}
          </dd>
        {/if}
      </dl>
    </section>
  {/if}
</div>

<style>
  .response-metadata {
    container: response-metadata / inline-size;
  }

  .response-grid > dd {
    min-width: 0;
    font-size: 13px;
    font-weight: 500;
    color: var(--foreground);
    overflow-wrap: anywhere;
  }

  @container response-metadata (max-width: 380px) {
    .response-grid {
      grid-template-columns: minmax(0, 1fr);
      row-gap: 0.125rem;
    }

    .response-grid > dd {
      margin-bottom: 0.5rem;
      text-align: left;
    }
  }
</style>
