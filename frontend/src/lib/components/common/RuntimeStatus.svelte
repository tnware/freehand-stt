<script lang="ts">
  import type { runtimePresentation } from "$lib/utils/managedRuntime";
  import StatusBadge from "./StatusBadge.svelte";
  import StatusIndicator from "./StatusIndicator.svelte";
  let {
    view,
    label = view.label,
    problem = "",
    loading = false,
    badge = false,
  }: {
    view: Pick<
      ReturnType<typeof runtimePresentation>,
      "label" | "tone" | "busy"
    >;
    label?: string;
    problem?: string;
    loading?: boolean;
    badge?: boolean;
  } = $props();
  const busy = $derived(view.busy || loading);
  const tone = $derived(problem ? "danger" : loading ? "accent" : view.tone);
  const colors = {
    neutral: "text-secondary-foreground",
    accent: "text-accent-text",
    success: "text-success",
    warning: "text-warning",
    danger: "text-destructive",
  };
</script>

{#snippet content()}
  <span class="flex min-w-0 items-start gap-1.5">
    <span class="mt-px inline-flex shrink-0"
      ><StatusIndicator {tone} {busy} /></span
    >
    <span class="min-w-0 break-words [overflow-wrap:anywhere]">{label}</span>
  </span>
{/snippet}

<span
  data-slot="runtime-status"
  data-tone={tone}
  data-busy={busy}
  class="inline-flex min-w-0 max-w-full shrink-0 text-xs font-medium leading-4 {badge
    ? ''
    : colors[tone]}"
>
  {#if badge}<StatusBadge {tone}>{@render content()}</StatusBadge>
  {:else}{@render content()}{/if}
</span>
