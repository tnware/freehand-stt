<script lang="ts">
  import type { Snippet } from "svelte";
  import Disclosure from "./Disclosure.svelte";
  let { title, children }: { title: string; children: Snippet } = $props();
</script>

<Disclosure {title} compact class="border-b-0">
  {@render children()}
</Disclosure>
