<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    children,
    title,
    icon,
  }: { children: Snippet; title?: string; icon?: Snippet } = $props();
</script>

<!--
  One group surface holds related settings; rows share the same inset and
  dividers, while individual inputs retain their interactive boundaries.
-->
<div
  class="settings-group min-w-0 rounded-sm border border-hairline bg-well px-3 [&>*+*]:border-t [&>*+*]:border-hairline"
>
  {#if title}
    <div class="flex items-center gap-2 py-3">
      {#if icon}<span
          class="content-section-icon flex items-center justify-center [&>svg]:size-4"
          aria-hidden="true">{@render icon()}</span
        >{/if}
      <h4 class="content-section-title">{title}</h4>
    </div>
  {/if}
  {@render children()}
</div>
