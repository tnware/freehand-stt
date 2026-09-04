<script lang="ts">
  import freehandMark from "$lib/assets/freehand-mark.svg";

  let { size = "default" }: { size?: "default" | "large" } = $props();
</script>

<!-- The mark stands on the ground unframed: a bordered tile would read as a
     third container in a header that is already one. -->
<img
  src={freehandMark}
  alt=""
  aria-label="Freehand"
  class="shrink-0 {size === 'large' ? 'size-7' : 'size-[19px]'}"
/>
