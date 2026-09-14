<script lang="ts">
  import BrandMark from "$lib/components/shell/BrandMark.svelte";

  let { paneLabel }: { paneLabel: string } = $props();
</script>

<!--
  Identity and nothing else. The mode tabs this row used to carry are the
  activity rail, the shortcut and endpoint health are the status bar, and the
  settings button is the foot of the rail — which takes the header from 72px
  to 36px and gives the height back to the transcript.
-->
<header
  class="flex h-9 shrink-0 items-center gap-2.5 border-b border-hairline bg-well px-3"
>
  <BrandMark />
  <h1 class="font-display text-[13px] font-semibold tracking-tight">
    Freehand
  </h1>
  <span class="h-3.5 w-px shrink-0 bg-border" aria-hidden="true"></span>
  <p class="truncate text-xs text-muted-foreground">{paneLabel}</p>
</header>
