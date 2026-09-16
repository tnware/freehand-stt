<script lang="ts">
  import type { Component } from "svelte";
  import ChevronRightIcon from "@lucide/svelte/icons/chevron-right";
  let {
    title,
    summary = "",
    icon: Icon,
    open,
    controls,
    onclick,
  }: {
    title: string;
    summary?: string;
    icon?: Component;
    open: boolean;
    controls: string;
    onclick: () => void;
  } = $props();
</script>

<button
  type="button"
  class="disclosure-trigger border-b border-hairline"
  aria-expanded={open}
  aria-controls={controls}
  {onclick}
>
  <ChevronRightIcon class="disclosure-chevron" aria-hidden="true" />
  {#if Icon}<span class="disclosure-icon" aria-hidden="true"><Icon /></span
    >{/if}
  <span class="min-w-0 flex-1">
    <span class="block font-medium">{title}</span>
    {#if summary}<span
        class="mt-1 block text-xs font-normal leading-relaxed text-muted-foreground"
        >{summary}</span
      >{/if}
  </span>
</button>
