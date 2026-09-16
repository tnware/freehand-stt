<script lang="ts">
  import type { Component, Snippet } from "svelte";

  let {
    icon: Icon,
    title,
    description,
    variant = "pane",
    busy = false,
    actions,
  }: {
    icon: Component;
    title: string;
    description: string;
    variant?: "pane" | "compact";
    busy?: boolean;
    actions?: Snippet;
  } = $props();
</script>

<div
  data-slot="empty-state"
  data-variant={variant}
  class="flex min-w-0 flex-1 flex-col text-center {variant === 'pane'
    ? 'px-5 py-8'
    : 'px-3 py-4'}"
>
  <!-- Auto margins center when space permits without hiding the top in short panes. -->
  <div
    class="my-auto flex w-full flex-col items-center {variant === 'pane'
      ? 'gap-3'
      : 'gap-2'}"
  >
    <span
      class="grid shrink-0 place-items-center rounded-lg border border-hairline bg-well text-accent-text {variant ===
      'pane'
        ? 'size-12 shadow-lift'
        : 'size-8'}"
      aria-hidden="true"
    >
      <Icon
        class="{variant === 'pane' ? 'size-6' : 'size-4'} {busy
          ? 'motion-safe:animate-spin'
          : ''}"
      />
    </span>
    <p
      class="max-w-lg [overflow-wrap:anywhere] {variant === 'pane'
        ? 'content-title'
        : 'content-section-title'}"
    >
      {title}
    </p>
    <p
      class="content-meta [overflow-wrap:anywhere] {variant === 'pane'
        ? 'max-w-lg'
        : 'max-w-sm'}"
    >
      {description}
    </p>
    {#if actions}
      <div
        class="mt-1 flex max-w-full flex-wrap items-center justify-center gap-2"
      >
        {@render actions()}
      </div>
    {/if}
  </div>
</div>
