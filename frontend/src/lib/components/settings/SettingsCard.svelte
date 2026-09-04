<script lang="ts">
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();
</script>

<!--
  One card per setting cluster, with a quiet outer stroke and hairline row
  dividers. Controls retain their own familiar Windows input chrome while the
  card supplies the broader information hierarchy.
-->
<div
  class="overflow-hidden rounded-xl border border-card-stroke bg-card shadow-lift [&>*+*]:border-t [&>*+*]:border-hairline"
>
  {@render children()}
</div>
