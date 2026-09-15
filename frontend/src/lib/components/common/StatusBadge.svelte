<script lang="ts">
  import type { Snippet } from "svelte";
  import { cn } from "$lib/utils";

  let {
    tone = "neutral",
    dot = false,
    children,
    class: className,
  }: {
    tone?: "neutral" | "accent" | "success" | "warning" | "danger";
    dot?: boolean;
    children: Snippet;
    class?: string;
  } = $props();

  const tones = {
    neutral: "border-border bg-secondary text-secondary-foreground",
    accent: "border-accent-edge bg-accent-wash text-accent-text",
    success: "border-success/25 bg-success/10 text-success",
    warning: "border-warning/25 bg-warning/10 text-warning",
    danger: "border-destructive/25 bg-destructive/10 text-destructive",
  };
</script>

<span
  data-slot="status-badge"
  data-tone={tone}
  class={cn(
    "inline-flex max-w-full items-center gap-1.5 rounded-sm border px-1.5 py-0.5 text-[11px] font-medium leading-4",
    tones[tone],
    className,
  )}
>
  {#if dot}<span
      aria-hidden="true"
      class="size-1.5 shrink-0 rounded-full bg-current"
    ></span>{/if}
  <span class="min-w-0">{@render children()}</span>
</span>
