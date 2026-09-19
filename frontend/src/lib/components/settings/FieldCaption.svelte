<script lang="ts">
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";

  let {
    label,
    controlID,
    labelID,
    description = "",
    descriptionID,
    error = "",
    errorID,
  }: {
    label: string;
    controlID?: string;
    labelID?: string;
    description?: string;
    descriptionID?: string;
    error?: string;
    errorID?: string;
  } = $props();
</script>

<div class="min-w-0">
  {#if controlID}
    <label
      id={labelID}
      for={controlID}
      class="block cursor-pointer text-[13px] font-medium leading-[1.5] text-foreground [overflow-wrap:anywhere]"
      >{label}</label
    >
  {:else}
    <p
      id={labelID}
      class="text-[13px] font-medium leading-[1.5] text-foreground [overflow-wrap:anywhere]"
    >
      {label}
    </p>
  {/if}
  {#if description}
    <p
      id={descriptionID}
      class="mt-1 max-w-2xl text-xs leading-[1.5] text-muted-foreground [overflow-wrap:anywhere]"
    >
      {description}
    </p>
  {/if}
  {#if error}
    <p
      id={errorID}
      class="mt-1 flex items-start gap-1.5 text-xs leading-[1.5] text-destructive"
    >
      <CircleAlertIcon class="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
      <span class="min-w-0 [overflow-wrap:anywhere]">{error}</span>
    </p>
  {/if}
</div>
