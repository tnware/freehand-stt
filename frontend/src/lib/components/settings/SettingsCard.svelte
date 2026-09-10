<script lang="ts">
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();
</script>

<!--
  Flat settings groups share the page surface. Row dividers and consistent
  alignment provide hierarchy; inputs keep their own interactive boundaries.
-->
<div class="settings-group [&>*+*]:border-t [&>*+*]:border-hairline">
  {@render children()}
</div>
