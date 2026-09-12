<script lang="ts">
  import freehandMark from "$lib/assets/freehand-mark.svg";

  let { size = "default" }: { size?: "default" | "large" } = $props();
</script>

<!-- Reuse the canonical vector silhouette in the active interface palette. -->
<span
  role="img"
  aria-label="Freehand"
  class="brand-mark block shrink-0 bg-accent-text {size === 'large' ? 'size-7' : 'size-[19px]'}"
  style:mask-image={`url(${JSON.stringify(freehandMark)})`}
  style:mask-size="contain"
  style:mask-repeat="no-repeat"
  style:mask-position="center"
></span>

<style>
  @media (forced-colors: active) {
    .brand-mark {
      forced-color-adjust: none;
      background: CanvasText;
    }
  }
</style>
