<script lang="ts">
  import Disclosure from "$lib/components/common/Disclosure.svelte";
  import RuntimeStatus from "$lib/components/common/RuntimeStatus.svelte";
  import StatusIndicator from "$lib/components/common/StatusIndicator.svelte";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  import { Button } from "$lib/components/ui/button";
  import StatusBadge from "$lib/components/common/StatusBadge.svelte";
  import { CheckStatus } from "$bindings/connection";
  import {
    connectionDescription,
    connectionStatusLabel,
    connectionSucceeded,
    type TaskConnectionDetails,
  } from "$lib/utils/connection";
  let {
    details,
    disabled = false,
    onCheck,
    onEdit,
    onSettings,
  }: {
    details: TaskConnectionDetails;
    disabled?: boolean;
    onCheck: () => void;
    onEdit: () => void;
    onSettings: () => void;
  } = $props();
  const current = $derived(
    !details.stale && !details.busy && details.runtime?.ready !== false
      ? details.result
      : null,
  );
  const attention = $derived(
    current?.checks?.filter(
      (check) => check.status === CheckStatus.CheckAttention,
    ) ?? [],
  );
</script>

<div class="space-y-4">
  <div class="space-y-1">
    <h2 class="content-title">{details.task}</h2>
    <p class="content-meta">Active connection</p>
  </div>
  {#if details.selected}
    <div class="content-summary flex items-start gap-3">
      <ProviderIcon profile={details.selected.details.compatibilityProfile} />
      <div class="min-w-0 space-y-1">
        <p class="content-value break-words">{details.selected.name}</p>
        <p class="break-all text-xs text-muted-foreground">{details.host}</p>
        {#if details.model}<p
            class="break-all font-mono text-xs text-secondary-foreground"
          >
            {details.model}
          </p>{/if}
      </div>
    </div>
  {/if}
  <div class="space-y-2 text-[13px]" role="status" aria-live="polite">
    {#if details.loading}<p>Loading connection settings…</p>
    {:else if !details.enabled}<StatusBadge>Text to speech is off.</StatusBadge>
      <p class="text-xs text-muted-foreground">
        Enable it in Speech settings when you want to generate audio.
      </p>
    {:else if !details.selected}<StatusBadge tone="warning"
        >No connection selected.</StatusBadge
      >
      <p class="text-xs text-muted-foreground">
        Choose a saved connection for this task.
      </p>
    {:else if details.runtime && !details.runtime.ready}
      <RuntimeStatus
        view={details.runtime.presentation}
        label={details.runtime.label}
        badge
      />
      <p class="text-xs text-muted-foreground">{details.runtime.detail}</p>
    {:else if details.busy}<p
        class="flex items-center gap-1.5 text-accent-text"
      >
        <StatusIndicator tone="accent" busy />Checking connection…
      </p>
    {:else if details.stale}<StatusBadge tone="warning"
        >Settings changed since the last check.</StatusBadge
      >
      <p class="text-xs text-muted-foreground">
        Check again to see results for the active connection.
      </p>
    {:else if current}
      <StatusBadge
        tone={connectionSucceeded(current)
          ? attention.length
            ? "warning"
            : "success"
          : "danger"}
        dot
      >
        {connectionStatusLabel(current)}
      </StatusBadge>
      {#if !connectionSucceeded(current)}<p
          class="text-xs leading-relaxed text-muted-foreground"
        >
          {connectionDescription(current, details.platform)}
        </p>{/if}
      {#each attention as check (check.kind)}<p
          class="text-xs leading-relaxed text-warning"
        >
          {check.summary}{check.detail ? ` ${check.detail}` : ""}
        </p>{/each}
      <p class="text-xs text-muted-foreground">
        {current.latencyMilliseconds > 0
          ? `${current.latencyMilliseconds.toLocaleString()} ms · `
          : ""}Metadata check; no audio was sent.
      </p>
    {:else}<StatusBadge>Not checked yet.</StatusBadge>
      <p class="text-xs text-muted-foreground">
        Check access without recording or running a model.
      </p>{/if}
  </div>
  <div class="flex flex-wrap gap-2">
    {#if !details.enabled}
      <Button
        size="sm"
        disabled={disabled || details.loading}
        onclick={onSettings}>Speech settings</Button
      >
    {:else if details.selected}
      <Button
        variant="soft"
        size="sm"
        disabled={disabled ||
          details.loading ||
          details.busy ||
          details.runtime?.ready === false ||
          !details.selected}
        onclick={onCheck}
      >
        {details.busy
          ? "Checking…"
          : details.result || details.stale
            ? "Check again"
            : "Check connection"}
      </Button>
    {/if}
    <Button
      variant={!details.selected && details.enabled ? "default" : "outline"}
      size="sm"
      disabled={disabled || details.loading || details.busy}
      onclick={onEdit}
      >{details.selected ? "Edit connection" : "Choose connection"}</Button
    >
  </div>
  {#if current && details.selected && details.enabled}
    <Disclosure
      title="Technical details"
      compact
      class="border-t border-hairline"
    >
      <div>
        <ConnectionDiagnostics platform={details.platform} result={current} />
      </div>
    </Disclosure>
  {/if}
</div>
