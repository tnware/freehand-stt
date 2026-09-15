<script lang="ts">
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();
</script>

<!--
  One group surface holds related settings; rows share the same inset and
  dividers, while individual inputs retain their interactive boundaries.
-->
<div
  class="settings-group border-t border-hairline [&>*+*]:border-t [&>*+*]:border-hairline"
>
  {@render children()}
</div>
