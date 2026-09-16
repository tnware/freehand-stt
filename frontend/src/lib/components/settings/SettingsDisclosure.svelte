<script lang="ts">
  import type { Snippet } from "svelte";
  import Disclosure from "$lib/components/common/Disclosure.svelte";
  let {
    title,
    description,
    children,
  }: { title: string; description: string; children: Snippet } = $props();
</script>

<Disclosure
  {title}
  {description}
  bodyClass="settings-group [&>*+*]:border-t [&>*+*]:border-hairline"
>
  {@render children()}
</Disclosure>
