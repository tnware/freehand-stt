<script lang="ts">
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import CircleCheckIcon from "@lucide/svelte/icons/circle-check";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import TriangleAlertIcon from "@lucide/svelte/icons/triangle-alert";
  let {
    tone = "neutral",
    busy = false,
  }: {
    tone?: "neutral" | "accent" | "success" | "warning" | "danger";
    busy?: boolean;
  } = $props();
</script>

<span
  class="inline-flex size-3.5 shrink-0 items-center justify-center"
  aria-hidden="true"
  data-slot="status-indicator"
  data-busy={busy}
>
  {#if busy}
    <LoaderCircleIcon class="size-3.5 motion-safe:animate-spin" />
  {:else if tone === "danger"}
    <CircleAlertIcon class="size-3.5" />
  {:else if tone === "warning"}
    <TriangleAlertIcon class="size-3.5" />
  {:else if tone === "success"}
    <CircleCheckIcon class="size-3.5" />
  {:else}
    <span class="size-1.5 rounded-full bg-current"></span>
  {/if}
</span>
