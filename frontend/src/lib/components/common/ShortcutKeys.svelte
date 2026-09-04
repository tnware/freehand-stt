<script lang="ts">
  import * as Kbd from "$lib/components/ui/kbd";
  import { shortcutKeyLabels, shortcutSpokenLabel } from "$lib/utils/shortcuts";

  let {
    value,
    label = "Keyboard shortcut",
    emptyLabel = "Not configured",
  }: {
    value: string;
    label?: string;
    emptyLabel?: string;
  } = $props();

  const keys = $derived(shortcutKeyLabels(value));
  const spoken = $derived(shortcutSpokenLabel(value));
</script>

{#if keys.length > 0}
  <Kbd.Group aria-label={`${label}: ${spoken}`}>
    {#each keys as key, index (`${key}-${index}`)}
      <Kbd.Root>{key}</Kbd.Root>
      {#if index < keys.length - 1}
        <span aria-hidden="true" class="text-[10px] text-muted-foreground">+</span>
      {/if}
    {/each}
  </Kbd.Group>
{:else}
  <span class="text-xs text-muted-foreground">{emptyLabel}</span>
{/if}
