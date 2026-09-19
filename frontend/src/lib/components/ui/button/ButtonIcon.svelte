<script lang="ts">
  import type { Component } from "svelte";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

  let { icon: Icon, busy = false }: { icon: Component; busy?: boolean } =
    $props();
</script>

<!-- Inherit the owning button's icon size, including compact toolbar actions. -->
{#if busy}
  <LoaderCircleIcon class="motion-safe:animate-spin" aria-hidden="true" />
{:else}
  <Icon aria-hidden="true" />
{/if}
