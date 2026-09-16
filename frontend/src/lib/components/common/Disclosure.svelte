<script lang="ts">
  import type { Component, Snippet } from "svelte";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  import { cn } from "$lib/utils";

  let {
    title,
    description = "",
    icon: Icon,
    open = $bindable(false),
    compact = false,
    class: className,
    bodyClass,
    label,
    busy = false,
    tone = "default",
    children,
  }: {
    title: string;
    description?: string;
    icon?: Component;
    open?: boolean;
    compact?: boolean;
    class?: string;
    bodyClass?: string;
    label?: string;
    busy?: boolean;
    tone?: "default" | "warning";
    children: Snippet;
  } = $props();
</script>

<details
  class={cn("min-w-0 border-b border-hairline", className)}
  bind:open
  aria-label={label}
  aria-busy={busy || undefined}
>
  <summary
    class="disclosure-trigger"
    data-density={compact ? "compact" : undefined}
  >
    <ChevronRightIcon class="disclosure-chevron" aria-hidden="true" />
    {#if Icon}<span class="disclosure-icon" aria-hidden="true"><Icon /></span
      >{/if}
    <span class="min-w-0 flex-1">
      <span class="block font-medium" class:text-warning={tone === "warning"}
        >{title}</span
      >
      {#if description}<span
          class="mt-1 block text-xs font-normal leading-relaxed text-muted-foreground"
          >{description}</span
        >{/if}
    </span>
  </summary>
  <div class={cn("disclosure-body", bodyClass)}>{@render children()}</div>
</details>
