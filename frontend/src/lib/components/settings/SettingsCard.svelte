<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    children,
    title,
    icon,
  }: { children: Snippet; title?: string; icon?: Snippet } = $props();
</script>

<!--
  Settings use ruled groups on the editor surface. Controls keep their own
  boundaries, and every group shares the editor's content inset.
-->
<div
  class="settings-group min-w-0 border-b border-hairline [&>*+*]:border-t [&>*+*]:border-hairline"
>
  {#if title}
    <div class="flex min-h-[34px] items-center gap-2 py-1.5">
      {#if icon}<span
          class="content-section-icon flex items-center justify-center [&>svg]:size-4"
          aria-hidden="true">{@render icon()}</span
        >{/if}
      <h4 class="content-section-title">{title}</h4>
    </div>
  {/if}
  {@render children()}
</div>
