<script lang="ts">
  import CheckIcon from "@lucide/svelte/icons/check";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";

  // Decorative: the owning row announces the save result once, with context.
  let {
    pending = false,
    saved = false,
    failed = false,
  }: {
    pending?: boolean;
    saved?: boolean;
    failed?: boolean;
  } = $props();
</script>

<span
  class="inline-flex size-3.5 shrink-0 items-center justify-center"
  aria-hidden="true"
>
  {#if pending}
    <LoaderCircleIcon
      class="size-3.5 text-muted-foreground motion-safe:animate-spin"
    />
  {:else if failed}
    <CircleAlertIcon class="size-3.5 text-destructive" />
  {:else if saved}
    <CheckIcon class="size-3.5 text-success" />
  {/if}
</span>
