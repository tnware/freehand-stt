<script lang="ts">
  import ChevronDownIcon from "@lucide/svelte/icons/chevron-down";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import ProviderIcon from "$lib/components/ProviderIcon.svelte";
  import ConnectionDiagnostics from "$lib/components/settings/ConnectionDiagnostics.svelte";
  import { Button } from "$lib/components/ui/button";
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
  const current = $derived(!details.stale && !details.busy ? details.result : null);
  const attention = $derived(
    current?.checks?.filter((check) => check.status === CheckStatus.CheckAttention) ?? [],
  );
</script>

<div class="space-y-4">
  <div class="space-y-1">
    <h2 class="text-sm font-semibold">{details.task}</h2>
    <p class="text-xs text-muted-foreground">Active connection</p>
  </div>
  {#if details.selected}
    <div class="flex items-start gap-3 border-b border-hairline pb-4">
      <ProviderIcon profile={details.selected.details.compatibilityProfile} />
      <div class="min-w-0 space-y-1">
        <p class="break-words text-sm font-medium">{details.selected.name}</p>
        <p class="break-all text-xs text-muted-foreground">{details.host}</p>
        {#if details.model}<p class="break-all font-mono text-xs text-muted-foreground">
            {details.model}
          </p>{/if}
      </div>
    </div>
  {/if}
  <div class="space-y-2 text-sm" role="status" aria-live="polite">
    {#if details.loading}<p>Loading connection settings…</p>
    {:else if !details.enabled}<p>Text to speech is off.</p>
      <p class="text-xs text-muted-foreground">
        Enable it in Speech settings when you want to generate audio.
      </p>
    {:else if !details.selected}<p>No connection selected.</p>
      <p class="text-xs text-muted-foreground">Choose a saved connection for this task.</p>
    {:else if details.busy}<p class="flex items-center gap-2">
        <LoaderCircleIcon class="size-4 animate-spin motion-reduce:animate-none" />Checking
        connection…
      </p>
    {:else if details.stale}<p class="text-warning">Settings changed since the last check.</p>
      <p class="text-xs text-muted-foreground">
        Check again to see results for the active connection.
      </p>
    {:else if current}
      <p class:text-destructive={!connectionSucceeded(current)} class="font-medium">
        {connectionStatusLabel(current)}
      </p>
      {#if !connectionSucceeded(current)}<p class="text-xs leading-relaxed text-muted-foreground">
          {connectionDescription(current)}
        </p>{/if}
      {#each attention as check (check.kind)}<p class="text-xs leading-relaxed text-warning">
          {check.summary}{check.detail ? ` ${check.detail}` : ""}
        </p>{/each}
      <p class="text-xs text-muted-foreground">
        {current.latencyMilliseconds > 0
          ? `${current.latencyMilliseconds.toLocaleString()} ms · `
          : ""}Metadata check; no audio was sent.
      </p>
    {:else}<p>Not checked yet.</p>
      <p class="text-xs text-muted-foreground">
        Check access without recording or running a model.
      </p>{/if}
  </div>
  <div class="flex flex-wrap gap-2">
    {#if !details.enabled}
      <Button size="sm" disabled={disabled || details.loading} onclick={onSettings}
        >Speech settings</Button
      >
    {:else if details.selected}
      <Button
        size="sm"
        disabled={disabled || details.loading || details.busy || !details.selected}
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
      onclick={onEdit}>{details.selected ? "Edit connection" : "Choose connection"}</Button
    >
  </div>
  {#if current && details.selected && details.enabled}
    <details class="group border-t border-hairline pt-3">
      <summary
        class="flex cursor-pointer list-none items-center justify-between gap-2 text-xs text-muted-foreground focus-visible:outline-2 focus-visible:outline-ring [&::-webkit-details-marker]:hidden"
        >Technical details<ChevronDownIcon class="size-4 group-open:rotate-180" /></summary
      >
      <div class="pt-4"><ConnectionDiagnostics result={current} /></div>
    </details>
  {/if}
</div>
