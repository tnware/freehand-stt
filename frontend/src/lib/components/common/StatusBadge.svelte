<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "$lib/utils";

  let {
    tone = "neutral",
    dot = false,
    pulse = false,
    children,
    class: className,
  }: {
    tone?:
      | "neutral"
      | "accent"
      | "success"
      | "warning"
      | "danger"
      | "recording";
    dot?: boolean;
    pulse?: boolean;
    children: Snippet;
    class?: string;
  } = $props();

  const tones = {
    recording:
      "border-record-edge bg-record-wash font-semibold text-record-text",
    neutral: "border-border bg-control-fill text-foreground",
    accent:
      "border-accent-edge bg-accent-wash-strong font-semibold text-accent-text",
    success: "border-success/40 bg-success/12 font-semibold text-success",
    warning: "border-warning/40 bg-warning/12 font-semibold text-warning",
    danger:
      "border-destructive/40 bg-destructive/12 font-semibold text-destructive",
  };
</script>

<span
  data-slot="status-badge"
  data-tone={tone}
  class={cn(
    "inline-flex max-w-full items-center gap-1.5 rounded-sm border px-1.5 py-0.5 text-xs font-medium leading-4",
    tones[tone],
    className,
  )}
>
  {#if dot}<span
      aria-hidden="true"
      class="size-1.5 shrink-0 rounded-full bg-current"
      class:motion-safe:animate-pulse={pulse}
    ></span>{/if}
  <span class="min-w-0">{@render children()}</span>
</span>
