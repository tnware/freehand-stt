<script lang="ts">
  import CircleHelpIcon from "@lucide/svelte/icons/circle-help";
  import { Button } from "$lib/components/ui/button";
  import type { TaskConnectionStatus } from "$lib/utils/connection";

  let {
    connectionState,
    version = "",
    aboutOpen = false,
    onAbout,
  }: {
    connectionState: TaskConnectionStatus;
    version?: string;
    aboutOpen?: boolean;
    onAbout: () => void;
  } = $props();
</script>

<!-- Endpoint reachability is different from dictation state. Keep it quiet,
     persistent, and truthful about whether a probe has actually run. -->
<footer
  class="flex min-h-9 shrink-0 items-center justify-between gap-3 border-t border-hairline bg-layer-fill px-4 text-xs"
>
  <div class="flex min-w-0 items-center gap-2.5" title={connectionState.title}>
    <span class="size-1.5 shrink-0 rounded-full {connectionState.dot}"></span>
    <span class="shrink-0 text-secondary-foreground"
      >{connectionState.scope}: {connectionState.label}</span
    >
    {#if connectionState.detail}
      <span class="truncate text-ink-quiet">{connectionState.detail}</span>
    {/if}
  </div>
  <div class="flex shrink-0 items-center gap-3">
    {#if version}
      <span class="text-ink-quiet">{version}</span>
    {/if}
    <Button
      variant="ghost"
      size="xs"
      class="-mr-1.5 h-6 text-xs text-secondary-foreground hover:text-foreground"
      onclick={onAbout}
      aria-label={aboutOpen ? "Focus About" : "Open About"}
      title={aboutOpen ? "Focus About" : "Open About"}
    >
      <CircleHelpIcon class="size-3.5" />
      About
    </Button>
  </div>
</footer>
